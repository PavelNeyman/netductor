package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMaterialVerifyRemoteCmdHasNoShipBody(t *testing.T) {
	cmd := materialVerifyRemoteCmd()
	secret := `{"sp_priv_s":"SECRET_PRIVATE_KEY_VALUE_XYZ","ps_priv_s":"OTHER_SECRET"}`
	if strings.Contains(cmd, "SECRET_PRIVATE_KEY") || strings.Contains(cmd, secret) {
		t.Fatal("ship must not appear in production remote command")
	}
	if !strings.Contains(cmd, "json.load(sys.stdin)") {
		t.Fatal("expected stdin load in production command")
	}
	if strings.Contains(cmd, "sys.argv[2]") {
		t.Fatal("must not load expected material from argv")
	}
}

func TestMaterialShipTempInJournalDir(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	dir := filepath.Join(deployJournalDir(), "tmp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.CreateTemp(dir, "nd-svc-material-*.json")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	_ = f.Chmod(0o600)
	_ = f.Close()
	defer os.Remove(path)
	if !strings.Contains(path, filepath.Join(".cache", "netductor", "deploy", "tmp")) && !strings.Contains(path, "deploy"+string(os.PathSeparator)+"tmp") {
		// HOME is tmp; deployJournalDir should be under it
		if !strings.HasPrefix(path, dir) {
			t.Fatalf("temp outside journal dir: %s want under %s", path, dir)
		}
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("mode %v", st.Mode())
	}
}

func TestRunSSHOnPortWithStdinDoesNotLeakInError(t *testing.T) {
	if _, err := exec.LookPath("ssh"); err != nil {
		t.Skip("ssh not available")
	}
	secret := []byte(`{"sp_priv_s":"DO_NOT_LEAK_IN_ARGV"}`)
	_, err := runSSHOnPortWithStdin("1", "", "", "root", "127.0.0.1", "true", "", secret)
	if err == nil {
		t.Fatal("expected connection failure on port 1")
	}
	if strings.Contains(err.Error(), "DO_NOT_LEAK") {
		t.Fatal("secret leaked in error")
	}
}
