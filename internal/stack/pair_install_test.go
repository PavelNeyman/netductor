package stack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallStagedPairToRestoresNodeOnTgFail(t *testing.T) {
	root := t.TempDir()
	nodeDest := filepath.Join(root, "netductor")
	tgDest := filepath.Join(root, "netductor-tg")
	_ = os.WriteFile(nodeDest, []byte("OLD-NODE"), 0o755)
	_ = os.WriteFile(tgDest, []byte("OLD-TG"), 0o755)
	src := t.TempDir()
	nodeSrc := filepath.Join(src, "netductor")
	tgSrc := filepath.Join(src, "netductor-tg")
	_ = os.WriteFile(nodeSrc, []byte("NEW-NODE"), 0o755)
	// tgSrc missing → install fails after node
	att := filepath.Join(root, "attempt")
	err := installStagedPairTo(nodeSrc, tgSrc, nodeDest, tgDest, att)
	if err == nil {
		t.Fatal("expected tg failure")
	}
	got, _ := os.ReadFile(nodeDest)
	if string(got) != "OLD-NODE" {
		t.Fatalf("node should be restored, got %q", got)
	}
}

func TestInstallStagedPairToHappy(t *testing.T) {
	root := t.TempDir()
	nodeDest := filepath.Join(root, "netductor")
	tgDest := filepath.Join(root, "netductor-tg")
	_ = os.WriteFile(nodeDest, []byte("OLD-NODE"), 0o755)
	_ = os.WriteFile(tgDest, []byte("OLD-TG"), 0o755)
	src := t.TempDir()
	nodeSrc := filepath.Join(src, "netductor")
	tgSrc := filepath.Join(src, "netductor-tg")
	_ = os.WriteFile(nodeSrc, []byte("NEW-NODE"), 0o755)
	_ = os.WriteFile(tgSrc, []byte("NEW-TG"), 0o755)
	att := filepath.Join(root, "attempt")
	if err := installStagedPairTo(nodeSrc, tgSrc, nodeDest, tgDest, att); err != nil {
		t.Fatal(err)
	}
	nb, _ := os.ReadFile(nodeDest)
	tb, _ := os.ReadFile(tgDest)
	if string(nb) != "NEW-NODE" || string(tb) != "NEW-TG" {
		t.Fatalf("got node=%q tg=%q", nb, tb)
	}
}
