package auth

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"tlo/config"
)

const sessionCookieName = "session_token"

func getSessionToken(r *http.Request) string {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   7 * 24 * 3600,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

type authResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func HandleRegister(usersDir string, sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, authResponse{Message: "方法不允许"})
			return
		}

		ip := getClientIP(r)
		if !config.IsWhitelistIP(ip) {
			perms := config.ResolvePermissionsForGroup("default")
			if !perms["register"] {
				writeJSON(w, 403, authResponse{Message: "注册功能暂不可用"})
				return
			}
		}

		var req struct {
			Username    string `json:"username"`
			Password    string `json:"password"`
			DisplayName string `json:"display_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, 400, authResponse{Message: "请求格式错误"})
			return
		}

		if req.DisplayName == "" {
			req.DisplayName = req.Username
		}

		if err := CreateUser(usersDir, req.Username, req.Password, req.DisplayName); err != nil {
			writeJSON(w, 400, authResponse{Message: err.Error()})
			return
		}

		token, _, err := sm.CreateLoggedIn(req.Username)
		if err != nil {
			writeJSON(w, 500, authResponse{Message: "创建会话失败"})
			return
		}

		setSessionCookie(w, token)
		writeJSON(w, 200, authResponse{Success: true, Message: "注册成功"})
	}
}

func HandleLogin(usersDir string, sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, authResponse{Message: "方法不允许"})
			return
		}

		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, 400, authResponse{Message: "请求格式错误"})
			return
		}

		profile, err := LoadProfile(usersDir, req.Username)
		if err != nil {
			writeJSON(w, 401, authResponse{Message: "用户名或密码错误"})
			return
		}

		if !VerifyPassword(profile, req.Password) {
			writeJSON(w, 401, authResponse{Message: "用户名或密码错误"})
			return
		}

		token, _, err := sm.CreateLoggedIn(req.Username)
		if err != nil {
			writeJSON(w, 500, authResponse{Message: "创建会话失败"})
			return
		}

		setSessionCookie(w, token)
		writeJSON(w, 200, authResponse{Success: true, Message: "登录成功"})
	}
}

func HandleLogout(sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, authResponse{Message: "方法不允许"})
			return
		}

		token := getSessionToken(r)
		if token != "" {
			sm.Delete(token)
		}
		clearSessionCookie(w)
		writeJSON(w, 200, authResponse{Success: true, Message: "已退出登录"})
	}
}

func HandleChangePassword(usersDir string, sm *SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, 405, authResponse{Message: "方法不允许"})
			return
		}

		token := getSessionToken(r)
		session, ok := sm.Validate(token)
		if !ok || !session.LoggedIn {
			writeJSON(w, 401, authResponse{Message: "请先登录"})
			return
		}

		var req struct {
			OldPassword string `json:"old_password"`
			NewPassword string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, 400, authResponse{Message: "请求格式错误"})
			return
		}

		if req.NewPassword == "" {
			writeJSON(w, 400, authResponse{Message: "新密码不能为空"})
			return
		}

		profile, err := LoadProfile(usersDir, session.Username)
		if err != nil {
			writeJSON(w, 500, authResponse{Message: "读取用户信息失败"})
			return
		}

		if !VerifyPassword(profile, req.OldPassword) {
			writeJSON(w, 400, authResponse{Message: "原密码错误"})
			return
		}

		if err := UpdatePassword(usersDir, session.Username, req.NewPassword); err != nil {
			writeJSON(w, 500, authResponse{Message: "修改密码失败"})
			return
		}

		writeJSON(w, 200, authResponse{Success: true, Message: "密码修改成功"})
	}
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
