package upstream

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Pin records expected major/version for an external binary.
type Pin struct {
	Name    string `json:"name"` // sing-box | blocky
	Version string `json:"version,omitempty"`
	Note    string `json:"note,omitempty"`
}

type file struct {
	Pins []Pin `json:"pins"`
}

func path() string {
	return filepath.Join(paths.EtcDir(), "upstream-pins.json")
}

func Load() []Pin {
	b, err := os.ReadFile(path())
	if err != nil {
		return defaultPins()
	}
	var f file
	if json.Unmarshal(b, &f) != nil || len(f.Pins) == 0 {
		return defaultPins()
	}
	return f.Pins
}

func defaultPins() []Pin {
	return []Pin{
		{Name: "sing-box", Note: "set version after install"},
		{Name: "blocky", Note: "set version after install"},
	}
}

func Save(pins []Pin) error {
	_ = os.MkdirAll(filepath.Dir(path()), 0o700)
	b, _ := json.MarshalIndent(file{Pins: pins}, "", "  ")
	return os.WriteFile(path(), append(b, '\n'), 0o600)
}

// InstalledVersion best-effort --version parse.
func InstalledVersion(name string) string {
	bin := name
	switch name {
	case "sing-box":
		bin = "sing-box"
	case "blocky":
		bin = "blocky"
	}
	out, err := exec.Command(bin, "version").CombinedOutput()
	if err != nil {
		out, err = exec.Command(bin, "--version").CombinedOutput()
		if err != nil {
			return ""
		}
	}
	s := strings.TrimSpace(string(out))
	// first line token that looks like a version
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// "sing-box version 1.10.0" or "blocky version: v0.9"
		fields := strings.Fields(line)
		for _, f := range fields {
			f = strings.TrimPrefix(f, "v")
			if len(f) > 0 && f[0] >= '0' && f[0] <= '9' && strings.Contains(f, ".") {
				return f
			}
		}
		return line
	}
	return s
}

// Check returns human lines for doctor.
func Check() []string {
	var lines []string
	for _, p := range Load() {
		got := InstalledVersion(p.Name)
		if p.Version == "" {
			lines = append(lines, fmt.Sprintf("INFO upstream %s installed=%q (no pin)", p.Name, got))
			continue
		}
		if got == "" {
			lines = append(lines, fmt.Sprintf("WARN upstream %s pin=%s installed=(missing)", p.Name, p.Version))
			continue
		}
		if strings.HasPrefix(got, p.Version) || strings.Contains(got, p.Version) {
			lines = append(lines, fmt.Sprintf("OK upstream %s pin=%s installed=%s", p.Name, p.Version, got))
		} else {
			lines = append(lines, fmt.Sprintf("WARN upstream %s pin=%s installed=%s (drift)", p.Name, p.Version, got))
		}
	}
	return lines
}
