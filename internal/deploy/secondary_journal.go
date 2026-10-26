package deploy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
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

// DeployOrchState is durable orchestration state on the operator machine.
// Private keys / tokens are never stored here.
type DeployOrchState struct {
	PeerID                  string    `json:"peer_id"`
	AgentID                 string    `json:"agent_id,omitempty"`
	Host                    string    `json:"host"`
	LastSuccessfulStage     string    `json:"last_successful_stage"`
	CurrentStep             string    `json:"current_step,omitempty"` // in-progress op; does not regress last_successful_stage
	Status                  string    `json:"status"`
	LastError               string    `json:"last_error,omitempty"`
	MaterialSHA             string    `json:"material_sha,omitempty"`              // sha256 of ship JSON
	BootstrappedMaterialSHA string    `json:"bootstrapped_material_sha,omitempty"` // SHA for which bootstrap was confirmed
	UpdatedAt               time.Time `json:"updated_at"`
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

func loadDeployState(host string) (*DeployOrchState, error) {
	path := deployStatePath(host)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("deploy state read %s: %w", path, err)
	}
	var st DeployOrchState
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

func saveDeployState(st *DeployOrchState) error {
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
	tf, err := os.CreateTemp(dir, "state-*.tmp")
	if err != nil {
		return fmt.Errorf("deploy state temp: %w", err)
	}
	tmp := tf.Name()
	if _, err := tf.Write(b); err != nil {
		_ = tf.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("deploy state write: %w", err)
	}
	if err := tf.Sync(); err != nil {
		_ = tf.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("deploy state fsync: %w", err)
	}
	if err := tf.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("deploy state close: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("deploy state rename: %w", err)
	}
	// Best-effort directory fsync (durability of the rename).
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		_ = d.Close()
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
func mergeOrchState(prev *DeployOrchState, host, peerID, agentID, successStage, status, lastErr, materialSHA string) *DeployOrchState {
	st := &DeployOrchState{Host: host, Status: StatusRunning}
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
		// Never regress last_successful_stage (peer_add_started must not wipe material_transferred).
		if stageRank(successStage) >= stageRank(st.LastSuccessfulStage) {
			st.LastSuccessfulStage = successStage
			st.Stage = successStage
		}
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

// markCurrentStep records an in-progress operation without regressing last_successful_stage.
func markCurrentStep(prev **DeployOrchState, peerID, agentID, host, step string) error {
	st := mergeOrchState(*prev, host, peerID, agentID, "", StatusRunning, "", "")
	st.CurrentStep = step
	_ = appendDeployLog(st.PeerID, st.AgentID, host, "step:"+step, "")
	if err := saveDeployState(st); err != nil {
		return fmt.Errorf("deploy current step %s: %w", step, err)
	}
	*prev = st
	return nil
}

// checkpointSuccess records a confirmed successful stage. Blocking if state cannot be saved.
func checkpointSuccess(prev **DeployOrchState, peerID, agentID, host, stage string) error {
	st := mergeOrchState(*prev, host, peerID, agentID, stage, StatusRunning, "", "")
	st.CurrentStep = ""
	_ = appendDeployLog(st.PeerID, st.AgentID, host, stage, "")
	if err := saveDeployState(st); err != nil {
		return fmt.Errorf("deploy checkpoint %s: %w", stage, err)
	}
	*prev = st
	return nil
}

// checkpointSuccessWithSHA records stage and material ship fingerprint.
func checkpointSuccessWithSHA(prev **DeployOrchState, peerID, agentID, host, stage, materialSHA string) error {
	st := mergeOrchState(*prev, host, peerID, agentID, stage, StatusRunning, "", materialSHA)
	st.CurrentStep = ""
	// New/updated ship material invalidates prior bootstrap confirmation until re-bootstrap.
	if materialSHA != "" && st.BootstrappedMaterialSHA != "" && st.BootstrappedMaterialSHA != materialSHA {
		st.BootstrappedMaterialSHA = ""
	}
	_ = appendDeployLog(st.PeerID, st.AgentID, host, stage, "")
	if err := saveDeployState(st); err != nil {
		return fmt.Errorf("deploy checkpoint %s: %w", stage, err)
	}
	*prev = st
	return nil
}

// checkpointBootstrapOK records secondary_bootstrapped bound to this material SHA.
func checkpointBootstrapOK(prev **DeployOrchState, peerID, agentID, host, materialSHA string) error {
	st := mergeOrchState(*prev, host, peerID, agentID, StageSecondaryBootstrapped, StatusRunning, "", materialSHA)
	st.CurrentStep = ""
	if materialSHA != "" {
		st.BootstrappedMaterialSHA = materialSHA
	}
	_ = appendDeployLog(st.PeerID, st.AgentID, host, StageSecondaryBootstrapped, "")
	if err := saveDeployState(st); err != nil {
		return fmt.Errorf("deploy checkpoint bootstrap: %w", err)
	}
	*prev = st
	return nil
}

// checkpointFailure marks status failed without clearing last_successful_stage.

func checkpointFailure(prev **DeployOrchState, peerID, agentID, host, lastErr string) error {
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

func effectiveStage(st *DeployOrchState) string {
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

func canSkipProvision(st *DeployOrchState) bool {
	return stageRank(effectiveStage(st)) >= stageRank(StageProvisioned)
}

func canSkipPeerAdd(st *DeployOrchState) bool {
	return stageRank(effectiveStage(st)) >= stageRank(StagePeerPrepared)
}

func canSkipMaterialXfer(st *DeployOrchState) bool {
	return stageRank(effectiveStage(st)) >= stageRank(StageMaterialTransferred)
}

func canSkipBootstrapB(st *DeployOrchState) bool {
	return stageRank(effectiveStage(st)) >= stageRank(StageSecondaryBootstrapped)
}

// canSkipBootstrapForMaterial allows verify-only only when bootstrap was confirmed for this ship SHA.
func canSkipBootstrapForMaterial(st *DeployOrchState, shipSHA string) bool {
	if st == nil || shipSHA == "" {
		return false
	}
	if !canSkipBootstrapB(st) {
		return false
	}
	return st.BootstrappedMaterialSHA != "" && st.BootstrappedMaterialSHA == shipSHA
}

// Legacy helpers for older call sites / tests.
func writeDeployJournal(peerID, agentID, host, stage string) {
	var prev *DeployOrchState
	_ = checkpointSuccess(&prev, peerID, agentID, host, stage)
}

func writeDeployJournalErr(lastErr, peerID, agentID, host, _ string) {
	var prev *DeployOrchState
	_ = checkpointFailure(&prev, peerID, agentID, host, lastErr)
}

// ListDeployStates returns all state-*.json under the journal dir (may be empty).
func ListDeployStates() ([]*DeployOrchState, error) {
	dir := deployJournalDir()
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*DeployOrchState
	for _, e := range ents {
		name := e.Name()
		if !strings.HasPrefix(name, "state-") || !strings.HasSuffix(name, ".json") {
			continue
		}
		host := strings.TrimSuffix(strings.TrimPrefix(name, "state-"), ".json")
		// Host may have been sanitized; load by reconstructed path via ReadFile of full path.
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, err
		}
		var st DeployOrchState
		if err := json.Unmarshal(b, &st); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		if st.Host == "" {
			st.Host = host
		}
		out = append(out, &st)
	}
	return out, nil
}

// FormatDeployStateLine is one-line status for CLI.
func FormatDeployStateLine(st *DeployOrchState) string {
	if st == nil {
		return "(nil)"
	}
	return fmt.Sprintf("host=%s peer=%s agent=%s last_ok=%s status=%s err=%s sha=%s",
		st.Host, st.PeerID, st.AgentID, effectiveStage(st), st.Status, st.LastError, shortSHA(st.MaterialSHA))
}

func shortSHA(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12] + "…"
}

// ClearDeployJournal is the public clear for CLI (host IP or sanitized name).
// Uses the same host lock as DeploySecondary so clear cannot race an active deploy.
func ClearDeployJournal(host string) error {
	return withHostDeployLock(host, func() error {
		return clearDeployState(host)
	})
}

// LoadDeployJournal is the public load for CLI.
func LoadDeployJournal(host string) (*DeployOrchState, error) {
	return loadDeployState(host)
}

func hostLockPath(host string) string {
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' {
			return r
		}
		return '_'
	}, host)
	return filepath.Join(deployJournalDir(), "lock-"+safe)
}

// withHostDeployLock serializes DeploySecondary for one secondary host on the operator machine.
func withHostDeployLock(host string, fn func() error) error {
	if host == "" {
		return fn()
	}
	dir := deployJournalDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("deploy lock mkdir: %w", err)
	}
	f, err := os.OpenFile(hostLockPath(host), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return fmt.Errorf("deploy lock open: %w", err)
	}
	defer f.Close()
	if err := unix.Flock(int(f.Fd()), unix.LOCK_EX); err != nil {
		return fmt.Errorf("deploy lock: %w", err)
	}
	defer unix.Flock(int(f.Fd()), unix.LOCK_UN)
	return fn()
}

// failDeploy records failed_recoverable (preserving last success) and returns the original error.
func failDeploy(prev **DeployOrchState, peerID, agentID, host string, err error) error {
	if err == nil {
		return nil
	}
	if cerr := checkpointFailure(prev, peerID, agentID, host, err.Error()); cerr != nil {
		return fmt.Errorf("%w (also journal: %v)", err, cerr)
	}
	return err
}
