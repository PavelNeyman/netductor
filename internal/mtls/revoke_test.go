package mtls

import (
	"math/big"
	"os"
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
	ok, err := IsRevoked(n)
	if err != nil || !ok {
		t.Fatal("expected revoked", err)
	}
	list, err := ListRevoked()
	if err != nil || len(list) != 1 {
		t.Fatal(list, err)
	}
}

func TestIsRevokedFailClosedCorruptFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", dir)
	if err := RevokeSerial("aa", "n", "x"); err != nil {
		t.Fatal(err)
	}
	rp := revokePath()
	if err := os.WriteFile(rp, []byte("{not-json"), 0o600); err != nil {
		t.Fatal(err)
	}
	n := new(big.Int)
	n.SetString("aa", 16)
	_, err := IsRevoked(n)
	if err == nil {
		t.Fatal("expected error on corrupt revoke list")
	}
}
