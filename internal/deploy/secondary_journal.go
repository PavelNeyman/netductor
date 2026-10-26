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
)

// Orchestration status (separate from last successful stage).
const (
	StatusRunning           = "running"
	StatusFailedRecoverable = "failed_recoverable"
	StatusComplete          = "complete"
)

// deployOrchState is durable orchestration state on the operator machine.
// Private keys / tokens are never stored here.
type deployOrchState struct {
	PeerID              string    `json:"peer_id"`
	AgentID             string    `json:"agent_id,omitempty"`
	Host                string    `json:"host"`
	LastSuccessfulStage string    `json:"last_successful_stage"`
	Status              string    `json:"status"`
	LastError           string    `json:"last_error,omitempty"`
	MaterialSHA         string    `json:"material_sha,omitempty"` // sha256 of ship JSON (identity)
	UpdatedAt           time.Time `json:"updated_at"`
	// Legacy Stage kept for reading old journals; prefer LastSuccessfulStage.
	Stage string `json:"stage,omitempty"`
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
	path := deployStatePath(host)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("deploy state read %s: %w", path, err)
	}
	var st deployOrchState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, fmt.Errorf("deploy state corrupt at %s: %w (fix or remove file; refuse blind re-provision)", path, err)
	}
	if st.Host == "" {
		st.Host = host
	}
	// Migrate legacy single-stage journals.
	if st.LastSuccessfulStage == "" && st.Stage != "" && st.Stage != StageDeployComplete {
		if st.Stage == "failed_recoverable" {
			st.Status = StatusFailedRecoverable
		} else {
			st.LastSuccessfulStage = st.Stage
			if st.Status == "" {
				st.Status = StatusRunning
			}
		}
	}
	if st.Status == "" {
		st.Status = StatusRunning
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
		return fmt.Errorf("deploy state mkdir: %w", err)
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	path := deployStatePath(st.Host)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("deploy state write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("deploy state rename: %w", err)
	}
	return nil
}

func clearDeployState(host string) error {
	path := deployStatePath(host)
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("clear deploy state: %w", err)
	}
	return nil
}

// mergeOrchState updates st with non-empty IDs and stage/status; preserves prior IDs.
func mergeOrchState(prev *deployOrchState, host, peerID, agentID, successStage, status, lastErr, materialSHA string) *deployOrchState {
	st := &deployOrchState{Host: host, Status: StatusRunning}
	if prev != nil {
		cp := *prev
		st = &cp
		st.Host = host
	}
	if peerID != "" {
		st.PeerID = peerID
	}
	if agentID != "" {
		st.AgentID = agentID
	}
	if successStage != "" {
		st.LastSuccessfulStage = successStage
		st.Stage = successStage
	}
	if status != "" {
		st.Status = status
	}
	if materialSHA != "" {
		st.MaterialSHA = materialSHA
	}
	st.LastError = lastErr
	return st
}

// checkpointSuccess records a confirmed successful stage. Blocking if state cannot be saved.
func checkpointSuccess(prev **deployOrchState, peerID, agentID, host, stage string) error {
	st := mergeOrchState(*prev, host, peerID, agentID, stage, StatusRunning, "", "")
	_ = appendDeployLog(st.PeerID, st.AgentID, host, stage, "")
	if err := saveDeployState(st); err != nil {
		return fmt.Errorf("deploy checkpoint %s: %w", stage, err)
	}
	*prev = st
	return nil
}

// checkpointSuccessWithSHA records stage and material ship fingerprint.
func checkpointSuccessWithSHA(prev **deployOrchState, peerID, agentID, host, stage, materialSHA string) error {
	st := mergeOrchState(*prev, host, peerID, agentID, stage, StatusRunning, "", materialSHA)
	_ = appendDeployLog(st.PeerID, st.AgentID, host, stage, "")
	if err := saveDeployState(st); err != nil {
		return fmt.Errorf("deploy checkpoint %s: %w", stage, err)
	}
	*prev = st
	return nil
}

// checkpointFailure marks status failed without clearing last_successful_stage.

func checkpointFailure(prev **deployOrchState, peerID, agentID, host, lastErr string) error {
	st := mergeOrchState(*prev, host, peerID, agentID, "", StatusFailedRecoverable, lastErr, "")
	_ = appendDeployLog(st.PeerID, st.AgentID, host, StatusFailedRecoverable, lastErr)
	if err := saveDeployState(st); err != nil {
		return fmt.Errorf("deploy failure checkpoint: %w", err)
	}
	*prev = st
	return nil
}

func appendDeployLog(peerID, agentID, host, stage, lastErr string) error {
	dir := deployJournalDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\n",
		time.Now().UTC().Format(time.RFC3339), stage, peerID, agentID, host, lastErr)
	f, err := os.OpenFile(filepath.Join(dir, "journal.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, werr := f.WriteString(line)
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	return cerr
}

func effectiveStage(st *deployOrchState) string {
	if st == nil {
		return ""
	}
	if st.LastSuccessfulStage != "" {
		return st.LastSuccessfulStage
	}
	return st.Stage
}

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
	return -1
}

func canSkipProvision(st *deployOrchState) bool {
	return stageRank(effectiveStage(st)) >= stageRank(StageProvisioned)
}

func canSkipPeerAdd(st *deployOrchState) bool {
	return stageRank(effectiveStage(st)) >= stageRank(StagePeerPrepared)
}

func canSkipMaterialXfer(st *deployOrchState) bool {
	return stageRank(effectiveStage(st)) >= stageRank(StageMaterialTransferred)
}

func canSkipBootstrapB(st *deployOrchState) bool {
	return stageRank(effectiveStage(st)) >= stageRank(StageSecondaryBootstrapped)
}

// Legacy helpers for older call sites / tests.
func writeDeployJournal(peerID, agentID, host, stage string) {
	var prev *deployOrchState
	_ = checkpointSuccess(&prev, peerID, agentID, host, stage)
}

func writeDeployJournalErr(lastErr, peerID, agentID, host, _ string) {
	var prev *deployOrchState
	_ = checkpointFailure(&prev, peerID, agentID, host, lastErr)
}
