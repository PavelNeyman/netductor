package controlplane

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SoTEntry is a path that must exist (or be restorable) on control for promote readiness.
type SoTEntry struct {
	Rel     string // under state or etc
	Kind    string // file|dir
	Require bool   // hard fail on check-local if missing
	Note    string
}

// RequiredSoT lists minimum datasets from PHASE3 notes (relative layout hints).
func RequiredSoT() []SoTEntry {
	etc := paths.EtcDir()
	state := paths.StateDir()
	return []SoTEntry{
		{Rel: filepath.Join(etc, "vpn-users.json"), Kind: "file", Require: true, Note: "users"},
		{Rel: filepath.Join(etc, "secrets"), Kind: "dir", Require: true, Note: "CA/Reality/TG keys"},
		{Rel: filepath.Join(state, "secondary", "devices.json"), Kind: "file", Require: false, Note: "egress registry (empty ok if no secondary yet)"},
		{Rel: filepath.Join(etc, "svc-paths"), Kind: "dir", Require: false, Note: "peer material"},
		{Rel: filepath.Join(etc, "netductor.conf"), Kind: "file", Require: false, Note: "domain/redirect"},
	}
}

// CheckLocalSoT reports missing required paths on this node (post-restore readiness).
func CheckLocalSoT() (ok bool, lines []string) {
	ok = true
	for _, e := range RequiredSoT() {
		st, err := os.Stat(e.Rel)
		if err != nil {
			msg := fmt.Sprintf("MISSING %s (%s)", e.Rel, e.Note)
			if e.Require {
				ok = false
				lines = append(lines, "FAIL "+msg)
			} else {
				lines = append(lines, "WARN "+msg)
			}
			continue
		}
		if e.Kind == "dir" && !st.IsDir() {
			ok = false
			lines = append(lines, "FAIL not a dir: "+e.Rel)
			continue
		}
		if e.Kind == "file" && st.IsDir() {
			ok = false
			lines = append(lines, "FAIL not a file: "+e.Rel)
			continue
		}
		lines = append(lines, "OK   "+e.Rel+" — "+e.Note)
	}
	return ok, lines
}

// EndpointsPath optional operator-managed list for docs/agents (one URL per line).
func EndpointsPath() string {
	return filepath.Join(paths.EtcDir(), "control_endpoints")
}

// ReadEndpoints returns non-empty trimmed lines from control_endpoints file.
func ReadEndpoints() []string {
	b, err := os.ReadFile(EndpointsPath())
	if err != nil {
		return nil
	}
	var out []string
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out
}
