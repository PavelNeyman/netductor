package vpn

import (
	"fmt"
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
	Kind          string   `json:"kind,omitempty"`
}

func List() ([]User, error) {
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

// IsDeviceProfile: explicit kind=device or edge-* accounts.
func IsDeviceProfile(name string) bool {
	name = strings.TrimSpace(name)
	if IsEdgeUser(name) {
		return true
	}
	r, err := loadRegistry()
	if err != nil {
		return false
	}
	for _, u := range r.Users {
		if u.Name == name {
			return strings.EqualFold(strings.TrimSpace(u.Kind), "device")
		}
	}
	return false
}

// SetKind moves a VPN profile between user and device lists without rotating UUID.
func SetKind(name, kind string) error {
	name = strings.TrimSpace(name)
	kind = strings.ToLower(strings.TrimSpace(kind))
	if kind != "user" && kind != "device" {
		return fmt.Errorf("kind must be user|device")
	}
	if !ValidName(name) {
		return fmt.Errorf("invalid name")
	}
	r, err := loadRegistry()
	if err != nil {
		return err
	}
	found := false
	for i := range r.Users {
		if r.Users[i].Name != name {
			continue
		}
		found = true
		if kind == "user" {
			r.Users[i].Kind = ""
		} else {
			r.Users[i].Kind = "device"
		}
		break
	}
	if !found {
		return fmt.Errorf("user not found: %s", name)
	}
	return writeRegistry(r)
}

// ListHumanNames returns non-device, non-relay names for Users UI.
func ListHumanNames() []string {
	r, err := loadRegistry()
	if err != nil {
		return nil
	}
	var out []string
	for _, u := range r.Users {
		if u.Name == "relay-uplink" || IsEdgeUser(u.Name) {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(u.Kind), "device") {
			continue
		}
		out = append(out, u.Name)
	}
	return out
}

// ListDeviceNames returns kind=device profiles for Devices UI.
func ListDeviceNames(includeEdge bool) []string {
	r, err := loadRegistry()
	if err != nil {
		return nil
	}
	var out []string
	for _, u := range r.Users {
		if u.Name == "relay-uplink" {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(u.Kind), "device") {
			out = append(out, u.Name)
			continue
		}
		if includeEdge && IsEdgeUser(u.Name) {
			out = append(out, u.Name)
		}
	}
	return out
}
