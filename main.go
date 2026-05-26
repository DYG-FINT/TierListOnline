package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"tlo/config"
	"tlo/hub"
)

//go:embed web
var embeddedWeb embed.FS

var Version = "dev"

func main() {
	h := hub.NewHub()
	h.LoadState()
	go h.Run()

	cssSub, _ := fs.Sub(embeddedWeb, "web/css")
	jsSub, _ := fs.Sub(embeddedWeb, "web/js")
	cssFS := http.FileServer(http.FS(cssSub))
	jsFS := http.FileServer(http.FS(jsSub))
	uploadsFS := http.FileServer(http.Dir(config.UploadDir))

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveHome)
	mux.Handle("/css/", http.StripPrefix("/css/", cssFS))
	mux.Handle("/js/", http.StripPrefix("/js/", jsFS))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", uploadsFS))
	mux.HandleFunc("/ws", h.HandleWS)

	port := loadSettings()
	go watchSettings()
	log.Printf("Tier List 服务器启动于 http://127.0.0.1:%s", port)
	log.Printf("服务器版本：%s", Version)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := embeddedWeb.ReadFile("web/Home.html")
	if err != nil {
		http.Error(w, "页面未找到", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

type serverSettings struct {
	Port              int                          `json:"port"`
	MaxUploadSizeMB   int                          `json:"max_upload_size_mb"`
	AllowedExtensions []string                     `json:"allowed_extensions"`
	PermissionPresets map[string]map[string]bool   `json:"permission_presets"`
	Permissions       map[string]bool              `json:"permissions"`
	WhitelistIPs      []string                     `json:"whitelist_ips"`
}

func loadSettings() string {
	defaultSettings := serverSettings{
		Port:            23331,
		MaxUploadSizeMB: 10,
		AllowedExtensions: []string{
			".png", ".jpg", ".jpeg", ".gif", ".webp",
			".ico", ".bmp", ".svg", ".svgz",
			".tiff", ".tif",
			".avif", ".heic", ".heif",
			".jp2", ".jpx", ".j2k", ".jxl",
		},
		PermissionPresets: config.DefaultPermissionPresets,
		Permissions:       config.DefaultPermissionsUsage,
		WhitelistIPs:      []string{"127.0.0.1"},
	}

	data, err := os.ReadFile("settings.json")
	if err != nil {
		if os.IsNotExist(err) {
			if d, e := json.MarshalIndent(defaultSettings, "", "  "); e == nil {
				if e2 := os.WriteFile("settings.json", d, 0644); e2 != nil {
					log.Printf("自动生成 settings.json 失败：%v", e2)
				} else {
					log.Println("已自动生成 settings.json")
				}
			}
			applySettings(defaultSettings)
			return fmt.Sprintf("%d", defaultSettings.Port)
		}
		log.Printf("读取 settings.json 失败：%v，使用默认配置", err)
		applySettings(defaultSettings)
		return fmt.Sprintf("%d", defaultSettings.Port)
	}

	var s serverSettings
	if e := json.Unmarshal(data, &s); e != nil || s.Port <= 0 {
		log.Println("settings.json 格式无效，使用默认配置")
		applySettings(defaultSettings)
		if port := os.Getenv("PORT"); port != "" {
			return port
		}
		return fmt.Sprintf("%d", defaultSettings.Port)
	}

	applySettings(s)
	return fmt.Sprintf("%d", s.Port)
}

func applySettings(s serverSettings) {
	config.ApplySettings(s.MaxUploadSizeMB, s.AllowedExtensions, s.Permissions, s.PermissionPresets, s.WhitelistIPs)
}

func watchSettings() {
	var lastMod time.Time
	if info, err := os.Stat("settings.json"); err == nil {
		lastMod = info.ModTime()
	}

	for {
		time.Sleep(2 * time.Second)
		info, err := os.Stat("settings.json")
		if err != nil {
			continue
		}
		if !info.ModTime().After(lastMod) {
			continue
		}
		lastMod = info.ModTime()

		data, err := os.ReadFile("settings.json")
		if err != nil {
			log.Printf("热重载 settings.json 失败（读取）：%v", err)
			continue
		}

		var s serverSettings
		if err := json.Unmarshal(data, &s); err != nil {
			log.Printf("热重载 settings.json 失败（解析）：%v", err)
			continue
		}

		applySettings(s)
		log.Printf("settings.json 已热重载（白名单IP数：%d）", len(s.WhitelistIPs))
	}
}
