package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// ComponentsManifest lists what was installed on this core (for recover).
type ComponentsManifest struct {
	Version    string   `json:"version"`
	UpdatedAt  string   `json:"updated_at"`
	Components []string `json:"components"`
}

func componentsPath() string {
	return filepath.Join(paths.StateDir(), "components.json")
}

// WriteComponentsManifest records installed set (merge with existing).
func WriteComponentsManifest(comps []string) error {
	_ = os.MkdirAll(paths.StateDir(), 0o755)
	prev := ReadComponentsManifest()
	set := map[string]struct{}{}
	for _, c := range prev {
		set[c] = struct{}{}
	}
	for _, c := range comps {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		set[c] = struct{}{}
	}
	// always ensure baseline dirs
	set["dirs"] = struct{}{}
	out := make([]string, 0, len(set))
	// stable preferred order
	order := []string{"dirs", "hardening", "singbox", "blocky", "vpn-users", "api", "metrics", "telegram", "backup", "lampac"}
	seen := map[string]bool{}
	for _, c := range order {
		if _, ok := set[c]; ok {
			out = append(out, c)
			seen[c] = true
		}
	}
	for c := range set {
		if !seen[c] {
			out = append(out, c)
		}
	}
	m := ComponentsManifest{
		Version:    version.Release,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
		Components: out,
	}
	raw, _ := json.MarshalIndent(m, "", "  ")
	return os.WriteFile(componentsPath(), append(raw, '\n'), 0o644)
}

// ReadComponentsManifest returns component list or default core set.
func ReadComponentsManifest() []string {
	b, err := os.ReadFile(componentsPath())
	if err != nil {
		return nil
	}
	var m ComponentsManifest
	if json.Unmarshal(b, &m) != nil || len(m.Components) == 0 {
		return nil
	}
	return m.Components
}

// MarkComponentInstalled adds one component to the manifest.
func MarkComponentInstalled(name string) {
	_ = WriteComponentsManifest([]string{name})
}
