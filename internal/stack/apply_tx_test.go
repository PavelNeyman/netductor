package stack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecoverRestoresNodeAfterKill(t *testing.T) {
	dir := t.TempDir()
	node := filepath.Join(dir, "netductor")
	tg := filepath.Join(dir, "netductor-tg")
	if err := os.WriteFile(node, []byte("old-node"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tg, []byte("old-tg"), 0o755); err != nil {
		t.Fatal(err)
	}
	att := filepath.Join(dir, "attempt")
	newNode := filepath.Join(dir, "new-node")
	newTG := filepath.Join(dir, "new-tg")
	if err := os.WriteFile(newNode, []byte("new-node"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newTG, []byte("new-tg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := installStagedPairTo(newNode, newTG, node, tg, att); err != nil {
		t.Fatal(err)
	}
	// Simulate kill after node_installed: put new node live, old tg, journal stuck.
	if err := os.WriteFile(node, []byte("new-node"), 0o755); err != nil {
		t.Fatal(err)
	}
	tx, err := loadTx(att)
	if err != nil {
		t.Fatal(err)
	}
	tx.Phase = txNodeInstalled
	if err := writeTx(att, *tx); err != nil {
		t.Fatal(err)
	}
	if err := recoverApplyTx(att); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(node)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old-node" {
		t.Fatalf("node not restored: %s", got)
	}
	if _, err := os.Stat(filepath.Join(att, "netductor")); err != nil {
		t.Fatal("pre-image removed")
	}
}

func TestRecoverCorruptJournalBlocks(t *testing.T) {
	dir := t.TempDir()
	att := filepath.Join(dir, "attempt")
	if err := os.MkdirAll(att, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(att, "tx.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := recoverApplyTx(att); err == nil {
		t.Fatal("corrupt journal must block")
	}
}
