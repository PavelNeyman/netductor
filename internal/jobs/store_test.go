package jobs

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestStartSucceed(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	s.Register("noop", func(ctx context.Context, j *Job, appendLog func(string), setStep func(Step)) (any, error) {
		appendLog("hello")
		setStep(Step{ID: "x", Message: "ok", Done: true})
		return map[string]any{"ok": true}, nil
	})
	j, err := s.Start("noop", nil)
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		g := s.Get(j.ID)
		g.mu.Lock()
		st := g.Status
		g.mu.Unlock()
		if st == StatusSucceeded {
			return
		}
		if st == StatusFailed {
			t.Fatal(g.Error)
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timeout", filepath.Base(dir))
}
