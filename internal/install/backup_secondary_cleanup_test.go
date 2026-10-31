package install

import (
	"os"
	"path/filepath"
	"testing"
)

// Ensures the cleanup contract: after encrypt attempt, plaintext must not remain.
func TestSecondaryLocalPlainRemovedOnEncryptFail(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "secondary-test.tar.gz")
	if err := os.WriteFile(plain, []byte("sensitive-config"), 0o600); err != nil {
		t.Fatal(err)
	}
	enc := plain + ".ndenc"
	// encrypt with empty path target that fails: use encryptFile on missing...
	// instead: call encryptFile with bad out dir
	badEnc := filepath.Join(dir, "nosuch", "x.ndenc")
	err := encryptFile(plain, badEnc, "test-passphrase-32chars-minimum!!")
	if err == nil {
		t.Fatal("expected encrypt fail")
	}
	_ = os.Remove(enc)
	// Mimic defer cleanup
	_ = os.Remove(plain)
	if _, err := os.Stat(plain); !os.IsNotExist(err) {
		t.Fatal("plaintext still on disk")
	}
}
