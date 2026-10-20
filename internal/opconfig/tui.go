// Package opconfig — operator workstation config (Mac/PC).
// File: ~/.config/netductor/operator.conf
//
// Not the same as node /etc/netductor/netductor.conf (KEY=value product env on VPS).
package opconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Settings is the on-disk operator connection profile (TUI + Web).
type Settings struct {
	RemoteHost     string `json:"remote_host"`
	RemoteUser     string `json:"remote_user"`
	RemoteKey      string `json:"remote_key"`
	RemotePassword string `json:"remote_password"`
	Lang           string `json:"lang"`
	SecondaryHost  string `json:"secondary_host"`
	LastEdgeID     string `json:"last_edge_id"`
	Mode           string `json:"mode"`
	// Web Operate Result panel (VS Code–style dock). Persisted with connection profile.
	ResultVisible  string `json:"result_visible"`  // "1" | "0" | ""
	ResultPosition string `json:"result_position"` // bottom | right | top
	ResultMode     string `json:"result_mode"`     // overlay | inline | both
	ResultSize     string `json:"result_size"`     // px for active dock edge
	// EdgeUSBPackages — comma/space-separated OpenWrt package base names for USB/overlay offline cache.
	// Empty = deploy.DefaultUSBPkgNames(). Edit without rebuilding op.
	EdgeUSBPackages string `json:"edge_usb_packages"`
}

// TUI is an alias kept for existing call sites.
type TUI = Settings

func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "netductor")
}

// Path is ~/.config/netductor/operator.conf
func Path() string {
	dir := configDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "operator.conf")
}

func Load() Settings {
	s := Settings{RemoteUser: "root", Lang: "auto"}
	path := Path()
	if path == "" {
		return s
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	parseYAML(string(b), &s)
	if s.RemoteUser == "" {
		s.RemoteUser = "root"
	}
	if s.Lang == "" {
		s.Lang = "auto"
	}
	return s
}

func Save(s Settings) error {
	path := Path()
	if path == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if s.RemoteUser == "" {
		s.RemoteUser = "root"
	}
	if s.Lang == "" {
		s.Lang = "auto"
	}
	body := fmt.Sprintf(`# netductor operator settings (TUI + Web)
# Path: ~/.config/netductor/operator.conf
#
# Not /etc/netductor/netductor.conf on the VPS (node product KEY=value).
#
# lang: auto | ru | en
# mode: vps | workstation | openwrt | operator
# result_*: Web Operate Result dock (visible 0|1, position bottom|right|top, mode overlay|inline|both, size px)
# edge_usb_packages: comma-separated OpenWrt base names for offline USB/overlay cache (empty = built-in defaults)
remote_host: %q
remote_user: %q
remote_key: %q
remote_password: %q
secondary_host: %q
last_edge_id: %q
lang: %q
mode: %q
result_visible: %q
result_position: %q
result_mode: %q
result_size: %q
edge_usb_packages: %q
`, s.RemoteHost, s.RemoteUser, s.RemoteKey, s.RemotePassword, s.SecondaryHost, s.LastEdgeID, s.Lang, s.Mode,
		s.ResultVisible, s.ResultPosition, s.ResultMode, s.ResultSize, s.EdgeUSBPackages)
	return os.WriteFile(path, []byte(body), 0o600)
}

func Merge(base, patch Settings, clearPassword bool) Settings {
	out := base
	if patch.RemoteHost != "" {
		out.RemoteHost = patch.RemoteHost
	}
	if patch.RemoteUser != "" {
		out.RemoteUser = patch.RemoteUser
	}
	if patch.RemoteKey != "" {
		out.RemoteKey = patch.RemoteKey
	}
	if patch.RemotePassword != "" {
		out.RemotePassword = patch.RemotePassword
	} else if clearPassword {
		out.RemotePassword = ""
	}
	if patch.Lang != "" {
		out.Lang = patch.Lang
	}
	if patch.SecondaryHost != "" {
		out.SecondaryHost = patch.SecondaryHost
	}
	if patch.LastEdgeID != "" {
		out.LastEdgeID = patch.LastEdgeID
	}
	if patch.Mode != "" {
		out.Mode = patch.Mode
	}
	if patch.ResultVisible != "" {
		out.ResultVisible = patch.ResultVisible
	}
	if patch.ResultPosition != "" {
		out.ResultPosition = patch.ResultPosition
	}
	if patch.ResultMode != "" {
		out.ResultMode = patch.ResultMode
	}
	if patch.ResultSize != "" {
		out.ResultSize = patch.ResultSize
	}
	// EdgeUSBPackages: allow explicit clear via single "-"
	if patch.EdgeUSBPackages == "-" {
		out.EdgeUSBPackages = ""
	} else if patch.EdgeUSBPackages != "" {
		out.EdgeUSBPackages = patch.EdgeUSBPackages
	}
	return out
}

func parseYAML(text string, s *Settings) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, ":")
		if i <= 0 {
			continue
		}
		k := strings.TrimSpace(line[:i])
		v := strings.Trim(strings.TrimSpace(line[i+1:]), `"'`)
		switch k {
		case "remote_host":
			s.RemoteHost = v
		case "remote_user":
			s.RemoteUser = v
		case "remote_key":
			s.RemoteKey = v
		case "remote_password":
			s.RemotePassword = v
		case "secondary_host":
			s.SecondaryHost = v
		case "last_edge_id":
			s.LastEdgeID = v
		case "lang":
			s.Lang = v
		case "mode":
			s.Mode = v
		case "result_visible":
			s.ResultVisible = v
		case "result_position":
			s.ResultPosition = v
		case "result_mode":
			s.ResultMode = v
		case "result_size":
			s.ResultSize = v
		case "edge_usb_packages":
			s.EdgeUSBPackages = v
		}
	}
}
