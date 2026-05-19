package config

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"strings"

	"tlo/models"
)

const (
	StateFile  = "storage/state.json"
	UploadDir  = "storage/uploads"
	MaxMsgSize = 10 * 1024 * 1024 // 10 MB
)

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
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return ext
	}
	return ".png"
}
