package main

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// ========== Configuration ==========

//go:embed web
var embeddedWeb embed.FS

const (
	stateFile  = "storage/state.json"
	uploadDir  = "storage/uploads"
	maxMsgSize = 10 * 1024 * 1024 // 10 MB
)

var defaultRows = []TierRow{
	{ID: genID(), Label: "夯", Color: "#FF7F7F", Images: []ImageItem{}},
	{ID: genID(), Label: "顶级", Color: "#FFBF7F", Images: []ImageItem{}},
	{ID: genID(), Label: "人上人", Color: "#FFDF7F", Images: []ImageItem{}},
	{ID: genID(), Label: "NPC", Color: "#FFFF7F", Images: []ImageItem{}},
	{ID: genID(), Label: "拉完了", Color: "#BFFF7F", Images: []ImageItem{}},
}

// ========== Data Types ==========

type TierList struct {
	Title         string      `json:"title"`
	BgColor       string      `json:"bg_color"`
	Rows          []TierRow   `json:"rows"`
	StagingImages []ImageItem `json:"staging_images"`
}

type TierRow struct {
	ID     string      `json:"id"`
	Label  string      `json:"label"`
	Color  string      `json:"color"`
	Images []ImageItem `json:"images"`
}

type ImageItem struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

// ========== WebSocket Hub ==========

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	state      *TierList
	mu         sync.RWMutex
}

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		state:      &TierList{},
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("客户端已连接，当前在线：%d", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			log.Printf("客户端已断开，当前在线：%d", len(h.clients))

		case msg := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					delete(h.clients, client)
					close(client.send)
				}
			}
		}
	}
}

// ========== State Persistence ==========

func (h *Hub) loadState() {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("创建上传目录失败：%v", err)
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		log.Println("未找到状态文件，正在初始化默认配置。")
		h.initDefaults()
		return
	}

	var tl TierList
	if err := json.Unmarshal(data, &tl); err != nil {
		log.Printf("状态文件解析失败，正在重新初始化：%v", err)
		h.initDefaults()
		return
	}

	h.state = &tl
	if h.state.Rows == nil {
		h.state.Rows = []TierRow{}
	}
	if h.state.StagingImages == nil {
		h.state.StagingImages = []ImageItem{}
	}
	if h.state.BgColor == "" {
		h.state.BgColor = "#1a1a1a"
	}

	h.cleanupOrphans()
	log.Printf("状态已加载：%d 行，%d 张暂存图片", len(h.state.Rows), len(h.state.StagingImages))
}

func (h *Hub) initDefaults() {
	rows := make([]TierRow, len(defaultRows))
	for i, r := range defaultRows {
		rows[i] = TierRow{
			ID:     r.ID,
			Label:  r.Label,
			Color:  r.Color,
			Images: []ImageItem{},
		}
	}
	h.state = &TierList{
		Title:         "从夯到拉锐评",
		BgColor:       "#1a1a1a",
		Rows:          rows,
		StagingImages: []ImageItem{},
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

	tmpPath := stateFile + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		log.Printf("状态临时文件写入失败：%v", err)
		return
	}
	if err := os.Rename(tmpPath, stateFile); err != nil {
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

	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !refFiles[entry.Name()] {
			path := filepath.Join(uploadDir, entry.Name())
			if err := os.Remove(path); err != nil {
				log.Printf("清理孤立文件失败 %s：%v", entry.Name(), err)
			} else {
				log.Printf("已清理孤立文件：%s", entry.Name())
			}
		}
	}
}

// ========== Helpers ==========

func genID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func fileExt(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp":
		return ext
	}
	return ".png"
}

// ========== Message Types ==========

type Message struct {
	Type string `json:"type"`

	// set_title / title_updated
	Title string `json:"title,omitempty"`

	// set_bg_color / bg_color_updated
	Color string `json:"color,omitempty"`

	// update_label / label_updated
	RowID string `json:"row_id,omitempty"`
	Label string `json:"label,omitempty"`

	// add_row
	Position        string `json:"position,omitempty"`
	RelativeToRowID string `json:"relative_to_row_id,omitempty"`

	// row_added
	Row  *TierRow  `json:"row,omitempty"`
	Rows []TierRow `json:"rows,omitempty"`

	// row_deleted / row_cleared
	StagingImages []ImageItem `json:"staging_images,omitempty"`

	// move_row
	Direction string `json:"direction,omitempty"`

	// upload_image
	Filename string `json:"filename,omitempty"`
	Data     string `json:"data,omitempty"`

	// image_uploaded
	Image *ImageItem `json:"image,omitempty"`

	// move_image
	ImageID     string `json:"image_id,omitempty"`
	TargetRowID string `json:"target_row_id,omitempty"`
	TargetIndex int    `json:"position,omitempty"`
}

// fullStateMsg is the flat full_state payload sent to clients on join
type fullStateMsg struct {
	Type          string      `json:"type"`
	Title         string      `json:"title"`
	BgColor       string      `json:"bg_color"`
	Rows          []TierRow   `json:"rows"`
	StagingImages []ImageItem `json:"staging_images"`
}

func (h *Hub) buildFullState() []byte {
	h.mu.RLock()
	msg := fullStateMsg{
		Type:          "full_state",
		Title:         h.state.Title,
		BgColor:       h.state.BgColor,
		Rows:          h.state.Rows,
		StagingImages: h.state.StagingImages,
	}
	h.mu.RUnlock()
	data, _ := json.Marshal(msg)
	return data
}

// ========== Mutation Methods ==========

func (h *Hub) setTitle(title string) []byte {
	h.mu.Lock()
	h.state.Title = title
	h.mu.Unlock()
	h.saveState()
	b, _ := json.Marshal(Message{Type: "title_updated", Title: title})
	return b
}

func (h *Hub) setBgColor(color string) []byte {
	h.mu.Lock()
	h.state.BgColor = color
	h.mu.Unlock()
	h.saveState()
	b, _ := json.Marshal(Message{Type: "bg_color_updated", Color: color})
	return b
}


func (h *Hub) resetState() []byte {
	h.mu.Lock()
	// Delete all image files from disk
	for _, row := range h.state.Rows {
		for _, img := range row.Images {
			os.Remove(filepath.Join(uploadDir, img.Filename))
		}
	}
	for _, img := range h.state.StagingImages {
		os.Remove(filepath.Join(uploadDir, img.Filename))
	}

	// Rebuild default rows with new IDs
	rows := make([]TierRow, len(defaultRows))
	for i, r := range defaultRows {
		rows[i] = TierRow{
			ID:     genID(),
			Label:  r.Label,
			Color:  r.Color,
			Images: []ImageItem{},
		}
	}
	h.state = &TierList{
		Title:         "从夯到拉锐评",
		BgColor:       "#1a1a1a",
		Rows:          rows,
		StagingImages: []ImageItem{},
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
	b, _ := json.Marshal(Message{Type: "label_updated", RowID: rowID, Label: label})
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
	b, _ := json.Marshal(Message{Type: "label_color_updated", RowID: rowID, Color: color})
	return b
}

func (h *Hub) addRow(position, relativeID string) []byte {
	h.mu.Lock()
	newRow := TierRow{
		ID:     genID(),
		Label:  "?",
		Color:  "#858585",
		Images: []ImageItem{},
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
		h.state.Rows = append(h.state.Rows[:idx], append([]TierRow{newRow}, h.state.Rows[idx:]...)...)
	} else {
		h.state.Rows = append(h.state.Rows[:idx+1], append([]TierRow{newRow}, h.state.Rows[idx+1:]...)...)
	}

	rowsCopy := make([]TierRow, len(h.state.Rows))
	copy(rowsCopy, h.state.Rows)
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(Message{Type: "row_added", Row: &newRow, Rows: rowsCopy})
	return b
}

func (h *Hub) deleteRow(rowID string) []byte {
	h.mu.Lock()
	if len(h.state.Rows) <= 1 {
		h.mu.Unlock()
		return nil
	}
	var movedImages []ImageItem
	var newRows []TierRow
	for _, r := range h.state.Rows {
		if r.ID == rowID {
			movedImages = r.Images
			h.state.StagingImages = append(h.state.StagingImages, r.Images...)
		} else {
			newRows = append(newRows, r)
		}
	}
	h.state.Rows = newRows
	stagingCopy := make([]ImageItem, len(movedImages))
	copy(stagingCopy, movedImages)
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(Message{Type: "row_deleted", RowID: rowID, StagingImages: stagingCopy})
	return b
}

func (h *Hub) clearRow(rowID string) []byte {
	h.mu.Lock()
	var movedImages []ImageItem
	for i := range h.state.Rows {
		if h.state.Rows[i].ID == rowID {
			movedImages = make([]ImageItem, len(h.state.Rows[i].Images))
			copy(movedImages, h.state.Rows[i].Images)
			h.state.StagingImages = append(h.state.StagingImages, h.state.Rows[i].Images...)
			h.state.Rows[i].Images = []ImageItem{}
			break
		}
	}
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(Message{Type: "row_cleared", RowID: rowID, StagingImages: movedImages})
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
	rowsCopy := make([]TierRow, len(h.state.Rows))
	copy(rowsCopy, h.state.Rows)
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(Message{Type: "rows_reordered", Rows: rowsCopy})
	return b
}

func (h *Hub) uploadImage(filename, b64data string) []byte {
	raw, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		log.Printf("Base64图片解码失败：%v", err)
		return nil
	}

	ext := fileExt(filename)
	id := genID()
	savedName := id + ext
	savePath := filepath.Join(uploadDir, savedName)

	if err := os.WriteFile(savePath, raw, 0644); err != nil {
		log.Printf("图片文件写入失败：%v", err)
		return nil
	}

	img := ImageItem{
		ID:       id,
		Filename: savedName,
		URL:      "/uploads/" + savedName,
	}

	h.mu.Lock()
	h.state.StagingImages = append(h.state.StagingImages, img)
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(Message{Type: "image_uploaded", Image: &img})
	return b
}

func (h *Hub) moveImage(imageID, targetRowID string, targetIndex int) []byte {
	h.mu.Lock()
	var movedImg ImageItem
	found := false

	// Remove from source (rows or staging)
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

	// Insert at target
	if targetRowID == "" || targetRowID == "null" {
		// Move to staging
		if targetIndex < 0 || targetIndex >= len(h.state.StagingImages) {
			h.state.StagingImages = append(h.state.StagingImages, movedImg)
		} else {
			h.state.StagingImages = append(
				h.state.StagingImages[:targetIndex],
				append([]ImageItem{movedImg}, h.state.StagingImages[targetIndex:]...)...,
			)
		}
	} else {
		for i := range h.state.Rows {
			if h.state.Rows[i].ID == targetRowID {
				if targetIndex < 0 || targetIndex >= len(h.state.Rows[i].Images) {
					h.state.Rows[i].Images = append(h.state.Rows[i].Images, movedImg)
				} else {
					h.state.Rows[i].Images = append(
						h.state.Rows[i].Images[:targetIndex],
						append([]ImageItem{movedImg}, h.state.Rows[i].Images[targetIndex:]...)...,
					)
				}
				break
			}
		}
	}
	h.mu.Unlock()
	h.saveState()

	b, _ := json.Marshal(Message{
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
		os.Remove(filepath.Join(uploadDir, deletedFilename))
		h.saveState()
	}

	b, _ := json.Marshal(Message{Type: "image_deleted", ImageID: imageID})
	return b
}

// ========== Message Router ==========

func (h *Hub) handleMessage(_ *Client, raw []byte) {
	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		log.Printf("消息解析失败：%v", err)
		return
	}

	var broadcast []byte

	switch msg.Type {
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
		broadcast = h.uploadImage(msg.Filename, msg.Data)
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
	case "reset":
			broadcast = h.resetState()
	default:
		log.Printf("未知消息类型：%s", msg.Type)
		return
	}

	if broadcast != nil {
		h.broadcast <- broadcast
	}
}

// ========== WebSocket Handler ==========

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *Hub) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败：%v", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- client

	// Send full state to the new client
	fullState := h.buildFullState()
	if err := conn.WriteMessage(websocket.TextMessage, fullState); err != nil {
		log.Printf("发送完整状态失败：%v", err)
	}

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMsgSize)

	for {
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket错误：%v", err)
			}
			break
		}
		c.hub.handleMessage(c, msg)
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ========== HTTP Handlers ==========

func (h *Hub) serveHome(w http.ResponseWriter, r *http.Request) {
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

// ========== Settings ==========

type ServerSettings struct {
	Port int `json:"port"`
}

func loadPort() string {
	data, err := os.ReadFile("settings.json")
	if err != nil {
		if os.IsNotExist(err) {
			defaultSettings := ServerSettings{Port: 23331}
			if d, e := json.MarshalIndent(defaultSettings, "", "  "); e == nil {
				if e2 := os.WriteFile("settings.json", d, 0644); e2 != nil {
					log.Printf("自动生成 settings.json 失败：%v", e2)
				} else {
					log.Println("已自动生成 settings.json")
				}
			}
			return "23331"
		}
		log.Printf("读取 settings.json 失败：%v，使用默认端口", err)
		return "23331"
	}
	var s ServerSettings
	if json.Unmarshal(data, &s) == nil && s.Port > 0 {
		return fmt.Sprintf("%d", s.Port)
	}
	log.Println("settings.json 格式无效，使用默认端口")
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return "23331"
}

// ========== Main ==========

func main() {
	hub := newHub()
	hub.loadState()
	go hub.run()

	// Static file servers (embedded)
	cssSub, _ := fs.Sub(embeddedWeb, "web/css")
	jsSub, _ := fs.Sub(embeddedWeb, "web/js")
	cssFS := http.FileServer(http.FS(cssSub))
	jsFS := http.FileServer(http.FS(jsSub))
	uploadsFS := http.FileServer(http.Dir(uploadDir))

	mux := http.NewServeMux()
	mux.HandleFunc("/", hub.serveHome)
	mux.Handle("/css/", http.StripPrefix("/css/", cssFS))
	mux.Handle("/js/", http.StripPrefix("/js/", jsFS))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", uploadsFS))
	mux.HandleFunc("/ws", hub.handleWS)

	port := loadPort()
	log.Printf("Tier List 服务器启动于 http://127.0.0.1:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
