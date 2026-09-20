package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// TUI settings: ~/.config/netductor/tui.yaml (hand-editable)

type tuiSettings struct {
	RemoteHost     string // remote_host — day-2 primary
	RemoteUser     string // remote_user
	RemoteKey      string // remote_key — SSH private key for primary
	RemotePassword string // remote_password — only for first deploy; prefer empty after
	Lang           string // lang: auto|ru|en
	SecondaryHost  string // secondary_host
	LastEdgeID     string // last_edge_id
}

func tuiConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "netductor", "tui.yaml")
}

func tuiConfigPathLegacyJSON() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".config", "netductor", "tui.json")
}

func loadTUISettings() tuiSettings {
	s := tuiSettings{RemoteUser: "root", Lang: "auto"}
	path := tuiConfigPath()
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			parseSimpleYAML(string(b), &s)
			if s.RemoteUser == "" {
				s.RemoteUser = "root"
			}
			if s.Lang == "" {
				s.Lang = "auto"
			}
			return s
		}
	}
	// one-time migrate from json if present
	if jp := tuiConfigPathLegacyJSON(); jp != "" {
		if b, err := os.ReadFile(jp); err == nil {
			// minimal: extract quoted values by key
			migrateJSONish(string(b), &s)
			_ = saveTUISettings(s)
		}
	}
	return s
}

func saveTUISettings(s tuiSettings) error {
	path := tuiConfigPath()
	if path == "" {
		return os.ErrInvalid
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o700)
	if s.RemoteUser == "" {
		s.RemoteUser = "root"
	}
	if s.Lang == "" {
		s.Lang = "auto"
	}
	body := fmt.Sprintf(`# netductor TUI settings — edit freely
# lang: auto | ru | en  (auto = system locale)
# After primary deploy, remote_key is the SSH key; remote_password should be empty.
remote_host: %q
remote_user: %q
remote_key: %q
remote_password: %q
secondary_host: %q
last_edge_id: %q
lang: %q
`, s.RemoteHost, s.RemoteUser, s.RemoteKey, s.RemotePassword, s.SecondaryHost, s.LastEdgeID, s.Lang)
	return os.WriteFile(path, []byte(body), 0o600)
}

func parseSimpleYAML(text string, s *tuiSettings) {
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.IndexByte(line, ':')
		if i < 0 {
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
		case "lang":
			s.Lang = v
		case "secondary_host":
			s.SecondaryHost = v
		case "last_edge_id":
			s.LastEdgeID = v
		}
	}
}

func migrateJSONish(text string, s *tuiSettings) {
	// best-effort extract "key": "value"
	for _, key := range []string{"remote_host", "remote_user", "remote_key", "remote_password", "lang"} {
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
		}
	}
}

func (m *model) applySettings(s tuiSettings) {
	m.remoteHost = s.RemoteHost
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
	default: // auto / empty
		m.lang = detectLang()
		m.langPref = "auto"
	}
}

func (m model) snapshotSettings() tuiSettings {
	lang := m.langPref
	if lang == "" {
		lang = "auto"
	}
	// if user toggled with `l`, prefer explicit
	if lang != "auto" {
		if m.lang == langRU {
			lang = "ru"
		} else {
			lang = "en"
		}
	}
	return tuiSettings{
		RemoteHost: m.remoteHost, RemoteUser: orDefault(m.remoteUser, "root"),
		RemoteKey: m.remoteKey, RemotePassword: m.remotePassword, Lang: lang,
	}
}
