package main

import (
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/controlplane"
	"github.com/PavelNeyman/netductor/internal/metrics"
	"github.com/PavelNeyman/netductor/internal/version"
)

func registerNodeAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/node/status", handleNodeStatus)
	mux.HandleFunc("/api/node/leader", handleNodeLeader)
}

// handleNodeStatus is Phase 4 thin local read: version, control role, services.
// Requires operator session (no unauthenticated cluster surface).
func handleNodeStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	if !requireSession(w, r) {
		return
	}
	hn, _ := os.Hostname()
	role := controlplane.Get()
	writeJSON(w, 200, map[string]any{
		"ok":            true,
		"hostname":      hn,
		"version":       version.Release,
		"control_role":  role,
		"write_allowed": controlplane.WriteAllowed(),
		"is_leader":     controlplane.WriteAllowed(),
		"services":      nodeServiceSnapshot(),
		"metrics":       metrics.Collect(),
		"ts":            time.Now().UTC().Format(time.RFC3339),
	})
}

// handleNodeLeader returns whether this node accepts SoT writes (active control).
func handleNodeLeader(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, 405, map[string]string{"error": "method"})
		return
	}
	// Public-ish probe: no secrets; useful for op routing. Still rate-limit via network posture.
	writeJSON(w, 200, map[string]any{
		"control_role":  controlplane.Get(),
		"write_allowed": controlplane.WriteAllowed(),
		"is_leader":     controlplane.WriteAllowed(),
	})
}

func nodeServiceSnapshot() map[string]string {
	units := []string{
		"netductor-api", "netductor-telegram-bot", "sing-box", "blocky",
		"netductor-redirect", "netductor-secondary-agent",
	}
	out := make(map[string]string, len(units))
	for _, u := range units {
		b, err := exec.Command("systemctl", "is-active", u).Output()
		st := strings.TrimSpace(string(b))
		if err != nil && st == "" {
			st = "inactive"
		}
		out[u] = st
	}
	return out
}
