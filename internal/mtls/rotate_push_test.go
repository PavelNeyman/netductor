package mtls

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupRotationTest(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	t.Setenv("NETDUCTOR_STATE", filepath.Join(root, "state"))
	if err := EnsureAll("127.0.0.1"); err != nil {
		t.Fatalf("EnsureAll: %v", err)
	}
	if _, _, _, err := EnsureClientFor("secondary-test"); err != nil {
		t.Fatalf("EnsureClientFor: %v", err)
	}
	return root
}

func TestRotateClientForPushIsIdempotentUntilConfirmation(t *testing.T) {
	setupRotationTest(t)
	oldInfo, err := ReadClientInfo("secondary-test")
	if err != nil {
		t.Fatal(err)
	}

	newInfo, _, _, _, err := RotateClientForPush("secondary-test", "test")
	if err != nil {
		t.Fatalf("RotateClientForPush: %v", err)
	}
	if newInfo.Serial == oldInfo.Serial {
		t.Fatal("rotation did not issue a new certificate")
	}
	pending, err := PendingRotates()
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending rotations = %d, err = %v", len(pending), err)
	}
	if pending[0].OldSerial != oldInfo.Serial || pending[0].NewSerial != newInfo.Serial {
		t.Fatalf("unexpected pending rotation: %+v", pending[0])
	}

	retryInfo, _, _, _, err := RotateClientForPush("secondary-test", "retry")
	if err != nil {
		t.Fatalf("idempotent retry: %v", err)
	}
	if retryInfo.Serial != newInfo.Serial {
		t.Fatalf("retry changed certificate serial: got %s want %s", retryInfo.Serial, newInfo.Serial)
	}

	if err := ConfirmRotate("secondary-test", oldInfo.Serial); err != nil {
		t.Fatalf("confirmation with old serial should be ignored: %v", err)
	}
	if err := ConfirmRotate("secondary-test", newInfo.Serial); err != nil {
		t.Fatalf("ConfirmRotate: %v", err)
	}
	oldSerial, ok := new(big.Int).SetString(oldInfo.Serial, 16)
	if !ok {
		t.Fatalf("invalid old serial %q", oldInfo.Serial)
	}
	revoked, err := IsRevoked(oldSerial)
	if err != nil || !revoked {
		t.Fatalf("old certificate revoked = %v, err = %v", revoked, err)
	}
	pending, err = PendingRotates()
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending rotations after confirmation = %d, err = %v", len(pending), err)
	}
}

func TestConfirmRotateRetainsPendingWhenRevocationFails(t *testing.T) {
	setupRotationTest(t)
	if _, _, _, _, err := RotateClientForPush("secondary-test", "test"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(revokePath(), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ConfirmRotate("secondary-test", ""); err == nil {
		t.Fatal("expected revocation error")
	}
	var list []pendingRotate
	if err := json.Unmarshal(mustReadFile(t, pendingPath()), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("pending rotation was discarded after revoke failure: %+v", list)
	}
}

func TestExpiredRotationReturnsRevocationFailure(t *testing.T) {
	setupRotationTest(t)
	old := pendingRotate{
		NodeID: "secondary-test", OldSerial: "abc123", NewSerial: "def456",
		CreatedAt: time.Now().Add(-48 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(-time.Hour).Unix(),
	}
	if err := savePending(map[string]pendingRotate{old.NodeID: old}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(revokePath(), []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := PendingRotates(); err == nil {
		t.Fatal("expected expired certificate revocation error")
	}
	var list []pendingRotate
	if err := json.Unmarshal(mustReadFile(t, pendingPath()), &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expired pending record was silently discarded: %+v", list)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
