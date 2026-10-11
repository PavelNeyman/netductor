package integrity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func genTestKey(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	priv := filepath.Join(dir, "id")
	if out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", priv, "-C", "test@netductor").CombinedOutput(); err != nil {
		t.Skipf("ssh-keygen: %v %s", err, out)
	}
	pub, err := os.ReadFile(priv + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(pub))
}

func TestParseAuthorizedKeysSkipsComments(t *testing.T) {
	line := genTestKey(t)
	dir := t.TempDir()
	p := filepath.Join(dir, "authorized_keys")
	body := "# comment\n\n" + line + "\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	keys := ParseAuthorizedKeys(p)
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Type == "" || keys[0].Fingerprint == "" {
		t.Fatalf("%+v", keys[0])
	}
	if !strings.Contains(keys[0].Comment, "test@netductor") && keys[0].Comment == "" {
		// comment optional depending on ssh-keygen -lf
		t.Logf("comment=%q", keys[0].Comment)
	}
}

func TestParseKeysFromText(t *testing.T) {
	line := genTestKey(t)
	ks := ParseKeysFromText("secondary-1", line+"\n")
	if len(ks) != 1 {
		t.Fatalf("expected 1, got %d", len(ks))
	}
	if ks[0].Host != "secondary-1" {
		t.Fatalf("host %q", ks[0].Host)
	}
	if !strings.Contains(ks[0].Line, "ssh-ed25519") {
		t.Fatalf("line %q", ks[0].Line)
	}
	if ks[0].Fingerprint == "" {
		t.Fatal("empty fp")
	}
}

func TestDefaultPathsNonEmpty(t *testing.T) {
	if len(DefaultPaths()) < 3 {
		t.Fatal("DefaultPaths too short")
	}
}
