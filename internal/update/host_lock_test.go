package update

import (
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestHostComponentLockBlocksConcurrent(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	unlock, err := tryHostLockOnce(ComponentTG, "local")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	var got atomic.Value
	done := make(chan struct{})
	go func() {
		err := withHostComponentLock(ComponentTG, "local", func() error {
			return nil
		})
		got.Store(err)
		close(done)
	}()
	select {
	case <-done:
		e, _ := got.Load().(error)
		if e == nil {
			t.Fatal("expected concurrent lock failure")
		}
		if e != nil && !contains(e.Error(), "already in progress") {
			t.Fatalf("want already in progress, got %v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("lock attempt hung")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}

func TestHostComponentLockReleases(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	err := withHostComponentLock(ComponentAgent, "local", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	err = withHostComponentLock(ComponentAgent, "local", func() error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	_ = os.Remove(hostLockPath(ComponentAgent, "local"))
}
