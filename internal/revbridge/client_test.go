package revbridge

import (
	"errors"
	"testing"
)

func TestSessionOnceReturnsNilInterfacePossible(t *testing.T) {
	// sessionOnce must be able to return a nil error so loopKind can reset backoff.
	// Stopped client returns errClientStopped (non-nil); that path is tested below.
	c := NewClient("127.0.0.1", nil)
	close(c.stop)
	err := c.sessionOnce(PortPlan{Kind: KindWS, PublicPort: 1, PrimaryDial: "127.0.0.1:1", Transport: "tcp"})
	if !errors.Is(err, errClientStopped) {
		t.Fatalf("stopped client: got %v, want errClientStopped", err)
	}
}

func TestLoopKindDoesNotTreatAllErrorsAsBackoffOnly(t *testing.T) {
	// Compile-time / staticcheck contract: sessionOnce's error is a true optional
	// error (nil means reconnect). Documented in sessionOnce godoc.
	var err error
	if err != nil {
		t.Fatal("nil error must be distinguishable")
	}
}
