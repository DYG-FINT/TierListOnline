package hub

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"

	"tlo/config"
	"tlo/models"
)

func (h *Hub) LoadState() {
	if err := os.MkdirAll(config.UploadDir, 0755); err != nil {
		log.Fatalf("创建上传目录失败：%v", err)
	}

	data, err := os.ReadFile(config.StateFile)
	if err != nil {
		log.Println("未找到状态文件，正在初始化默认配置。")
		h.initDefaults()
		return
	}

	var tl models.TierList
	if err := json.Unmarshal(data, &tl); err != nil {
		log.Printf("状态文件解析失败，正在重新初始化：%v", err)
		h.initDefaults()
		return
	}

	h.state = &tl
	if h.state.Rows == nil {
		h.state.Rows = []models.TierRow{}
	}
	if h.state.StagingImages == nil {
		h.state.StagingImages = []models.ImageItem{}
	}
	if h.state.BgColor == "" {
		h.state.BgColor = "#1a1a1a"
	}

	h.cleanupOrphans()
	log.Printf("状态已加载：%d 行，%d 张暂存图片", len(h.state.Rows), len(h.state.StagingImages))
}

func (h *Hub) initDefaults() {
	rows := make([]models.TierRow, len(config.DefaultRows))
	for i, r := range config.DefaultRows {
		rows[i] = models.TierRow{
			ID:     r.ID,
			Label:  r.Label,
			Color:  r.Color,
			Images: []models.ImageItem{},
		}
	}
	h.state = &models.TierList{
		Title:         "从夯到拉锐评",
		BgColor:       "#1a1a1a",
		Rows:          rows,
		StagingImages: []models.ImageItem{},
	}
	h.saveState()
}

func (h *Hub) saveState() {
	h.mu.RLock()
	data, err := json.MarshalIndent(h.state, "", "  ")
	h.mu.RUnlock()
	if err != nil {
		log.Printf("状态序列化失败：%v", err)
		return
	}

	tmpPath := config.StateFile + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		log.Printf("状态临时文件写入失败：%v", err)
		return
	}
	if err := os.Rename(tmpPath, config.StateFile); err != nil {
		log.Printf("状态文件重命名失败：%v", err)
	}
}

func (h *Hub) cleanupOrphans() {
	refFiles := make(map[string]bool)
	for _, row := range h.state.Rows {
		for _, img := range row.Images {
			refFiles[img.Filename] = true
		}
	}
	for _, img := range h.state.StagingImages {
		refFiles[img.Filename] = true
	}

	entries, err := os.ReadDir(config.UploadDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !refFiles[entry.Name()] {
			path := filepath.Join(config.UploadDir, entry.Name())
			if err := os.Remove(path); err != nil {
				log.Printf("清理孤立文件失败 %s：%v", entry.Name(), err)
			} else {
				log.Printf("已清理孤立文件：%s", entry.Name())
			}
		}
	}
}
