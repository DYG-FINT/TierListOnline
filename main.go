package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"tlo/config"
	"tlo/hub"
)

//go:embed web
var embeddedWeb embed.FS

func main() {
	h := hub.NewHub()
	h.LoadState()
	go h.Run()

	cssSub, _ := fs.Sub(embeddedWeb, "web/css")
	jsSub, _ := fs.Sub(embeddedWeb, "web/js")
	cssFS := http.FileServer(http.FS(cssSub))
	jsFS := http.FileServer(http.FS(jsSub))
	uploadsFS := http.FileServer(http.Dir("storage/uploads"))

	mux := http.NewServeMux()
	mux.HandleFunc("/", serveHome)
	mux.Handle("/css/", http.StripPrefix("/css/", cssFS))
	mux.Handle("/js/", http.StripPrefix("/js/", jsFS))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", uploadsFS))
	mux.HandleFunc("/ws", h.HandleWS)

	port := loadSettings()
	log.Printf("Tier List 服务器启动于 http://127.0.0.1:%s", port)
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
	Port            int `json:"port"`
	MaxUploadSizeMB int `json:"max_upload_size_mb"`
	MaxImageWidth   int `json:"max_image_width"`
	MaxImageHeight  int `json:"max_image_height"`
}

func loadSettings() string {
	defaultSettings := serverSettings{
		Port:            23331,
		MaxUploadSizeMB: 10,
		MaxImageWidth:   4096,
		MaxImageHeight:  4096,
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
			config.MaxUploadSizeMB = defaultSettings.MaxUploadSizeMB
			config.MaxImageWidth = defaultSettings.MaxImageWidth
			config.MaxImageHeight = defaultSettings.MaxImageHeight
			return fmt.Sprintf("%d", defaultSettings.Port)
		}
		log.Printf("读取 settings.json 失败：%v，使用默认配置", err)
		config.MaxUploadSizeMB = defaultSettings.MaxUploadSizeMB
		config.MaxImageWidth = defaultSettings.MaxImageWidth
		config.MaxImageHeight = defaultSettings.MaxImageHeight
		return fmt.Sprintf("%d", defaultSettings.Port)
	}

	var s serverSettings
	if e := json.Unmarshal(data, &s); e != nil || s.Port <= 0 {
		log.Println("settings.json 格式无效，使用默认配置")
		config.MaxUploadSizeMB = defaultSettings.MaxUploadSizeMB
		config.MaxImageWidth = defaultSettings.MaxImageWidth
		config.MaxImageHeight = defaultSettings.MaxImageHeight
		if port := os.Getenv("PORT"); port != "" {
			return port
		}
		return fmt.Sprintf("%d", defaultSettings.Port)
	}

	// Apply settings
	config.MaxUploadSizeMB = s.MaxUploadSizeMB
	if config.MaxUploadSizeMB <= 0 {
		config.MaxUploadSizeMB = defaultSettings.MaxUploadSizeMB
	}
	config.MaxImageWidth = s.MaxImageWidth
	config.MaxImageHeight = s.MaxImageHeight

	return fmt.Sprintf("%d", s.Port)
}
