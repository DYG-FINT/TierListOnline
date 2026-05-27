package hub

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"tlo/auth"
	"tlo/config"
	"tlo/models"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	state      *models.TierList
	mu         sync.RWMutex

	sm          *auth.SessionManager
	onlineUsers map[string]onlineUserEntry
	onlineMu    sync.RWMutex
}

type onlineUserEntry struct {
	DisplayName     string
	PermissionGroup string
	Username        string // empty for guests
}

type Client struct {
	hub             *Hub
	conn            *websocket.Conn
	send            chan []byte
	IP              string
	ConnID          string // unique per connection
	SessionID       string
	Username        string
	DisplayName     string
	PermissionGroup string
	Permissions     map[string]bool
}

func NewHub(sm *auth.SessionManager) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		broadcast:   make(chan []byte, 256),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		state:       &models.TierList{},
		sm:          sm,
		onlineUsers: make(map[string]onlineUserEntry),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("客户端 %s 已连接，当前会话数：%d", client.IP, len(h.clients))

			h.onlineMu.Lock()
			h.onlineUsers[client.ConnID] = onlineUserEntry{
				DisplayName:     client.DisplayName,
				PermissionGroup: client.PermissionGroup,
				Username:        client.Username,
			}
			h.onlineMu.Unlock()

			h.broadcastOnlineUsers()

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			log.Printf("客户端 %s 已断开，当前会话数：%d", client.IP, len(h.clients))

			h.onlineMu.Lock()
			delete(h.onlineUsers, client.ConnID)
			h.onlineMu.Unlock()

			h.broadcastOnlineUsers()

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

func (h *Hub) buildFullState() []byte {
	h.mu.RLock()
	msg := models.FullStateMsg{
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

func (h *Hub) buildOnlineUsersMsg() []byte {
	h.onlineMu.RLock()
	defer h.onlineMu.RUnlock()

	users := make([]models.OnlineUser, 0, len(h.onlineUsers))
	for _, entry := range h.onlineUsers {
		displayName := entry.DisplayName
		permGroup := entry.PermissionGroup

		// Re-read profile for logged-in users to support hot-reload
		if entry.Username != "" {
			profile, err := auth.LoadProfile(config.UsersDir, entry.Username)
			if err == nil {
				displayName = profile.DisplayName
				permGroup = profile.PermissionGroup
			}
		}

		users = append(users, models.OnlineUser{
			DisplayName:     displayName,
			PermissionGroup: permGroup,
			Username:        entry.Username,
		})
	}

	msg := models.Message{
		Type:  "online_users",
		Count: len(users),
		Users: users,
	}
	data, _ := json.Marshal(msg)
	return data
}

func (h *Hub) broadcastOnlineUsers() {
	msg := h.buildOnlineUsersMsg()
	h.broadcast <- msg
}

func (h *Hub) BroadcastOnlineUsers() {
	h.broadcastOnlineUsers()
}

func (h *Hub) buildUserInfoMsg(client *Client) []byte {
	canModify := false
	if client.Username != "" {
		profile, err := auth.LoadProfile(config.UsersDir, client.Username)
		if err == nil {
			perms := config.ResolvePermissionsForGroup(profile.PermissionGroup)
			canModify = perms["modify_user_permission_group"]
		}
	} else {
		perms := config.ResolvePermissionsForGroup("default")
		canModify = perms["modify_user_permission_group"]
	}

	msg := models.Message{
		Type:                     "user_info",
		Username:                 client.Username,
		DisplayName:              client.DisplayName,
		PermissionGroup:          client.PermissionGroup,
		IsLoggedIn:               client.Username != "",
		CanModifyPermissionGroup: canModify,
	}
	data, _ := json.Marshal(msg)
	return data
}

func (h *Hub) resolveClientUser(r *http.Request) (sessionID, username, displayName, permGroup string, permissions map[string]bool) {
	permGroup = "default"
	displayName = "游客"

	cookie, err := r.Cookie("session_token")
	if err == nil && cookie.Value != "" {
		s, ok := h.sm.Validate(cookie.Value)
		if ok {
			sessionID = cookie.Value
			if s.LoggedIn && s.Username != "" {
				username = s.Username
				// Load profile from disk for natural hot-reload
				profile, err := auth.LoadProfile(config.UsersDir, username)
				if err == nil {
					displayName = profile.DisplayName
					permGroup = profile.PermissionGroup
				} else {
					displayName = s.GuestName
				}
			} else {
				displayName = s.GuestName
			}
		}
	}

	// Create new guest session if no valid session
	if sessionID == "" {
		token, _, err := h.sm.CreateGuest(displayName)
		if err == nil {
			sessionID = token
		} else {
			log.Printf("创建游客会话失败：%v", err)
		}
	}

	permissions = config.ResolvePermissionsForGroup(permGroup)
	return
}

func (c *Client) HasPermission(perm string) bool {
	permGroup := c.PermissionGroup

	// Re-read profile for logged-in users to support hot-reload of permission_group
	if c.Username != "" {
		profile, err := auth.LoadProfile(config.UsersDir, c.Username)
		if err == nil {
			permGroup = profile.PermissionGroup
			c.PermissionGroup = profile.PermissionGroup
		}
	}

	perms := config.ResolvePermissionsForGroup(permGroup)
	val, ok := perms[perm]
	if !ok {
		return false
	}
	return val
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func getClientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket升级失败：%v", err)
		return
	}

	sessionID, username, displayName, permGroup, permissions := h.resolveClientUser(r)

	client := &Client{
		hub:             h,
		conn:            conn,
		send:            make(chan []byte, 256),
		IP:              getClientIP(r),
		ConnID:          config.GenID(),
		SessionID:       sessionID,
		Username:        username,
		DisplayName:     displayName,
		PermissionGroup: permGroup,
		Permissions:     permissions,
	}

	// Set session cookie for new guest sessions
	if username == "" {
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   7 * 24 * 3600,
		})
	}

	h.register <- client

	fullState := h.buildFullState()
	if err := conn.WriteMessage(websocket.TextMessage, fullState); err != nil {
		log.Printf("发送完整状态失败：%v", err)
	}

	userInfo := h.buildUserInfoMsg(client)
	if err := conn.WriteMessage(websocket.TextMessage, userInfo); err != nil {
		log.Printf("发送用户信息失败：%v", err)
	}

	onlineUsers := h.buildOnlineUsersMsg()
	if err := conn.WriteMessage(websocket.TextMessage, onlineUsers); err != nil {
		log.Printf("发送在线用户列表失败：%v", err)
	}

	go client.writePump()
	go client.readPump()
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

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(config.MaxMsgSize)

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
