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
