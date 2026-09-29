package notify

import (
	"testing"
	"time"
)

func TestEnqueueDedupe(t *testing.T) {
	batchMu.Lock()
	pending = map[string]string{}
	batchMu.Unlock()
	EnqueueAlert("a", "msg-a1")
	EnqueueAlert("a", "msg-a2")
	EnqueueAlert("b", "msg-b")
	batchMu.Lock()
	defer batchMu.Unlock()
	if len(pending) != 2 {
		t.Fatalf("want 2 keys got %d", len(pending))
	}
	if pending["a"] != "msg-a2" {
		t.Fatalf("dedupe overwrite: %q", pending["a"])
	}
	_ = time.Second
}
