package vpn

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SNIPreset is a whitelist-friendly handshake name for Reality.
type SNIPreset struct {
	Name string `json:"name"`
	SNI  string `json:"sni"`
	Note string `json:"note,omitempty"`
}

func defaultSNIPresets() []SNIPreset {
	return []SNIPreset{
		{Name: "yandex", SNI: "ya.ru", Note: "default RU"},
		{Name: "vk", SNI: "api.vk.com", Note: "mobile-friendly"},
		{Name: "mail", SNI: "mail.ru", Note: "alt RU"},
		{Name: "gosuslugi", SNI: "gosuslugi.ru", Note: "gov WL"},
		{Name: "yota-try", SNI: "api.vk.me", Note: "experimental under carrier WL"},
	}
}

func sniPresetsPath() string {
	return filepath.Join(paths.EtcDir(), "sni_presets.json")
}

// ListSNIPresets loads operator overrides or defaults.
func ListSNIPresets() []SNIPreset {
	b, err := os.ReadFile(sniPresetsPath())
	if err != nil {
		return defaultSNIPresets()
	}
	var list []SNIPreset
	if json.Unmarshal(b, &list) != nil || len(list) == 0 {
		return defaultSNIPresets()
	}
	return list
}

// EnsureSNIPresetsFile writes defaults if missing.
func EnsureSNIPresetsFile() error {
	p := sniPresetsPath()
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	raw, _ := json.MarshalIndent(defaultSNIPresets(), "", "  ")
	return os.WriteFile(p, append(raw, '\n'), 0o644)
}
