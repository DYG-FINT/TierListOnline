package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const sessionLifetime = 7 * 24 * time.Hour

type Session struct {
	LoggedIn  bool   `json:"logged_in"`
	Username  string `json:"username"`
	GuestName string `json:"guest_name"`
	CreatedAt int64  `json:"created_at"`
	ExpiresAt int64  `json:"expires_at"`
}

type SessionManager struct {
	mu  sync.RWMutex
	dir string
}

func NewSessionManager(dir string) *SessionManager {
	os.MkdirAll(dir, 0755)
	return &SessionManager{dir: dir}
}

func (sm *SessionManager) CreateGuest(guestName string) (string, *Session, error) {
	now := time.Now()
	s := &Session{
		LoggedIn:  false,
		Username:  "",
		GuestName: guestName,
		CreatedAt: now.Unix(),
		ExpiresAt: now.Add(sessionLifetime).Unix(),
	}
	token, err := sm.save(s)
	if err != nil {
		return "", nil, err
	}
	return token, s, nil
}

func (sm *SessionManager) CreateLoggedIn(username string) (string, *Session, error) {
	now := time.Now()
	s := &Session{
		LoggedIn:  true,
		Username:  username,
		GuestName: username,
		CreatedAt: now.Unix(),
		ExpiresAt: now.Add(sessionLifetime).Unix(),
	}
	token, err := sm.save(s)
	if err != nil {
		return "", nil, err
	}
	return token, s, nil
}

func (sm *SessionManager) Validate(token string) (*Session, bool) {
	if token == "" {
		return nil, false
	}
	// Sanitize token to prevent path traversal
	token = filepath.Base(token)
	if token == "." || token == ".." {
		return nil, false
	}

	path := filepath.Join(sm.dir, token+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, false
	}

	// Check expiry
	if time.Now().Unix() > s.ExpiresAt {
		os.Remove(path)
		return nil, false
	}

	// Renew the session (touch expires_at)
	now := time.Now()
	s.ExpiresAt = now.Add(sessionLifetime).Unix()
	sm.mu.Lock()
	newData, _ := json.MarshalIndent(s, "", "  ")
	if newData != nil {
		if err := os.WriteFile(path, newData, 0644); err != nil {
			log.Printf("续期session文件失败：%v", err)
		}
	}
	sm.mu.Unlock()

	return &s, true
}

func (sm *SessionManager) UpdateGuestName(token string, newName string) error {
	token = filepath.Base(token)
	path := filepath.Join(sm.dir, token+".json")

	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	s.GuestName = newName
	newData, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, newData, 0644)
}

func (sm *SessionManager) Delete(token string) {
	token = filepath.Base(token)
	if token == "." || token == ".." {
		return
	}
	path := filepath.Join(sm.dir, token+".json")
	os.Remove(path)
}

func (sm *SessionManager) CleanupExpired() {
	entries, err := os.ReadDir(sm.dir)
	if err != nil {
		return
	}
	now := time.Now().Unix()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(sm.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var s Session
		if json.Unmarshal(data, &s) != nil {
			continue
		}
		if now > s.ExpiresAt {
			os.Remove(path)
		}
	}
}

func (sm *SessionManager) save(s *Session) (string, error) {
	token := genToken()
	path := filepath.Join(sm.dir, token+".json")

	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}

	return token, nil
}

func genToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}
