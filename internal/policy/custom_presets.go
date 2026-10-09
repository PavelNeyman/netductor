package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// CustomPreset is a user-defined access template (not the built-in media/full/none).
type CustomPreset struct {
	ID            string    `json:"id"`
	Title         string    `json:"title"`
	AllowInternet bool      `json:"allow_internet"`
	ServicesMode  string    `json:"services_mode"` // all | list
	Services      []string  `json:"services,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type customPresetFile struct {
	Presets []CustomPreset `json:"presets"`
}

var customMu sync.Mutex

func customPresetPath() string {
	return filepath.Join(paths.EtcDir(), "policy-presets.json")
}

func LoadCustomPresets() ([]CustomPreset, error) {
	customMu.Lock()
	defer customMu.Unlock()
	return loadCustomPresetsUnlocked()
}

func loadCustomPresetsUnlocked() ([]CustomPreset, error) {
	b, err := os.ReadFile(customPresetPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f customPresetFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return f.Presets, nil
}

func saveCustomPresetsUnlocked(list []CustomPreset) error {
	_ = os.MkdirAll(filepath.Dir(customPresetPath()), 0o700)
	b, err := json.MarshalIndent(customPresetFile{Presets: list}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(customPresetPath(), append(b, '\n'), 0o600)
}

func slugID(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if unicode.IsSpace(r) || r == '/' {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = fmt.Sprintf("preset-%d", time.Now().Unix()%100000)
	}
	return out
}

// UpsertCustomPreset creates or updates by id (or title slug).
func UpsertCustomPreset(p CustomPreset) (CustomPreset, error) {
	customMu.Lock()
	defer customMu.Unlock()
	list, err := loadCustomPresetsUnlocked()
	if err != nil {
		return p, err
	}
	if p.ID == "" {
		p.ID = slugID(p.Title)
	} else {
		p.ID = slugID(p.ID)
	}
	if p.Title == "" {
		p.Title = p.ID
	}
	p.Normalize()
	now := time.Now().UTC()
	found := false
	for i := range list {
		if list[i].ID == p.ID {
			p.CreatedAt = list[i].CreatedAt
			if p.CreatedAt.IsZero() {
				p.CreatedAt = now
			}
			p.UpdatedAt = now
			list[i] = p
			found = true
			break
		}
	}
	if !found {
		p.CreatedAt = now
		p.UpdatedAt = now
		list = append(list, p)
	}
	if err := saveCustomPresetsUnlocked(list); err != nil {
		return p, err
	}
	return p, nil
}

func (p *CustomPreset) Normalize() {
	if p.ServicesMode != "all" && p.ServicesMode != "list" {
		if len(p.Services) == 0 && p.AllowInternet {
			// empty list + internet = internet-only style
			p.ServicesMode = "list"
		} else if len(p.Services) == 0 {
			p.ServicesMode = "all"
		} else {
			p.ServicesMode = "list"
		}
	}
}

// DeleteCustomPreset removes by id.
func DeleteCustomPreset(id string) error {
	customMu.Lock()
	defer customMu.Unlock()
	id = slugID(id)
	list, err := loadCustomPresetsUnlocked()
	if err != nil {
		return err
	}
	out := list[:0]
	for _, p := range list {
		if p.ID != id {
			out = append(out, p)
		}
	}
	return saveCustomPresetsUnlocked(out)
}

// GetCustomPreset returns preset by id.
func GetCustomPreset(id string) (CustomPreset, bool) {
	list, err := LoadCustomPresets()
	if err != nil {
		return CustomPreset{}, false
	}
	id = slugID(id)
	for _, p := range list {
		if p.ID == id {
			return p, true
		}
	}
	return CustomPreset{}, false
}

// ApplyCustomPreset maps custom preset → AccessPolicy.
func ApplyCustomPreset(p CustomPreset, base AccessPolicy) AccessPolicy {
	out := base
	out.AllowInternet = p.AllowInternet
	out.ServicesMode = p.ServicesMode
	out.Services = append([]string{}, p.Services...)
	out.Normalize()
	return out
}

// ApplyNamedPreset tries built-in first, then custom id.
func ApplyNamedPreset(name string, base AccessPolicy) AccessPolicy {
	n := strings.ToLower(strings.TrimSpace(name))
	switch n {
	case PresetFull, "all", PresetMedia, PresetNone, "internet":
		return ApplyPreset(n, base)
	}
	if cp, ok := GetCustomPreset(name); ok {
		return ApplyCustomPreset(cp, base)
	}
	return ApplyPreset(name, base)
}
