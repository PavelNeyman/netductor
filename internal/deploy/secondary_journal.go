package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Deploy stages for secondary orchestration journal (no secrets).
const (
	StageProvisionStarted      = "provision_started"
	StageProvisioned           = "provisioned"
	StagePeerAddStarted        = "peer_add_started"
	StagePeerPrepared          = "peer_prepared"
	StageMaterialTransferred   = "material_transferred"
	StageSecondaryBootstrapped = "secondary_bootstrapped"
	StageFirewallVerified      = "firewall_verified"
	StageNetworkVerified       = "network_verified"
	StageHeartbeatVerified     = "heartbeat_verified"
	StageDeployComplete        = "deploy_complete"
	StageFailedRecoverable     = "failed_recoverable"
)

// deployOrchState is durable orchestration state on the operator machine.
// Private keys / tokens are never stored here.
type deployOrchState struct {
	PeerID    string    `json:"peer_id"`
	AgentID   string    `json:"agent_id,omitempty"`
	Host      string    `json:"host"`
	Stage     string    `json:"stage"`
	UpdatedAt time.Time `json:"updated_at"`
	LastError string    `json:"last_error,omitempty"`
}

func deployJournalDir() string {
	home := os.Getenv("HOME")
	if home == "" {
		return filepath.Join(os.TempDir(), "netductor-deploy")
	}
	return filepath.Join(home, ".cache", "netductor", "deploy")
}

func deployStatePath(host string) string {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			return r
		}
		return '_'
	}, host)
	return filepath.Join(deployJournalDir(), "state-"+safe+".json")
}

func loadDeployState(host string) (*deployOrchState, error) {
	b, err := os.ReadFile(deployStatePath(host))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var st deployOrchState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, fmt.Errorf("deploy state corrupt: %w", err)
	}
	return &st, nil
}

func saveDeployState(st *deployOrchState) error {
	if st == nil || st.Host == "" {
		return fmt.Errorf("empty deploy state")
	}
	st.UpdatedAt = time.Now().UTC()
	dir := deployJournalDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	path := deployStatePath(st.Host)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func clearDeployState(host string) {
	_ = os.Remove(deployStatePath(host))
}

// writeDeployJournal appends a line and updates structured state (no secrets).
func writeDeployJournal(peerID, agentID, host, stage string) {
	writeDeployJournalErr("", peerID, agentID, host, stage)
}

func writeDeployJournalErr(lastErr, peerID, agentID, host, stage string) {
	dir := deployJournalDir()
	_ = os.MkdirAll(dir, 0o700)
	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\n", time.Now().UTC().Format(time.RFC3339), stage, peerID, agentID, host)
	f, err := os.OpenFile(filepath.Join(dir, "journal.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err == nil {
		_, _ = f.WriteString(line)
		_ = f.Close()
	}
	st := &deployOrchState{PeerID: peerID, AgentID: agentID, Host: host, Stage: stage, LastError: lastErr}
	_ = saveDeployState(st)
}

// stageRank orders stages for resume decisions (higher = further along).
func stageRank(s string) int {
	order := []string{
		StageProvisionStarted,
		StageProvisioned,
		StagePeerAddStarted,
		StagePeerPrepared,
		StageMaterialTransferred,
		StageSecondaryBootstrapped,
		StageFirewallVerified,
		StageNetworkVerified,
		StageHeartbeatVerified,
		StageDeployComplete,
	}
	for i, x := range order {
		if x == s {
			return i
		}
	}
	if s == StageFailedRecoverable {
		return -1
	}
	return -1
}

func canSkipProvision(st *deployOrchState) bool {
	if st == nil {
		return false
	}
	return stageRank(st.Stage) >= stageRank(StageProvisioned)
}

func canSkipPeerAdd(st *deployOrchState) bool {
	if st == nil {
		return false
	}
	return stageRank(st.Stage) >= stageRank(StagePeerPrepared)
}

func canSkipMaterialXfer(st *deployOrchState) bool {
	if st == nil {
		return false
	}
	return stageRank(st.Stage) >= stageRank(StageMaterialTransferred)
}

func canSkipBootstrapB(st *deployOrchState) bool {
	if st == nil {
		return false
	}
	return stageRank(st.Stage) >= stageRank(StageSecondaryBootstrapped)
}
