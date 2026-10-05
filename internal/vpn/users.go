package vpn

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_-]{0,63}$`)

func Bin() string {
	return paths.VPNBin()
}
func Clients() string { return paths.ClientsDir() }

func ValidName(name string) bool {
	return name != "" && nameRe.MatchString(name)
}

type User struct {
	Name          string   `json:"name"`
	Enabled       bool     `json:"enabled"`
	UUID          string   `json:"uuid"`
	Note          string   `json:"note"`
	Created       string   `json:"created"`
	AllowInternet *bool    `json:"allow_internet,omitempty"`
	Services      []string `json:"services,omitempty"`
	ServicesMode  string   `json:"services_mode,omitempty"`
}

func List() ([]User, error) {
	// R13: native only — CLI fallback hid real registry errors
	return ListNative()
}

func Add(name, note string) (string, error) {
	return AddNative(name, note)
}

func Note(name, note string) (string, error) {
	if err := SetNoteNative(name, note); err != nil {
		return "", err
	}
	return "note updated " + name, nil
}
func Disable(name string) (string, error) {
	if err := SetEnabledNative(name, false); err != nil {
		return "", err
	}
	return "disabled " + name, nil
}
func Enable(name string) (string, error) {
	if err := SetEnabledNative(name, true); err != nil {
		return "", err
	}
	return "enabled " + name, nil
}
func Revoke(name string) (string, error) {
	if err := RevokeNative(name); err != nil {
		return "", err
	}
	return "revoked " + name, nil
}

func ReadClient(name string, candidates ...string) (string, bool) {
	if !ValidName(name) {
		return "", false
	}
	for _, c := range candidates {
		if c == "" || strings.Contains(c, "/") || strings.Contains(c, "..") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(Clients(), name, c))
		if err == nil {
			return strings.TrimSpace(string(b)), true
		}
	}
	return "", false
}

func QRPath(name string) string {
	return filepath.Join(Clients(), name, "qr.png")
}

func Rename(oldName, newName string) (string, error) {
	if err := RenameNative(oldName, newName); err != nil {
		return "", err
	}
	return newName, nil
}

// IsEdgeUser reports VPN accounts created for OpenWrt/edge routers (not human Users UI).
func IsEdgeUser(name string) bool {
	name = strings.TrimSpace(name)
	return strings.HasPrefix(name, "edge-")
}
