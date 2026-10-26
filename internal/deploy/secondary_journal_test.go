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
	writeDeployJournal("peer-x", "agent-1", host, StageProvisioned)
	st, err := loadDeployState(host)
	if err != nil {
		t.Fatal(err)
	}
	if st == nil || st.Stage != StageProvisioned || st.PeerID != "peer-x" || st.AgentID != "agent-1" {
		t.Fatalf("%+v", st)
	}
	if !canSkipProvision(st) {
		t.Fatal("should skip provision")
	}
	if canSkipPeerAdd(st) {
		t.Fatal("should not skip peer add yet")
	}
	writeDeployJournal("peer-x", "agent-1", host, StagePeerPrepared)
	st, _ = loadDeployState(host)
	if !canSkipPeerAdd(st) {
		t.Fatal("should skip peer add")
	}
	writeDeployJournal("peer-x", "agent-1", host, StageDeployComplete)
	clearDeployState(host)
	st, err = loadDeployState(host)
	if err != nil {
		t.Fatal(err)
	}
	if st != nil {
		t.Fatalf("expected cleared, got %+v", st)
	}
	// append log exists
	if _, err := os.Stat(filepath.Join(tmp, ".cache", "netductor", "deploy", "journal.log")); err != nil {
		t.Fatal(err)
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
	stages := []string{
		StageProvisionStarted, StageProvisioned, StagePeerAddStarted, StagePeerPrepared,
		StageMaterialTransferred, StageSecondaryBootstrapped, StageFirewallVerified,
		StageNetworkVerified, StageHeartbeatVerified, StageDeployComplete,
	}
	for i, s := range stages {
		writeDeployJournal("peer-c", "ag-1", host, s)
		st, err := loadDeployState(host)
		if err != nil || st == nil || st.Stage != s {
			t.Fatalf("%d %s: %+v %v", i, s, st, err)
		}
		if i > 0 && stageRank(s) <= stageRank(stages[i-1]) && s != StageDeployComplete {
			// ranks non-decreasing through verified path
			if stageRank(s) < stageRank(stages[i-1]) {
				t.Fatalf("rank dropped %s -> %s", stages[i-1], s)
			}
		}
	}
	if !canSkipBootstrapB(mustState(t, host)) {
		t.Fatal("should skip bootstrap after complete path stages")
	}
}

func mustState(t *testing.T, host string) *deployOrchState {
	t.Helper()
	st, err := loadDeployState(host)
	if err != nil || st == nil {
		t.Fatal(err, st)
	}
	return st
}
