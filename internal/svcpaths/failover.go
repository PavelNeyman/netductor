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
	Enabled              bool `json:"enabled"`
	UsersToSPOnVLESSDown bool `json:"users_to_sp_on_vless_down"`
	FailThreshold        int  `json:"fail_threshold"` // consecutive fails → DOWN
	OKThreshold          int  `json:"ok_threshold"`
	StableSeconds        int  `json:"stable_seconds"` // failback wait
}

func DefaultPolicy() Policy {
	return Policy{
		Enabled:              true,
		UsersToSPOnVLESSDown: true,
		FailThreshold:        3,
		OKThreshold:          3,
		StableSeconds:        45,
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
	SPFails      int       `json:"sp_fails"`
	SPOks        int       `json:"sp_oks"`
	PSFails      int       `json:"ps_fails"`
	PSOks        int       `json:"ps_oks"`
	VLESSFails   int       `json:"vless_fails"`
	VLESSOks     int       `json:"vless_oks"`
	SPDown       bool      `json:"sp_down"`
	PSDown       bool      `json:"ps_down"`
	VLESSDown    bool      `json:"vless_down"`
	UserPath     string    `json:"user_path"`      // "vless" | "sp" | "ps" | "degraded"
	AgentURLMode string    `json:"agent_url_mode"` // "tunnel" (SP) | "ps" (PS) — never public
	UpdatedAt    time.Time `json:"updated_at"`
	LastAction   string    `json:"last_action,omitempty"`
	// StableSince tracks continuous healthy time for failback (StableSeconds policy).
	SPStableSince    time.Time `json:"sp_stable_since,omitempty"`
	PSStableSince    time.Time `json:"ps_stable_since,omitempty"`
	VLESSStableSince time.Time `json:"vless_stable_since,omitempty"`
	LastSampleAt     time.Time `json:"last_sample_at,omitempty"`
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
	SP    int    `json:"svc_sp_up"`
	PS    int    `json:"svc_ps_up"`
	VLESS int    `json:"vless_tcp443"`
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

	now := time.Now().UTC()
	stableFor := time.Duration(p.StableSeconds) * time.Second
	// Min interval between counted samples (agent + timer fallback must not double-count).
	const minSample = 20 * time.Second
	if !s.LastSampleAt.IsZero() && now.Sub(s.LastSampleAt) < minSample {
		sum := fmt.Sprintf("sp_down=%v ps_down=%v vless_down=%v user=%s agent=%s action=sample_skipped",
			s.SPDown, s.PSDown, s.VLESSDown, s.UserPath, s.AgentURLMode)
		return s, sum, nil
	}
	s.LastSampleAt = now
	upd := func(up int, fails, oks *int, down *bool, stableSince *time.Time) {
		if up == 1 {
			*oks++
			*fails = 0
			if *down {
				if stableSince.IsZero() {
					*stableSince = now
				}
				if *oks >= p.OKThreshold && now.Sub(*stableSince) >= stableFor {
					*down = false
					*stableSince = time.Time{}
				}
			} else {
				*stableSince = time.Time{}
			}
		} else {
			*fails++
			*oks = 0
			*stableSince = time.Time{}
			if *fails >= p.FailThreshold {
				*down = true
			}
		}
	}
	upd(h.SP, &s.SPFails, &s.SPOks, &s.SPDown, &s.SPStableSince)
	upd(h.PS, &s.PSFails, &s.PSOks, &s.PSDown, &s.PSStableSince)
	if h.Role == "secondary" {
		upd(h.VLESS, &s.VLESSFails, &s.VLESSOks, &s.VLESSDown, &s.VLESSStableSince)
	}

	action := "none"
	if !p.Enabled {
		s.LastAction = "disabled"
		if err := saveState(s); err != nil {
			return s, "failover disabled", fmt.Errorf("save state: %w", err)
		}
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

	// Agent URL: tunnel-only. SP preferred; PS when SP down. Never public WAN.
	prevAgent := s.AgentURLMode
	if s.SPDown && !s.PSDown {
		s.AgentURLMode = "ps"
	} else if s.SPDown && s.PSDown {
		s.AgentURLMode = "ps" // still PS VIP; both down → agent offline alerts
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
	if err := saveState(s); err != nil {
		return s, action, fmt.Errorf("save state: %w", err)
	}

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
	db, err := json.MarshalIndent(desired, "", "  ")
	if err != nil {
		return s, action, err
	}
	if err := os.MkdirAll("/var/lib/netductor", 0o755); err != nil {
		return s, action, err
	}
	if err := os.WriteFile("/var/lib/netductor/failover-desired.json", db, 0o644); err != nil {
		return s, action, fmt.Errorf("write failover-desired: %w", err)
	}

	sum := fmt.Sprintf("sp_down=%v ps_down=%v vless_down=%v user=%s agent=%s action=%s",
		s.SPDown, s.PSDown, s.VLESSDown, s.UserPath, s.AgentURLMode, action)
	return s, sum, nil
}
