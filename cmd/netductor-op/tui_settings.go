package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/opconfig"
)

// TUI settings: ~/.config/netductor/tui.yaml — shared with Web via internal/opconfig.

type tuiSettings struct {
	RemoteHost     string
	RemoteUser     string
	RemoteKey      string
	RemotePassword string
	Lang           string
	SecondaryHost  string
	LastEdgeID     string
	Mode           string
}

func tuiConfigPath() string { return opconfig.Path() }

func tuiConfigPathLegacyJSON() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "netductor", "tui.json")
}

func fromOp(s opconfig.TUI) tuiSettings {
	return tuiSettings{
		RemoteHost: s.RemoteHost, RemoteUser: s.RemoteUser, RemoteKey: s.RemoteKey,
		RemotePassword: s.RemotePassword, Lang: s.Lang, SecondaryHost: s.SecondaryHost,
		LastEdgeID: s.LastEdgeID, Mode: s.Mode,
	}
}

func toOp(s tuiSettings) opconfig.TUI {
	return opconfig.TUI{
		RemoteHost: s.RemoteHost, RemoteUser: s.RemoteUser, RemoteKey: s.RemoteKey,
		RemotePassword: s.RemotePassword, Lang: s.Lang, SecondaryHost: s.SecondaryHost,
		LastEdgeID: s.LastEdgeID, Mode: s.Mode,
	}
}

func loadTUISettings() tuiSettings {
	s := fromOp(opconfig.Load())
	if s.RemoteUser == "" {
		s.RemoteUser = "root"
	}
	if s.Lang == "" {
		s.Lang = "auto"
	}
	if s.RemoteHost == "" {
		if jp := tuiConfigPathLegacyJSON(); jp != "" {
			if b, err := os.ReadFile(jp); err == nil {
				migrateJSONish(string(b), &s)
				_ = saveTUISettings(s)
			}
		}
	}
	return s
}

func saveTUISettings(s tuiSettings) error {
	return opconfig.Save(toOp(s))
}

func migrateJSONish(text string, s *tuiSettings) {
	for _, key := range []string{"remote_host", "remote_user", "remote_key", "remote_password", "lang", "secondary_host"} {
		pat := `"` + key + `"`
		i := strings.Index(text, pat)
		if i < 0 {
			continue
		}
		rest := text[i+len(pat):]
		j := strings.Index(rest, `"`)
		if j < 0 {
			continue
		}
		rest = rest[j+1:]
		k := strings.Index(rest, `"`)
		if k < 0 {
			continue
		}
		v := rest[:k]
		switch key {
		case "remote_host":
			s.RemoteHost = v
		case "remote_user":
			s.RemoteUser = v
		case "remote_key":
			s.RemoteKey = v
		case "remote_password":
			s.RemotePassword = v
		case "lang":
			s.Lang = v
		case "secondary_host":
			s.SecondaryHost = v
		}
	}
}

func (m *model) applySettings(s tuiSettings) {
	m.remoteHost = s.RemoteHost
	if s.Mode != "" {
		m.mode = runMode(s.Mode)
	}
	m.remoteUser = s.RemoteUser
	if m.remoteUser == "" {
		m.remoteUser = "root"
	}
	m.remoteKey = s.RemoteKey
	m.remotePassword = s.RemotePassword
	m.langPref = s.Lang
	switch strings.ToLower(strings.TrimSpace(s.Lang)) {
	case "ru":
		m.lang = langRU
	case "en":
		m.lang = langEN
	default:
		m.lang = detectLang()
		m.langPref = "auto"
	}
}

func (m model) snapshotSettings() tuiSettings {
	lang := m.langPref
	if lang == "" {
		lang = "auto"
	}
	if lang != "auto" {
		if m.lang == langRU {
			lang = "ru"
		} else {
			lang = "en"
		}
	}
	prev := loadTUISettings()
	return tuiSettings{
		RemoteHost: m.remoteHost, RemoteUser: orDefault(m.remoteUser, "root"),
		RemoteKey: m.remoteKey, RemotePassword: m.remotePassword, Lang: lang,
		Mode: string(m.mode), SecondaryHost: prev.SecondaryHost, LastEdgeID: prev.LastEdgeID,
	}
}
