package install

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRecoveryHTTPClientBadPEM(t *testing.T) {
	dir := t.TempDir()
	pin := filepath.Join(dir, "pin.crt")
	if err := os.WriteFile(pin, []byte("not a pem"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("NETDUCTOR_RECOVERY_CA", pin)
	c, err := recoveryHTTPClient()
	if err == nil || c != nil {
		t.Fatalf("want error on bad PEM, got client=%v err=%v", c, err)
	}
}
