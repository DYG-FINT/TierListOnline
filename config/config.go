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
	StateFile  = "storage/active/state.json"
	UploadDir  = "storage/active/uploads"
	PresetDir  = "storage/presets"
	MaxMsgSize = 10 * 1024 * 1024 // 10 MB
)

var DefaultPermissions = map[string]bool{
	"set_title":            true,
	"update_label":         true,
	"update_label_color":   true,
	"add_row":              true,
	"delete_row":           true,
	"clear_row":            true,
	"move_row":             true,
	"upload_image":         true,
	"move_image":           true,
	"delete_image":         true,
	"stage_all":            true,
	"apply_color_sequence": true,
	"toggle_image_fit":     true,
	"list_presets":         true,
	"save_preset":          true,
	"load_preset":          true,
}

var DefaultPermissionGroups = map[string]map[string]bool{
	"default": {"#free": true},
}

var DefaultPermissionPresets = map[string]map[string]bool{
	"free": {
		"set_title":            true,
		"update_label":         true,
		"update_label_color":   true,
		"add_row":              true,
		"delete_row":           true,
		"clear_row":            true,
		"move_row":             true,
		"upload_image":         true,
		"move_image":           true,
		"delete_image":         true,
		"stage_all":            true,
		"apply_color_sequence": true,
		"toggle_image_fit":     true,
		"list_presets":         true,
		"save_preset":          true,
		"load_preset":          true,
	},
	"temporary_free": {
		"#free":       true,
		"save_preset": false,
	},
	"sort_only": {
		"move_image":       true,
		"toggle_image_fit": true,
	},
	"reset": {
		"list_presets": true,
		"load_preset":  true,
		"move_image":   true,
	},
	"collaborative": {
		"#sort_only": true,
		"#reset":     true,
		"stage_all":  true,
	},
}

var (
	cfgMu               sync.RWMutex
	maxUploadSizeMB     int      = 10
	allowedExtensions   []string
	permissionGroups    map[string]map[string]bool
	permissionPresets   map[string]map[string]bool
	resolvedPermissions map[string]bool
	whitelistIPs        []string
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

func HasPermission(perm string) bool {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	if resolvedPermissions == nil {
		return true
	}
	val, ok := resolvedPermissions[perm]
	if !ok {
		return false
	}
	return val
}

func GetAllPermissions() map[string]bool {
	cfgMu.RLock()
	defer cfgMu.RUnlock()
	if resolvedPermissions == nil {
		return DefaultPermissions
	}
	cp := make(map[string]bool, len(resolvedPermissions))
	for k, v := range resolvedPermissions {
		cp[k] = v
	}
	return cp
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

func ApplySettings(maxSize int, extensions []string, groups map[string]map[string]bool, presets map[string]map[string]bool, ips []string) {
	cfgMu.Lock()
	defer cfgMu.Unlock()
	if maxSize > 0 {
		maxUploadSizeMB = maxSize
	}
	if len(extensions) > 0 {
		allowedExtensions = extensions
	}
	if groups != nil {
		permissionGroups = groups
	}
	if presets != nil {
		permissionPresets = presets
	}
	resolvedPermissions = resolvePermissions(permissionGroups["default"], permissionPresets)
	whitelistIPs = ips
}

func resolvePermissions(perms map[string]bool, presets map[string]map[string]bool) map[string]bool {
	result := make(map[string]bool)
	if perms != nil {
		resolveMap(result, perms, presets, nil)
	}
	return result
}

func resolveMap(result map[string]bool, input map[string]bool, presets map[string]map[string]bool, visited map[string]bool) {
	if visited == nil {
		visited = make(map[string]bool)
	}

	// First pass: resolve preset references
	for key, val := range input {
		if strings.HasPrefix(key, "#") && val {
			name := key[1:]
			if visited[name] {
				continue
			}
			visited[name] = true
			if preset, ok := presets[name]; ok {
				resolveMap(result, preset, presets, visited)
			}
			delete(visited, name)
		}
	}

	// Second pass: apply direct keys (override presets)
	for key, val := range input {
		if !strings.HasPrefix(key, "#") {
			result[key] = val
		}
	}
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
