package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"tlo/config"
)

type UserProfile struct {
	DisplayName     string          `json:"display_name"`
	PasswordHash    string          `json:"password_hash"`
	PermissionGroup string          `json:"permission_group"`
	Permissions     map[string]bool `json:"permissions"`
}

func ValidateUsername(name string) error {
	if name == "" {
		return errors.New("用户名不能为空")
	}
	if len(name) > 64 {
		return errors.New("用户名过长")
	}
	if strings.Contains(name, " ") {
		return errors.New("用户名不能包含空格")
	}
	if strings.ContainsAny(name, "\\/:*?\"<>|") {
		return errors.New("用户名包含非法字符")
	}
	// Reject names starting with dot (hidden files)
	if strings.HasPrefix(name, ".") {
		return errors.New("用户名不能以点号开头")
	}
	// Reject reserved Windows device names
	lower := strings.ToLower(name)
	reserved := map[string]bool{
		"con": true, "prn": true, "aux": true, "nul": true,
		"com1": true, "com2": true, "com3": true, "com4": true,
		"com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
		"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true,
		"lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
	}
	if reserved[lower] {
		return errors.New("用户名包含保留名称")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func VerifyPassword(profile *UserProfile, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(password))
	return err == nil
}

func CreateUser(usersDir, username, password, displayName string) error {
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if password == "" {
		return errors.New("密码不能为空")
	}
	if displayName == "" {
		displayName = username
	}

	userDir := filepath.Join(usersDir, username)
	if _, err := os.Stat(userDir); err == nil {
		return errors.New("用户已存在")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return err
	}

	profile := UserProfile{
		DisplayName:     displayName,
		PasswordHash:    hash,
		PermissionGroup: config.GetNewUserPermissionGroup(),
		Permissions:     make(map[string]bool),
	}

	if err := os.MkdirAll(userDir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(userDir, "profile.json"), data, 0644)
}

func LoadProfile(usersDir, username string) (*UserProfile, error) {
	path := filepath.Join(usersDir, username, "profile.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var profile UserProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	if profile.Permissions == nil {
		profile.Permissions = make(map[string]bool)
	}
	return &profile, nil
}

func UpdatePassword(usersDir, username, newPassword string) error {
	profile, err := LoadProfile(usersDir, username)
	if err != nil {
		return err
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}

	profile.PasswordHash = hash
	return saveProfile(usersDir, username, profile)
}

func UpdatePermissionGroup(usersDir, username, permissionGroup string) error {
	profile, err := LoadProfile(usersDir, username)
	if err != nil {
		return err
	}

	profile.PermissionGroup = permissionGroup
	return saveProfile(usersDir, username, profile)
}

func UpdateDisplayName(usersDir, username, displayName string) error {
	profile, err := LoadProfile(usersDir, username)
	if err != nil {
		return err
	}

	profile.DisplayName = displayName
	return saveProfile(usersDir, username, profile)
}

func saveProfile(usersDir, username string, profile *UserProfile) error {
	path := filepath.Join(usersDir, username, "profile.json")
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
