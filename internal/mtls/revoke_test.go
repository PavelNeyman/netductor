package mtls

import (
	"math/big"
	"path/filepath"
	"testing"
)

func TestRevokeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", dir)
	// Dir() uses paths.EtcDir secrets/mtls — need paths env
	t.Setenv("NETDUCTOR_ETC", dir)
	_ = filepath.Join(dir, "secrets", "mtls")
	if err := RevokeSerial("abc123", "node1", "test"); err != nil {
		t.Fatal(err)
	}
	n := new(big.Int)
	n.SetString("abc123", 16)
	if !IsRevoked(n) {
		t.Fatal("expected revoked")
	}
	list, err := ListRevoked()
	if err != nil || len(list) != 1 {
		t.Fatal(list, err)
	}
}
