package hub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"tlo/auth"
	"tlo/config"
	"tlo/models"
)

func (h *Hub) setTitle(title string) []byte {
	h.mu.Lock()
	h.state.Title = title
	h.mu.Unlock()
	h.saveState()
	b, _ := json.Marshal(models.Message{Type: "title_updated", Title: title})
	return b
}

func (h *Hub) setBgColor(color string) []byte {
	h.mu.Lock()
	h.state.BgColor = color
	h.mu.Unlock()
	h.saveState()
	b, _ := json.Marshal(models.Message{Type: "bg_color_updated", Color: color})
	return b
}

func (h *Hub) resetState() []byte {
	h.mu.Lock()
	for _, row := range h.state.Rows {
		for _, img := range row.Images {
			os.Remove(filepath.Join(config.UploadDir, img.Filename))
		}
	}
	for _, img := range h.state.StagingImages {
		os.Remove(filepath.Join(config.UploadDir, img.Filename))
	}

	rows := make([]models.TierRow, len(config.DefaultRows))
	for i, r := range config.DefaultRows {
		rows[i] = models.TierRow{
			ID:     config.GenID(),
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
	h.mu.Unlock()
	h.saveState()
	return h.buildFullState()
}

func (h *Hub) updateLabel(rowID, label string) []byte {
	h.mu.Lock()
	for i := range h.state.Rows {
		if h.state.Rows[i].ID == rowID {
			h.state.Rows[i].Label = label
			break
		}
	}
	h.mu.Unlock()
	h.saveState()
	b, _ := json.Marshal(models.Message{Type: "label_updated", RowID: rowID, Label: label})
	return b
}

func (h *Hub) updateLabelColor(rowID, color string) []byte {
	h.mu.Lock()
	for i := range h.state.Rows {
		if h.state.Rows[i].ID == rowID {
			h.state.Rows[i].Color = color
			break
		}
	}
	h.mu.Unlock()
	h.saveState()
	b, _ := json.Marshal(models.Message{Type: "label_color_updated", RowID: rowID, Color: color})
	return b
}

func (h *Hub) addRow(position, relativeID string) []byte {
	h.mu.Lock()
	newRow := models.TierRow{
		ID:     config.GenID(),
		Label:  "?",
		Color:  "#858585",
		Images: []models.ImageItem{},
	}

	idx := -1
	for i, r := range h.state.Rows {
		if r.ID == relativeID {
			idx = i
			break
		}
	}
	if idx == -1 {
		h.state.Rows = append(h.state.Rows, newRow)
	} else if position == "above" {
		h.state.Rows = append(h.state.Rows[:idx], append([]models.TierRow{newRow}, h.state.Rows[idx:]...)...)
	} else {
		h.state.Rows = append(h.state.Rows[:idx+1], append([]models.TierRow{newRow}, h.state.Rows[idx+1:]...)...)
	}

	rowsCopy := make([]models.TierRow, len(h.state.Rows))
	copy(rowsCopy, h.state.Rows)
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(models.Message{Type: "row_added", Row: &newRow, Rows: rowsCopy})
	return b
}

func (h *Hub) deleteRow(rowID string) []byte {
	h.mu.Lock()
	if len(h.state.Rows) <= 1 {
		h.mu.Unlock()
		return nil
	}
	var movedImages []models.ImageItem
	var newRows []models.TierRow
	for _, r := range h.state.Rows {
		if r.ID == rowID {
			movedImages = r.Images
			h.state.StagingImages = append(h.state.StagingImages, r.Images...)
		} else {
			newRows = append(newRows, r)
		}
	}
	h.state.Rows = newRows
	stagingCopy := make([]models.ImageItem, len(movedImages))
	copy(stagingCopy, movedImages)
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(models.Message{Type: "row_deleted", RowID: rowID, StagingImages: stagingCopy})
	return b
}

func (h *Hub) clearRow(rowID string) []byte {
	h.mu.Lock()
	var movedImages []models.ImageItem
	for i := range h.state.Rows {
		if h.state.Rows[i].ID == rowID {
			movedImages = make([]models.ImageItem, len(h.state.Rows[i].Images))
			copy(movedImages, h.state.Rows[i].Images)
			h.state.StagingImages = append(h.state.StagingImages, h.state.Rows[i].Images...)
			h.state.Rows[i].Images = []models.ImageItem{}
			break
		}
	}
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(models.Message{Type: "row_cleared", RowID: rowID, StagingImages: movedImages})
	return b
}

func (h *Hub) moveRow(rowID, direction string) []byte {
	h.mu.Lock()
	idx := -1
	for i, r := range h.state.Rows {
		if r.ID == rowID {
			idx = i
			break
		}
	}
	if idx >= 0 {
		if direction == "up" && idx > 0 {
			h.state.Rows[idx], h.state.Rows[idx-1] = h.state.Rows[idx-1], h.state.Rows[idx]
		} else if direction == "down" && idx < len(h.state.Rows)-1 {
			h.state.Rows[idx], h.state.Rows[idx+1] = h.state.Rows[idx+1], h.state.Rows[idx]
		}
	}
	rowsCopy := make([]models.TierRow, len(h.state.Rows))
	copy(rowsCopy, h.state.Rows)
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(models.Message{Type: "rows_reordered", Rows: rowsCopy})
	return b
}

func (h *Hub) uploadImage(filename, b64data string) ([]byte, []byte) {
	raw, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		log.Printf("Base64图片解码失败：%v", err)
		r, _ := json.Marshal(models.Message{Type: "upload_rejected", Error: "图片数据解码失败"})
		return nil, r
	}

	// Check file size
	maxSize := config.GetMaxUploadSizeMB()
	if maxSize > 0 && len(raw) > maxSize*1024*1024 {
		r, _ := json.Marshal(models.Message{Type: "upload_rejected", Error: fmt.Sprintf("文件大小超过限制（最大 %d MB）", maxSize)})
		return nil, r
	}

	ext := config.FileExt(filename)
	id := config.GenID()
	savedName := id + ext
	savePath := filepath.Join(config.UploadDir, savedName)

	if err := os.WriteFile(savePath, raw, 0644); err != nil {
		log.Printf("图片文件写入失败：%v", err)
		r, _ := json.Marshal(models.Message{Type: "upload_rejected", Error: "服务器写入文件失败，请重试"})
		return nil, r
	}

	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	baseName = strings.TrimSpace(baseName)
	if len(baseName) > 128 {
		baseName = baseName[:128]
	}

	img := models.ImageItem{
		ID:       id,
		Name:     baseName,
		Filename: savedName,
		URL:      "/uploads/" + savedName,
	}

	h.mu.Lock()
	h.state.StagingImages = append(h.state.StagingImages, img)
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(models.Message{Type: "image_uploaded", Image: &img})
	return b, nil
}

func (h *Hub) moveImage(imageID, targetRowID string, targetIndex int) []byte {
	h.mu.Lock()
	var movedImg models.ImageItem
	found := false

	for i := range h.state.Rows {
		for j, img := range h.state.Rows[i].Images {
			if img.ID == imageID {
				movedImg = img
				h.state.Rows[i].Images = append(h.state.Rows[i].Images[:j], h.state.Rows[i].Images[j+1:]...)
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		for j, img := range h.state.StagingImages {
			if img.ID == imageID {
				movedImg = img
				h.state.StagingImages = append(h.state.StagingImages[:j], h.state.StagingImages[j+1:]...)
				found = true
				break
			}
		}
	}

	if !found {
		h.mu.Unlock()
		return nil
	}

	// Always append to end
	if targetRowID == "" || targetRowID == "null" {
		h.state.StagingImages = append(h.state.StagingImages, movedImg)
	} else {
		for i := range h.state.Rows {
			if h.state.Rows[i].ID == targetRowID {
				h.state.Rows[i].Images = append(h.state.Rows[i].Images, movedImg)
				break
			}
		}
	}
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(models.Message{
		Type:        "image_moved",
		ImageID:     imageID,
		TargetRowID: targetRowID,
		TargetIndex: targetIndex,
	})
	return b
}

func (h *Hub) deleteImage(imageID string) []byte {
	h.mu.Lock()
	var deletedFilename string
	found := false

	for i := range h.state.Rows {
		for j, img := range h.state.Rows[i].Images {
			if img.ID == imageID {
				deletedFilename = img.Filename
				h.state.Rows[i].Images = append(h.state.Rows[i].Images[:j], h.state.Rows[i].Images[j+1:]...)
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		for j, img := range h.state.StagingImages {
			if img.ID == imageID {
				deletedFilename = img.Filename
				h.state.StagingImages = append(h.state.StagingImages[:j], h.state.StagingImages[j+1:]...)
				found = true
				break
			}
		}
	}
	h.mu.Unlock()

	if found {
		os.Remove(filepath.Join(config.UploadDir, deletedFilename))
		h.saveState()
	}

	b, _ := json.Marshal(models.Message{Type: "image_deleted", ImageID: imageID})
	return b
}

func (h *Hub) stageAll() []byte {
	h.mu.Lock()
	for i := range h.state.Rows {
		h.state.StagingImages = append(h.state.StagingImages, h.state.Rows[i].Images...)
		h.state.Rows[i].Images = []models.ImageItem{}
	}
	stagingCopy := make([]models.ImageItem, len(h.state.StagingImages))
	copy(stagingCopy, h.state.StagingImages)
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(models.Message{Type: "all_staged", StagingImages: stagingCopy})
	return b
}

func (h *Hub) applyColorSequence(rowID string) []byte {
	h.mu.Lock()
	startIdx := -1
	for i, r := range h.state.Rows {
		if r.ID == rowID {
			startIdx = i
			break
		}
	}
	if startIdx == -1 || startIdx >= len(h.state.Rows)-1 {
		h.mu.Unlock()
		return nil
	}

	currentColor := h.state.Rows[startIdx].Color
	paletteIdx := -1
	for i, c := range config.ColorPalette {
		if c == currentColor {
			paletteIdx = i
			break
		}
	}

	for i := startIdx + 1; i < len(h.state.Rows); i++ {
		paletteIdx++
		if paletteIdx >= len(config.ColorPalette) {
			paletteIdx = 0
		}
		h.state.Rows[i].Color = config.ColorPalette[paletteIdx]
	}

	rowsCopy := make([]models.TierRow, len(h.state.Rows))
	copy(rowsCopy, h.state.Rows)
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(models.Message{Type: "color_sequence_applied", Rows: rowsCopy})
	return b
}

func (h *Hub) toggleImageFit(imageID string) []byte {
	h.mu.Lock()
	found := false
	var newVal bool

	for i := range h.state.Rows {
		for j := range h.state.Rows[i].Images {
			if h.state.Rows[i].Images[j].ID == imageID {
				h.state.Rows[i].Images[j].FitWidth = !h.state.Rows[i].Images[j].FitWidth
				newVal = h.state.Rows[i].Images[j].FitWidth
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		for i := range h.state.StagingImages {
			if h.state.StagingImages[i].ID == imageID {
				h.state.StagingImages[i].FitWidth = !h.state.StagingImages[i].FitWidth
				newVal = h.state.StagingImages[i].FitWidth
				found = true
				break
			}
		}
	}
	h.mu.Unlock()

	if !found {
		return nil
	}
	h.saveState()

	b, _ := json.Marshal(models.Message{
		Type:     "image_fit_toggled",
		ImageID:  imageID,
		FitWidth: newVal,
	})
	return b
}

func (h *Hub) renameImage(imageID, imageName string) ([]byte, []byte) {
	trimmed := strings.TrimSpace(imageName)
	truncated := false
	if len(trimmed) > 128 {
		trimmed = trimmed[:128]
		truncated = true
	}

	h.mu.Lock()
	found := false
	for i := range h.state.Rows {
		for j := range h.state.Rows[i].Images {
			if h.state.Rows[i].Images[j].ID == imageID {
				h.state.Rows[i].Images[j].Name = trimmed
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		for i := range h.state.StagingImages {
			if h.state.StagingImages[i].ID == imageID {
				h.state.StagingImages[i].Name = trimmed
				found = true
				break
			}
		}
	}
	h.mu.Unlock()

	if !found {
		return nil, nil
	}
	h.saveState()

	b, _ := json.Marshal(models.Message{
		Type:      "image_renamed",
		ImageID:   imageID,
		ImageName: trimmed,
	})

	var directResp []byte
	if truncated {
		directResp, _ = json.Marshal(models.Message{Type: "image_name_truncated", Error: "图片名称已截断至128个字符"})
	}
	return b, directResp
}

func (h *Hub) listPresets() []byte {
	names := listPresetNames()
	if names == nil {
		names = []string{}
	}
	b, _ := json.Marshal(models.Message{Type: "presets_list", Presets: names})
	return b
}

func (h *Hub) savePreset(name string) []byte {
	if err := h.savePresetToDisk(name); err != nil {
		if os.IsExist(err) {
			b, _ := json.Marshal(models.Message{Type: "preset_saved", Success: false, Error: "预设名称已存在"})
			return b
		}
		b, _ := json.Marshal(models.Message{Type: "preset_saved", Success: false, Error: "预设保存失败"})
		return b
	}
	log.Printf("预设 %s 已保存", name)
	b, _ := json.Marshal(models.Message{Type: "preset_saved", Success: true, PresetName: name})
	return b
}

func (h *Hub) loadPreset(name string) []byte {
	if name == "" {
		return h.resetState()
	}
	if err := h.loadPresetFromDisk(name); err != nil {
		log.Printf("加载预设 %s 失败：%v", name, err)
		return nil
	}
	h.mu.Lock()
	h.state = &models.TierList{}
	h.mu.Unlock()
	h.LoadState()
	log.Printf("预设 %s 已加载", name)
	return h.buildFullState()
}

var alwaysAllowed = map[string]bool{
	"change_display_name": true,
}

func (h *Hub) handleMessage(client *Client, raw []byte) {
	var msg models.Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("消息解析失败：%v", err)
		return
	}

	if !alwaysAllowed[msg.Type] && !config.IsWhitelistIP(client.IP) && !client.HasPermission(msg.Type) {
		log.Printf("权限不足：用户 %s（%s） 的 %s 操作被拒绝", client.DisplayName, client.IP, msg.Type)
		reject, _ := json.Marshal(models.Message{Type: "action_rejected", Error: "您没有权限执行此操作"})
		select {
		case client.send <- reject:
		default:
		}
		select {
		case client.send <- h.buildFullState():
		default:
		}
		return
	}

	var broadcast []byte

	switch msg.Type {
	case "change_display_name":
		if msg.NewDisplayName == "" {
			return
		}
		if client.Username != "" {
			// Logged-in user: update profile.json
			if err := auth.UpdateDisplayName(config.UsersDir, client.Username, msg.NewDisplayName); err != nil {
				log.Printf("更新显示名称失败：%v", err)
				reject, _ := json.Marshal(models.Message{Type: "action_rejected", Error: "更新显示名称失败"})
				select {
				case client.send <- reject:
				default:
				}
				return
			}
			// Update session guest_name to match
			h.sm.UpdateGuestName(client.SessionID, msg.NewDisplayName)
		} else {
			// Guest: update session guest_name
			if err := h.sm.UpdateGuestName(client.SessionID, msg.NewDisplayName); err != nil {
				log.Printf("更新游客名称失败：%v", err)
				return
			}
		}
		client.DisplayName = msg.NewDisplayName

		// Update onlineUsers entry
		h.onlineMu.Lock()
		if entry, ok := h.onlineUsers[client.ConnID]; ok {
			entry.DisplayName = msg.NewDisplayName
			if client.Username != "" {
				entry.PermissionGroup = client.PermissionGroup
			}
			h.onlineUsers[client.ConnID] = entry
		}
		h.onlineMu.Unlock()

		// Re-broadcast online users
		h.broadcastOnlineUsers()

		// Send updated user_info to the client
		userInfo := h.buildUserInfoMsg(client)
		select {
		case client.send <- userInfo:
		default:
		}
		return

	case "set_title":
		broadcast = h.setTitle(msg.Title)
	case "update_label":
		broadcast = h.updateLabel(msg.RowID, msg.Label)
	case "update_label_color":
		broadcast = h.updateLabelColor(msg.RowID, msg.Color)
	case "add_row":
		broadcast = h.addRow(msg.Position, msg.RelativeToRowID)
	case "delete_row":
		broadcast = h.deleteRow(msg.RowID)
	case "clear_row":
		broadcast = h.clearRow(msg.RowID)
	case "move_row":
		broadcast = h.moveRow(msg.RowID, msg.Direction)
	case "upload_image":
		var reject []byte
		broadcast, reject = h.uploadImage(msg.Filename, msg.Data)
		if reject != nil {
			select {
			case client.send <- reject:
			default:
			}
			return
		}
		if broadcast == nil {
			return
		}
	case "move_image":
		broadcast = h.moveImage(msg.ImageID, msg.TargetRowID, msg.TargetIndex)
		if broadcast == nil {
			return
		}
	case "delete_image":
		broadcast = h.deleteImage(msg.ImageID)
	case "stage_all":
		broadcast = h.stageAll()
	case "apply_color_sequence":
		broadcast = h.applyColorSequence(msg.RowID)
	case "toggle_image_fit":
		broadcast = h.toggleImageFit(msg.ImageID)
	case "rename_image":
		var directResp []byte
		broadcast, directResp = h.renameImage(msg.ImageID, msg.ImageName)
		if directResp != nil {
			select {
			case client.send <- directResp:
			default:
			}
		}
		if broadcast == nil {
			return
		}
	case "list_presets":
		resp := h.listPresets()
		select {
		case client.send <- resp:
		default:
		}
		return
	case "save_preset":
		resp := h.savePreset(msg.PresetName)
		select {
		case client.send <- resp:
		default:
		}
		return
	case "load_preset":
		broadcast = h.loadPreset(msg.PresetName)
		if broadcast == nil {
			reject, _ := json.Marshal(models.Message{Type: "action_rejected", Error: "预设加载失败，预设可能已被删除"})
			select {
			case client.send <- reject:
			default:
			}
			return
		}
	default:
		log.Printf("未知消息类型：%s", msg.Type)
		return
	}

	if broadcast != nil {
		h.broadcast <- broadcast
	}
}
