package install

import (
	"encoding/json"
	"os"
	"os/exec"
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
	order := []string{"dirs", "hardening", "singbox", "blocky", "vpn-users", "api", "metrics", "telegram", "backup", "lampac", "git", "registry"}
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

// MarkComponentRemoved drops a component from the manifest (after uninstall).
func MarkComponentRemoved(name string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	prev := ReadComponentsManifest()
	out := make([]string, 0, len(prev))
	for _, c := range prev {
		if c != name {
			out = append(out, c)
		}
	}
	_ = os.MkdirAll(paths.StateDir(), 0o755)
	m := ComponentsManifest{
		Version:    version.Release,
		UpdatedAt:  time.Now().UTC().Format(time.RFC3339),
		Components: out,
	}
	raw, _ := json.MarshalIndent(m, "", "  ")
	_ = os.WriteFile(componentsPath(), append(raw, '\n'), 0o644)
}

// SyncComponentsFromDisk adds components that are present on disk / running.
// Called before every backup so manifest tracks install/uninstall without manual lists.
func SyncComponentsFromDisk() {
	var add []string
	if _, err := os.Stat(filepath.Join(paths.OptDir(), "lampac")); err == nil {
		add = append(add, "lampac")
	}
	if _, err := os.Stat(filepath.Join(paths.StateDir(), "git")); err == nil {
		// non-empty root
		if ents, e := os.ReadDir(filepath.Join(paths.StateDir(), "git")); e == nil && len(ents) > 0 {
			add = append(add, "git")
		}
	}
	if _, err := os.Stat(filepath.Join(paths.StateDir(), "registry")); err == nil {
		add = append(add, "registry")
	}
	// docker containers
	if out, err := dockerPsNames(); err == nil {
		if strings.Contains(out, "netductor-lampac") {
			add = append(add, "lampac")
		}
		if strings.Contains(out, "netductor-registry") {
			add = append(add, "registry")
		}
	}
	if len(add) > 0 {
		_ = WriteComponentsManifest(add)
	}
	// remove if gone
	prev := ReadComponentsManifest()
	for _, c := range prev {
		switch c {
		case "lampac":
			if _, err := os.Stat(filepath.Join(paths.OptDir(), "lampac")); err != nil {
				if out, _ := dockerPsNames(); !strings.Contains(out, "netductor-lampac") {
					MarkComponentRemoved("lampac")
				}
			}
		case "git":
			ents, err := os.ReadDir(filepath.Join(paths.StateDir(), "git"))
			if err != nil || len(ents) == 0 {
				MarkComponentRemoved("git")
			}
		case "registry":
			if out, _ := dockerPsNames(); !strings.Contains(out, "netductor-registry") {
				if _, err := os.Stat(filepath.Join(paths.StateDir(), "registry")); err != nil {
					MarkComponentRemoved("registry")
				}
			}
		}
	}
}

func dockerPsNames() (string, error) {
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}").CombinedOutput()
	return string(out), err
}
