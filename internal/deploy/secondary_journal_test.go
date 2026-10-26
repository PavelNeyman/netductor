package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeployJournalStateRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	host := "198.51.100.10"
	var prev *deployOrchState
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
	var prev *deployOrchState
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
	var prev *deployOrchState
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
	var prev *deployOrchState
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
	var prev *deployOrchState
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
	st := &deployOrchState{
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
