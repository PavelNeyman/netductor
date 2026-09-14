package vpn

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SNIPreset is a whitelist-oriented Reality server_name.
type SNIPreset struct {
	Name string `json:"name"`
	SNI  string `json:"sni"`
	Note string `json:"note,omitempty"`
}

func defaultSNIPresets() []SNIPreset {
	return []SNIPreset{
		{Name: "vk-api", SNI: "api.vk.me", Note: "default — commercial WL profiles"},
		{Name: "vk", SNI: "api.vk.com", Note: "VK API"},
		{Name: "vk-live", SNI: "vklive.enotfast.com", Note: "habr sample"},
		{Name: "userapi", SNI: "userapi.com", Note: "VK CDN"},
		{Name: "yandex", SNI: "ya.ru", Note: "Yandex"},
		{Name: "yastatic", SNI: "yastatic.net", Note: "Yandex CDN"},
		{Name: "ya-storage", SNI: "storage.yandex.net", Note: "Yandex Object Storage"},
		{Name: "okcdn", SNI: "okcdn.ru", Note: "OK CDN"},
		{Name: "mail", SNI: "mail.ru", Note: "Mail.ru"},
		{Name: "x5", SNI: "id.x5.ru", Note: "commercial WL hop"},
		{Name: "gosuslugi", SNI: "gosuslugi.ru", Note: "gov"},
	}
}

func sniPresetsPath() string {
	return filepath.Join(paths.EtcDir(), "sni_presets.json")
}

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

func EnsureSNIPresetsFile() error {
	p := sniPresetsPath()
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	raw, _ := json.MarshalIndent(defaultSNIPresets(), "", "  ")
	return os.WriteFile(p, append(raw, 10), 0o644)
}
