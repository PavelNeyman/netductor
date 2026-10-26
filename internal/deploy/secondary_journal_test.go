package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeployJournalStateRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.10"
	var prev *DeployOrchState
	if err := checkpointSuccess(&prev, "peer-x", "agent-1", host, StageProvisioned); err != nil {
		t.Fatal(err)
	}
	st, err := loadDeployState(host)
	if err != nil {
		t.Fatal(err)
	}
	if st == nil || st.LastSuccessfulStage != StageProvisioned || st.PeerID != "peer-x" || st.AgentID != "agent-1" {
		t.Fatalf("%+v", st)
	}
	if st.Status != StatusRunning {
		t.Fatalf("status %s", st.Status)
	}
	if !canSkipProvision(st) {
		t.Fatal("should skip provision")
	}
	if canSkipPeerAdd(st) {
		t.Fatal("should not skip peer add yet")
	}
	if err := checkpointSuccess(&prev, "peer-x", "agent-1", host, StagePeerPrepared); err != nil {
		t.Fatal(err)
	}
	st, _ = loadDeployState(host)
	if !canSkipPeerAdd(st) {
		t.Fatal("should skip peer add")
	}
	if err := checkpointSuccess(&prev, "peer-x", "agent-1", host, StageDeployComplete); err != nil {
		t.Fatal(err)
	}
	if err := clearDeployState(host); err != nil {
		t.Fatal(err)
	}
	st, err = loadDeployState(host)
	if err != nil {
		t.Fatal(err)
	}
	if st != nil {
		t.Fatalf("expected cleared, got %+v", st)
	}
	if _, err := os.Stat(filepath.Join(tmp, ".cache", "netductor", "deploy", "journal.log")); err != nil {
		t.Fatal(err)
	}
}

func TestFailedRecoverableKeepsLastSuccess(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.20"
	var prev *DeployOrchState
	if err := checkpointSuccess(&prev, "peer-y", "ag-2", host, StageNetworkVerified); err != nil {
		t.Fatal(err)
	}
	if err := checkpointFailure(&prev, "peer-y", "ag-2", host, "ufw boom"); err != nil {
		t.Fatal(err)
	}
	st, err := loadDeployState(host)
	if err != nil {
		t.Fatal(err)
	}
	if st.LastSuccessfulStage != StageNetworkVerified {
		t.Fatalf("last success wiped: %+v", st)
	}
	if st.Status != StatusFailedRecoverable {
		t.Fatalf("status=%s", st.Status)
	}
	if st.LastError != "ufw boom" {
		t.Fatalf("err=%s", st.LastError)
	}
	// Resume must still skip provision after failure checkpoint
	if !canSkipProvision(st) {
		t.Fatal("after failed_recoverable should still skip provision based on last success")
	}
	if !canSkipBootstrapB(st) {
		t.Fatal("network_verified implies bootstrap done")
	}
}

func TestCorruptStateFailsClosed(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.30"
	dir := filepath.Join(tmp, ".cache", "netductor", "deploy")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "state-198.51.100.30.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := loadDeployState(host)
	if err == nil {
		t.Fatal("expected corrupt error")
	}
}

func TestPeerIDPreservedAcrossStages(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.40"
	var prev *DeployOrchState
	if err := checkpointSuccess(&prev, "custom-peer", "", host, StageProvisionStarted); err != nil {
		t.Fatal(err)
	}
	// AgentID added later without peerID arg empty should keep peer
	if err := checkpointSuccess(&prev, "", "agent-z", host, StageProvisioned); err != nil {
		t.Fatal(err)
	}
	st, _ := loadDeployState(host)
	if st.PeerID != "custom-peer" || st.AgentID != "agent-z" {
		t.Fatalf("%+v", st)
	}
	if err := checkpointFailure(&prev, "", "", host, "sync fail"); err != nil {
		t.Fatal(err)
	}
	st, _ = loadDeployState(host)
	if st.PeerID != "custom-peer" || st.AgentID != "agent-z" {
		t.Fatalf("ids lost on failure: %+v", st)
	}
}

func TestStageRankOrder(t *testing.T) {
	if stageRank(StageProvisioned) >= stageRank(StagePeerPrepared) {
		t.Fatal("rank order")
	}
	if stageRank(StageDeployComplete) < stageRank(StageHeartbeatVerified) {
		t.Fatal("complete after heartbeat")
	}
}

func TestDeployStageChain(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "203.0.113.50"
	var prev *DeployOrchState
	stages := []string{
		StageProvisionStarted, StageProvisioned, StagePeerAddStarted, StagePeerPrepared,
		StageMaterialTransferred, StageSecondaryBootstrapped, StageFirewallVerified,
		StageNetworkVerified, StageHeartbeatVerified, StageDeployComplete,
	}
	for i, s := range stages {
		if err := checkpointSuccess(&prev, "peer-c", "ag-1", host, s); err != nil {
			t.Fatal(err)
		}
		st, err := loadDeployState(host)
		if err != nil || st == nil || st.LastSuccessfulStage != s {
			t.Fatalf("%d %s: %+v %v", i, s, st, err)
		}
	}
	if !canSkipBootstrapB(prev) {
		t.Fatal("should skip bootstrap after complete path stages")
	}
}

func TestLegacyStageMigration(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "203.0.113.60"
	dir := filepath.Join(tmp, ".cache", "netductor", "deploy")
	_ = os.MkdirAll(dir, 0o700)
	// Old format: only "stage"
	raw := `{"peer_id":"p1","agent_id":"a1","host":"203.0.113.60","stage":"provisioned"}`
	if err := os.WriteFile(filepath.Join(dir, "state-203.0.113.60.json"), []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	st, err := loadDeployState(host)
	if err != nil {
		t.Fatal(err)
	}
	if st.LastSuccessfulStage != StageProvisioned {
		t.Fatalf("%+v", st)
	}
	if !canSkipProvision(st) {
		t.Fatal("legacy should skip provision")
	}
}

func TestMaterialSHAPreservedOnFailure(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.70"
	var prev *DeployOrchState
	sha := "abcdef0123456789"
	if err := checkpointSuccessWithSHA(&prev, "peer-m", "ag-m", host, StageMaterialTransferred, sha); err != nil {
		t.Fatal(err)
	}
	if err := checkpointFailure(&prev, "peer-m", "ag-m", host, "bootstrap fail"); err != nil {
		t.Fatal(err)
	}
	st, err := loadDeployState(host)
	if err != nil {
		t.Fatal(err)
	}
	if st.MaterialSHA != sha {
		t.Fatalf("material sha lost: %+v", st)
	}
	if st.LastSuccessfulStage != StageMaterialTransferred {
		t.Fatalf("stage wiped: %+v", st)
	}
	if !canSkipMaterialXfer(st) {
		t.Fatal("should skip material xfer")
	}
}

func TestCanSkipPeerAddUsesLastSuccess(t *testing.T) {
	st := &DeployOrchState{
		LastSuccessfulStage: StagePeerPrepared,
		Status:              StatusFailedRecoverable,
		MaterialSHA:         "deadbeef",
	}
	if !canSkipPeerAdd(st) {
		t.Fatal("peer prepared should skip peer add generation path")
	}
	if canSkipMaterialXfer(st) {
		// Stage is peer_prepared, not material_transferred
		t.Fatal("should not skip material yet")
	}
	st.LastSuccessfulStage = StageMaterialTransferred
	if !canSkipMaterialXfer(st) {
		t.Fatal("should skip material")
	}
}

func TestLastSuccessfulStageMonotonic(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.80"
	var prev *DeployOrchState
	if err := checkpointSuccessWithSHA(&prev, "peer-m", "ag", host, StageMaterialTransferred, "abc123"); err != nil {
		t.Fatal(err)
	}
	// peer_add_started via markCurrentStep must not regress
	if err := markCurrentStep(&prev, "peer-m", "ag", host, StagePeerAddStarted); err != nil {
		t.Fatal(err)
	}
	if prev.LastSuccessfulStage != StageMaterialTransferred {
		t.Fatalf("regressed to %s", prev.LastSuccessfulStage)
	}
	if prev.CurrentStep != StagePeerAddStarted {
		t.Fatalf("current=%s", prev.CurrentStep)
	}
	if !canSkipMaterialXfer(prev) {
		t.Fatal("should still skip material xfer")
	}
	// Accidental checkpointSuccess with earlier stage must not regress either
	if err := checkpointSuccess(&prev, "peer-m", "ag", host, StagePeerPrepared); err != nil {
		t.Fatal(err)
	}
	if prev.LastSuccessfulStage != StageMaterialTransferred {
		t.Fatalf("checkpointSuccess regressed to %s", prev.LastSuccessfulStage)
	}
	if prev.MaterialSHA != "abc123" {
		t.Fatalf("sha lost %s", prev.MaterialSHA)
	}
}

func TestAdvancedJournalBlocksBlindProvisionDecision(t *testing.T) {
	st := &DeployOrchState{
		LastSuccessfulStage: StageSecondaryBootstrapped,
		AgentID:             "ag-1",
		Status:              StatusRunning,
	}
	if !canSkipProvision(st) {
		t.Fatal("advanced should skip provision")
	}
	if !canSkipBootstrapB(st) {
		t.Fatal("should skip bootstrap")
	}
}

func TestBootstrapSkipRequiresMatchingMaterialSHA(t *testing.T) {
	st := &DeployOrchState{
		LastSuccessfulStage:     StageSecondaryBootstrapped,
		MaterialSHA:             "sha-old",
		BootstrappedMaterialSHA: "sha-old",
	}
	if !canSkipBootstrapForMaterial(st, "sha-old") {
		t.Fatal("same SHA should skip")
	}
	if canSkipBootstrapForMaterial(st, "sha-new") {
		t.Fatal("changed material must force bootstrap")
	}
	st.BootstrappedMaterialSHA = ""
	if canSkipBootstrapForMaterial(st, "sha-old") {
		t.Fatal("empty bootstrap binding must force bootstrap")
	}
}

func TestMaterialTransferInvalidatesBootstrapBinding(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.90"
	var prev *DeployOrchState
	if err := checkpointBootstrapOK(&prev, "peer-x", "ag", host, "sha-a"); err != nil {
		t.Fatal(err)
	}
	if prev.BootstrappedMaterialSHA != "sha-a" {
		t.Fatal(prev.BootstrappedMaterialSHA)
	}
	// New ship SHA after mismatch re-SCP
	if err := checkpointSuccessWithSHA(&prev, "peer-x", "ag", host, StageMaterialTransferred, "sha-b"); err != nil {
		t.Fatal(err)
	}
	if prev.BootstrappedMaterialSHA != "" {
		t.Fatalf("bootstrap binding should clear on new material SHA, got %s", prev.BootstrappedMaterialSHA)
	}
	if canSkipBootstrapForMaterial(prev, "sha-b") {
		t.Fatal("must not skip bootstrap after material change")
	}
	if err := checkpointBootstrapOK(&prev, "peer-x", "ag", host, "sha-b"); err != nil {
		t.Fatal(err)
	}
	if !canSkipBootstrapForMaterial(prev, "sha-b") {
		t.Fatal("after re-bootstrap should skip")
	}
}

func TestCheckpointSuccessClearsCurrentStep(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.91"
	var prev *DeployOrchState
	if err := markCurrentStep(&prev, "p", "a", host, StagePeerAddStarted); err != nil {
		t.Fatal(err)
	}
	if prev.CurrentStep == "" {
		t.Fatal("expected current step")
	}
	if err := checkpointSuccess(&prev, "p", "a", host, StagePeerPrepared); err != nil {
		t.Fatal(err)
	}
	if prev.CurrentStep != "" {
		t.Fatalf("current step not cleared: %q", prev.CurrentStep)
	}
}

func TestDeployCompleteKeepsState(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.95"
	var prev *DeployOrchState
	if err := checkpointBootstrapOK(&prev, "peer-c", "ag-c", host, "sha-c"); err != nil {
		t.Fatal(err)
	}
	if err := checkpointDeployComplete(&prev, "peer-c", "ag-c", host); err != nil {
		t.Fatal(err)
	}
	if prev.Status != StatusComplete {
		t.Fatalf("status=%s", prev.Status)
	}
	loaded, err := loadDeployState(host)
	if err != nil || loaded == nil {
		t.Fatalf("state should remain after complete: %v %v", loaded, err)
	}
	if loaded.Status != StatusComplete || effectiveStage(loaded) != StageDeployComplete {
		t.Fatalf("%+v", loaded)
	}
	if !canSkipProvision(loaded) {
		t.Fatal("complete must still protect against blind provision")
	}
}

func TestRedactDeploySecrets(t *testing.T) {
	in := `prepare-pack failed: {"agent_token":"SECRETTOKENVALUE1234567890abcdef","sp_priv_s":"AAAA_PRIVATE_KEY_MATERIAL_XXXX"} and more`
	out := redactDeploySecrets(in)
	if strings.Contains(out, "SECRETTOKEN") || strings.Contains(out, "PRIVATE_KEY_MATERIAL") {
		t.Fatalf("secret leaked: %s", out)
	}
	if !strings.Contains(out, "redacted") && !strings.Contains(out, "[redacted]") {
		t.Fatalf("expected redaction markers: %s", out)
	}
}

func TestFailDeployRedactsJournal(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.77"
	var prev *DeployOrchState
	_ = checkpointSuccess(&prev, "p", "a", host, StageProvisioned)
	err := failDeploy(&prev, "p", "a", host, fmt.Errorf("boom token=SUPERSECRETVALUE123456789012345 and sp_priv_s=XYZ"))
	if err == nil {
		t.Fatal("expected err")
	}
	if strings.Contains(err.Error(), "SUPERSECRET") {
		t.Fatalf("err leaked: %v", err)
	}
	st, _ := loadDeployState(host)
	if st != nil && strings.Contains(st.LastError, "SUPERSECRET") {
		t.Fatalf("journal leaked: %s", st.LastError)
	}
}

func TestSafeStageErrNeverEmbedsPayload(t *testing.T) {
	err := safeStageErr("prepare-pack", "ssh", fmt.Errorf(`{"agent_token":"SECRET1234567890","sp_priv_s":"PRIVKEY"} timeout`))
	s := err.Error()
	if strings.Contains(s, "SECRET") || strings.Contains(s, "PRIVKEY") || strings.Contains(s, "agent_token") {
		t.Fatalf("payload in safe error: %s", s)
	}
	if !strings.Contains(s, "prepare-pack") || !strings.Contains(s, "timeout") {
		t.Fatalf("want stage+class: %s", s)
	}
}

func TestMaterialPreparedSameRankAsLegacyTransferred(t *testing.T) {
	if stageRank(StageMaterialPrepared) != stageRank(StageMaterialTransferred) {
		t.Fatalf("ranks differ %d vs %d", stageRank(StageMaterialPrepared), stageRank(StageMaterialTransferred))
	}
	st := &DeployOrchState{LastSuccessfulStage: StageMaterialTransferred}
	if !canSkipMaterialXfer(st) {
		t.Fatal("legacy material_transferred must still skip material stage")
	}
}
