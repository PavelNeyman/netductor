package svcpaths

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	policyPath = "/etc/netductor/svc-paths/failover-policy.json"
	statePath  = "/var/lib/netductor/failover-state.json"
)

// Policy controls auto actions (defaults conservative).
type Policy struct {
	Enabled            bool `json:"enabled"`
	UsersToSPOnVLESSDown bool `json:"users_to_sp_on_vless_down"`
	FailThreshold      int  `json:"fail_threshold"` // consecutive fails → DOWN
	OKThreshold        int  `json:"ok_threshold"`
	StableSeconds      int  `json:"stable_seconds"` // failback wait
}

func DefaultPolicy() Policy {
	return Policy{
		Enabled:            true,
		UsersToSPOnVLESSDown: false, // explicit opt-in
		FailThreshold:      3,
		OKThreshold:        3,
		StableSeconds:      45,
	}
}

func LoadPolicy() Policy {
	p := DefaultPolicy()
	b, err := os.ReadFile(policyPath)
	if err != nil {
		return p
	}
	_ = json.Unmarshal(b, &p)
	if p.FailThreshold < 1 {
		p.FailThreshold = 3
	}
	if p.OKThreshold < 1 {
		p.OKThreshold = 3
	}
	if p.StableSeconds < 10 {
		p.StableSeconds = 30
	}
	return p
}

func SavePolicy(p Policy) error {
	_ = os.MkdirAll(filepath.Dir(policyPath), 0o755)
	b, _ := json.MarshalIndent(p, "", "  ")
	return os.WriteFile(policyPath, b, 0o644)
}

// State is persisted anti-flap counters and last decision.
type State struct {
	SPFails    int       `json:"sp_fails"`
	SPOks      int       `json:"sp_oks"`
	PSFails    int       `json:"ps_fails"`
	PSOks      int       `json:"ps_oks"`
	VLESSFails int       `json:"vless_fails"`
	VLESSOks   int       `json:"vless_oks"`
	SPDown     bool      `json:"sp_down"`
	PSDown     bool      `json:"ps_down"`
	VLESSDown  bool      `json:"vless_down"`
	UserPath   string    `json:"user_path"` // "vless" | "sp" | "degraded"
	AgentURLMode string  `json:"agent_url_mode"` // "tunnel" | "public"
	UpdatedAt  time.Time `json:"updated_at"`
	LastAction string    `json:"last_action,omitempty"`
}

func LoadState() State {
	var s State
	b, err := os.ReadFile(statePath)
	if err != nil {
		s.UserPath = "vless"
		s.AgentURLMode = "tunnel"
		return s
	}
	_ = json.Unmarshal(b, &s)
	return s
}

func saveState(s State) error {
	_ = os.MkdirAll(filepath.Dir(statePath), 0o755)
	s.UpdatedAt = time.Now().UTC()
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(statePath, b, 0o644)
}

type healthSnap struct {
	SP    int `json:"svc_sp_up"`
	PS    int `json:"svc_ps_up"`
	VLESS int `json:"vless_tcp443"`
	Role  string `json:"role"`
}

func readHealthFile() healthSnap {
	var h healthSnap
	b, err := os.ReadFile(HealthJSON)
	if err != nil {
		return h
	}
	_ = json.Unmarshal(b, &h)
	return h
}

// Tick updates anti-flap state from health JSON. Returns human summary.
// Does not move user traffic by itself when UsersToSPOnVLESSDown is false.
func Tick() (State, string, error) {
	p := LoadPolicy()
	s := LoadState()
	h := readHealthFile()

	upd := func(up int, fails, oks *int, down *bool) {
		if up == 1 {
			*oks++
			*fails = 0
			if *oks >= p.OKThreshold {
				*down = false
			}
		} else {
			*fails++
			*oks = 0
			if *fails >= p.FailThreshold {
				*down = true
			}
		}
	}
	upd(h.SP, &s.SPFails, &s.SPOks, &s.SPDown)
	upd(h.PS, &s.PSFails, &s.PSOks, &s.PSDown)
	if h.Role == "secondary" || h.VLESS != 0 || h.Role == "primary" {
		// on primary health file has vless_tcp443=0 always; only secondary measures
		if h.Role == "secondary" {
			upd(h.VLESS, &s.VLESSFails, &s.VLESSOks, &s.VLESSDown)
		}
	}

	action := "none"
	if !p.Enabled {
		s.LastAction = "disabled"
		_ = saveState(s)
		return s, "failover disabled", nil
	}

	// Desired user path
	prevUser := s.UserPath
	if s.VLESSDown {
		if p.UsersToSPOnVLESSDown && !s.SPDown {
			s.UserPath = "sp"
		} else if p.UsersToSPOnVLESSDown && s.SPDown && !s.PSDown {
			s.UserPath = "ps"
		} else {
			s.UserPath = "degraded"
		}
	} else {
		s.UserPath = "vless"
	}
	if s.UserPath != prevUser {
		action = fmt.Sprintf("user_path %s→%s", prevUser, s.UserPath)
	}

	// Agent URL mode (secondary reads this via desired file)
	prevAgent := s.AgentURLMode
	if s.SPDown {
		s.AgentURLMode = "public"
	} else {
		s.AgentURLMode = "tunnel"
	}
	if s.AgentURLMode != prevAgent {
		if action == "none" {
			action = fmt.Sprintf("agent %s→%s", prevAgent, s.AgentURLMode)
		} else {
			action += "; agent " + prevAgent + "→" + s.AgentURLMode
		}
	}

	s.LastAction = action
	_ = saveState(s)

	// Write desired for secondary scripts / agent
	desired := map[string]any{
		"user_path":      s.UserPath,
		"agent_url_mode": s.AgentURLMode,
		"sp_down":        s.SPDown,
		"ps_down":        s.PSDown,
		"vless_down":     s.VLESSDown,
		"policy":         p,
		"ts":             time.Now().UTC().Format(time.RFC3339),
	}
	db, _ := json.MarshalIndent(desired, "", "  ")
	_ = os.MkdirAll("/var/lib/netductor", 0o755)
	_ = os.WriteFile("/var/lib/netductor/failover-desired.json", db, 0o644)

	sum := fmt.Sprintf("sp_down=%v ps_down=%v vless_down=%v user=%s agent=%s action=%s",
		s.SPDown, s.PSDown, s.VLESSDown, s.UserPath, s.AgentURLMode, action)
	return s, sum, nil
}
