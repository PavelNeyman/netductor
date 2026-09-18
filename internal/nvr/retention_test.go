package nvr

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunRetentionAgeAndMaxGB(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "cam1")
	_ = os.MkdirAll(old, 0o700)
	p1 := filepath.Join(old, "old.mp4")
	p2 := filepath.Join(old, "new.mp4")
	_ = os.WriteFile(p1, make([]byte, 1024), 0o600)
	_ = os.WriteFile(p2, make([]byte, 1024), 0o600)
	past := time.Now().Add(-48 * time.Hour)
	_ = os.Chtimes(p1, past, past)

	cfg := Config{Path: dir, RetentionDays: 1, MaxGB: 0, MinFreeGB: 0}
	rep, err := RunRetention(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Deleted < 1 {
		t.Fatalf("expected age delete, %+v", rep)
	}
	if _, err := os.Stat(p1); !os.IsNotExist(err) {
		t.Fatal("old should be gone")
	}
	if _, err := os.Stat(p2); err != nil {
		t.Fatal("new should remain")
	}
}
