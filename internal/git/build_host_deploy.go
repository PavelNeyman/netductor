package gitstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
)

type BuildHostState struct {
	ID           string  `json:"id"`
	Label        string  `json:"label,omitempty"`
	SSHHost      string  `json:"ssh_host"`
	SSHPort      int     `json:"ssh_port"`
	SSHUser      string  `json:"ssh_user"`
	Token        string  `json:"token,omitempty"`
	AgentVersion string  `json:"agent_version,omitempty"`
	Online       bool    `json:"online"`
	LastSeen     string  `json:"last_seen,omitempty"`
	Uname        string  `json:"uname,omitempty"`
	Docker       bool    `json:"docker"`
	DiskFreeGB   float64 `json:"disk_free_gb,omitempty"`
	LastError    string  `json:"last_error,omitempty"`
	DeployedAt   string  `json:"deployed_at,omitempty"`
	LastJobID    string  `json:"last_job_id,omitempty"`
	JobsDone     int     `json:"jobs_done,omitempty"`
}

func buildHostsStateDir() string {
	return filepath.Join(paths.StateDir(), "build-hosts")
}

func buildHostStatePath(id string) string {
	return filepath.Join(buildHostsStateDir(), sanitize(id)+".json")
}

func SaveBuildHostState(s BuildHostState) error {
	_ = os.MkdirAll(buildHostsStateDir(), 0o700)
	if s.ID == "" {
		s.ID = sanitize(s.SSHHost)
	}
	b, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(buildHostStatePath(s.ID), append(b, '\n'), 0o600)
}

func LoadBuildHostState(id string) (BuildHostState, error) {
	b, err := os.ReadFile(buildHostStatePath(sanitize(id)))
	if err != nil {
		return BuildHostState{}, err
	}
	var s BuildHostState
	err = json.Unmarshal(b, &s)
	return s, err
}

func PrimaryBuildHostOnline() bool {
	for _, s := range ListBuildHostStates() {
		if s.Online {
			return true
		}
	}
	return false
}

func ListBuildHostStates() []BuildHostState {
	ents, err := os.ReadDir(buildHostsStateDir())
	if err != nil {
		return nil
	}
	var out []BuildHostState
	now := time.Now().UTC()
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(buildHostsStateDir(), e.Name()))
		if err != nil {
			continue
		}
		var s BuildHostState
		if json.Unmarshal(b, &s) != nil {
			continue
		}
		s.Online = false
		if s.LastSeen != "" {
			if t, err := time.Parse(time.RFC3339, s.LastSeen); err == nil && now.Sub(t) < 90*time.Second {
				s.Online = true
			}
		}
		out = append(out, s)
	}
	return out
}

func RecordBuildHostHeartbeat(token string, facts map[string]any) error {
	for _, s := range ListBuildHostStates() {
		// reload with token
		full, err := LoadBuildHostState(s.ID)
		if err != nil || full.Token == "" || full.Token != token {
			continue
		}
		full.LastSeen = time.Now().UTC().Format(time.RFC3339)
		full.Online = true
		if v, ok := facts["agent_version"].(string); ok {
			full.AgentVersion = v
		}
		if v, ok := facts["uname"].(string); ok {
			full.Uname = v
		}
		if v, ok := facts["docker"].(bool); ok {
			full.Docker = v
		}
		if v, ok := facts["disk_free_gb"].(float64); ok {
			full.DiskFreeGB = v
		}
		return SaveBuildHostState(full)
	}
	return fmt.Errorf("unknown build-host token")
}

// WatchBuildHosts refreshes online flags from LastSeen (agent heartbeat over tunnel).
// Does not SSH from primary to build-host.
func WatchBuildHosts() {
	const stale = 3 * time.Minute
	now := time.Now().UTC()
	for _, s := range ListBuildHostStates() {
		full, err := LoadBuildHostState(s.ID)
		if err != nil {
			continue
		}
		staleHost := true
		if full.LastSeen != "" {
			if ts, e := time.Parse(time.RFC3339, full.LastSeen); e == nil {
				staleHost = now.Sub(ts) > stale
			}
		}
		was := full.Online
		full.Online = !staleHost && full.LastSeen != ""
		if was && !full.Online {
			notify.AlertOnce("build-host:offline:"+full.ID, fmt.Sprintf("⚠️ Build-host <b>%s</b> heartbeat stale", full.Label))
		}
		if full.DiskFreeGB > 0 && full.DiskFreeGB < 5 {
			notify.AlertOnce("build-host:disk:"+full.ID, fmt.Sprintf("⚠️ Build-host <b>%s</b> disk %.1fG", full.Label, full.DiskFreeGB))
		}
		_ = SaveBuildHostState(full)
	}
}
