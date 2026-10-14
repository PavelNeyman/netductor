// Package opconfig — operator workstation config (~/.config/netductor/tui.yaml).
package opconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TUI is the on-disk operator connection profile (shared by TUI + Web).
type TUI struct {
	RemoteHost     string `json:"remote_host"`
	RemoteUser     string `json:"remote_user"`
	RemoteKey      string `json:"remote_key"`
	RemotePassword string `json:"remote_password"` // first deploy only; prefer empty
	Lang           string `json:"lang"`           // auto|ru|en
	SecondaryHost  string `json:"secondary_host"`
	LastEdgeID     string `json:"last_edge_id"`
	Mode           string `json:"mode"`
}

func Path() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "netductor", "tui.yaml")
}

func Load() TUI {
	s := TUI{RemoteUser: "root", Lang: "auto"}
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

// Save writes the full profile (0600).
func Save(s TUI) error {
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
	body := fmt.Sprintf(`# netductor operator settings (TUI + Web) — edit freely
# lang: auto | ru | en
# mode: vps | workstation | openwrt | operator
remote_host: %q
remote_user: %q
remote_key: %q
remote_password: %q
secondary_host: %q
last_edge_id: %q
lang: %q
mode: %q
`, s.RemoteHost, s.RemoteUser, s.RemoteKey, s.RemotePassword, s.SecondaryHost, s.LastEdgeID, s.Lang, s.Mode)
	return os.WriteFile(path, []byte(body), 0o600)
}

// Merge applies non-empty fields from patch onto base. Empty patch strings leave base.
// If clearPassword is true, remote_password is cleared even when empty in patch.
func Merge(base, patch TUI, clearPassword bool) TUI {
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
	return out
}

func parseYAML(text string, s *TUI) {
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
		v := strings.TrimSpace(line[i+1:])
		v = strings.Trim(v, `"'`)
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
		}
	}
}
