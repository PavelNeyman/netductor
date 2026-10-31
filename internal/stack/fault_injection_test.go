package stack

import (
	"os"
	"path/filepath"
	"testing"
)

// F: fault-injection for staged node+tg pair install (no live systemctl).

func TestFaultNodeSrcMissingLeavesDestUnchanged(t *testing.T) {
	root := t.TempDir()
	nodeDest := filepath.Join(root, "netductor")
	tgDest := filepath.Join(root, "netductor-tg")
	_ = os.WriteFile(nodeDest, []byte("OLD-NODE"), 0o755)
	_ = os.WriteFile(tgDest, []byte("OLD-TG"), 0o755)
	src := t.TempDir()
	nodeSrc := filepath.Join(src, "missing-node")
	tgSrc := filepath.Join(src, "netductor-tg")
	_ = os.WriteFile(tgSrc, []byte("NEW-TG"), 0o755)
	att := filepath.Join(root, "attempt")
	err := installStagedPairTo(nodeSrc, tgSrc, nodeDest, tgDest, att)
	if err == nil {
		t.Fatal("expected node install failure")
	}
	got, _ := os.ReadFile(nodeDest)
	if string(got) != "OLD-NODE" {
		t.Fatalf("node changed on failed install: %q", got)
	}
	got, _ = os.ReadFile(tgDest)
	if string(got) != "OLD-TG" {
		t.Fatalf("tg should be untouched when node fails first: %q", got)
	}
}

func TestFaultTgFailRestoresNodeAndKeepsOldTg(t *testing.T) {
	root := t.TempDir()
	nodeDest := filepath.Join(root, "netductor")
	tgDest := filepath.Join(root, "netductor-tg")
	_ = os.WriteFile(nodeDest, []byte("OLD-NODE"), 0o755)
	_ = os.WriteFile(tgDest, []byte("OLD-TG"), 0o755)
	src := t.TempDir()
	nodeSrc := filepath.Join(src, "netductor")
	_ = os.WriteFile(nodeSrc, []byte("NEW-NODE"), 0o755)
	tgSrc := filepath.Join(src, "no-tg")
	att := filepath.Join(root, "attempt")
	err := installStagedPairTo(nodeSrc, tgSrc, nodeDest, tgDest, att)
	if err == nil {
		t.Fatal("expected tg failure")
	}
	if string(mustRead(nodeDest)) != "OLD-NODE" {
		t.Fatalf("node not restored: %q", mustRead(nodeDest))
	}
	if string(mustRead(tgDest)) != "OLD-TG" {
		t.Fatalf("tg should remain old when tg install fails: %q", mustRead(tgDest))
	}
	// attempt snapshot of old node must exist for forensics
	if _, err := os.Stat(filepath.Join(att, "netductor")); err != nil {
		t.Fatal("attempt backup missing", err)
	}
}

func TestFaultAttemptDirNotWritable(t *testing.T) {
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
	// attempt path is a file → MkdirAll fails or backup fails
	att := filepath.Join(root, "attempt")
	_ = os.WriteFile(att, []byte("not-a-dir"), 0o644)
	err := installStagedPairTo(nodeSrc, tgSrc, nodeDest, tgDest, att)
	if err == nil {
		t.Fatal("expected attempt dir failure")
	}
	if string(mustRead(nodeDest)) != "OLD-NODE" || string(mustRead(tgDest)) != "OLD-TG" {
		t.Fatal("binaries must not change when attempt backup cannot be prepared")
	}
}

func TestFaultCorruptNodeDestStillBackedUp(t *testing.T) {
	// Dest is readable file; after successful pair both new.
	root := t.TempDir()
	nodeDest := filepath.Join(root, "netductor")
	tgDest := filepath.Join(root, "netductor-tg")
	_ = os.WriteFile(nodeDest, []byte("OLD-NODE"), 0o755)
	_ = os.WriteFile(tgDest, []byte("OLD-TG"), 0o755)
	src := t.TempDir()
	_ = os.WriteFile(filepath.Join(src, "netductor"), []byte("NEW-NODE"), 0o755)
	_ = os.WriteFile(filepath.Join(src, "netductor-tg"), []byte("NEW-TG"), 0o755)
	att := filepath.Join(root, "attempt")
	if err := installStagedPairTo(filepath.Join(src, "netductor"), filepath.Join(src, "netductor-tg"), nodeDest, tgDest, att); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(att, "netductor"))
	if string(b) != "OLD-NODE" {
		t.Fatalf("attempt should keep pre-image: %q", b)
	}
}

func mustRead(path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}
