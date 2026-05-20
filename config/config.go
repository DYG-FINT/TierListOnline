package config

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"
	"sync"

	"tlo/models"
)

const (
	StateFile  = "storage/state.json"
	UploadDir  = "storage/uploads"
	MaxMsgSize = 10 * 1024 * 1024 // 10 MB

	ModeFree = "free"
	ModeSort = "sort"
)

var (
	cfgMu             sync.RWMutex
	maxUploadSizeMB   int      = 10
	allowedExtensions []string
	serverMode        string = ModeFree
	whitelistIPs      []string
)

func GetMaxUploadSizeMB() int {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return maxUploadSizeMB
}

func GetAllowedExtensions() []string {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return allowedExtensions
}

func GetServerMode() string {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	return serverMode
}

func IsWhitelistIP(ip string) bool {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	for _, w := range whitelistIPs {
		if w == ip {
			return true
		}
	}
	return false
}

func ApplySettings(maxSize int, extensions []string, mode string, ips []string) {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	if maxSize > 0 {
		maxUploadSizeMB = maxSize
	}
	if len(extensions) > 0 {
		allowedExtensions = extensions
	}
	if mode == ModeFree || mode == ModeSort {
		serverMode = mode
	}
	whitelistIPs = ips
}

var ColorPalette = []string{
	"#FF7F7F",
	"#FFBF7F",
	"#FFDF7F",
	"#FFFF7F",
	"#BFFF7F",
	"#7FFF7F",
	"#7FFFFF",
	"#7FBFFF",
	"#7F7FFF",
	"#FF7FFF",
	"#BF7FBF",
	"#3B3B3B",
	"#858585",
	"#CFCFCF",
	"#F7F7F7",
}

var DefaultRows = []models.TierRow{
	{ID: GenID(), Label: "夯", Color: "#FF7F7F", Images: []models.ImageItem{}},
	{ID: GenID(), Label: "顶级", Color: "#FFBF7F", Images: []models.ImageItem{}},
	{ID: GenID(), Label: "人上人", Color: "#FFDF7F", Images: []models.ImageItem{}},
	{ID: GenID(), Label: "NPC", Color: "#FFFF7F", Images: []models.ImageItem{}},
	{ID: GenID(), Label: "拉完了", Color: "#BFFF7F", Images: []models.ImageItem{}},
}

func GenID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func FileExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range GetAllowedExtensions() {
		if ext == allowed {
			return ext
		}
	}
	return ".png"
}
