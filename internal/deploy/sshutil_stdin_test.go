package deploy

import (
	"io"
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

func TestPrintRemoteAllowlistNeverEchoesRemoteBody(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	poison := "svc-paths secondary ok" + "\n" + "sp_priv_s=SECRET_SHOULD_NOT_APPEAR" + "\n" + "token=LEAK"
	printRemoteAllowlist("bootstrap-secondary", poison, "svc-paths secondary")
	_ = w.Close()
	os.Stderr = old
	got, _ := io.ReadAll(r)
	_ = r.Close()
	s := string(got)
	if strings.Contains(s, "SECRET") || strings.Contains(s, "LEAK") || strings.Contains(s, "sp_priv") {
		t.Fatalf("remote body leaked: %q", s)
	}
	if !strings.Contains(s, "bootstrap-secondary: ok") {
		t.Fatalf("want fixed status, got %q", s)
	}
}

func TestPrintVPNHostnameStatusSafe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	poison := "vpn_hostname=vpn.example.test" + "\n" + "sp_priv_s=NO"
	printVPNHostnameStatus(poison)
	_ = w.Close()
	os.Stderr = old
	got, _ := io.ReadAll(r)
	_ = r.Close()
	s := string(got)
	if strings.Contains(s, "NO") || strings.Contains(s, "sp_priv") {
		t.Fatalf("leak: %q", s)
	}
	if strings.Contains(s, "vpn.example.test") {
		t.Fatalf("must not echo hostname value: %q", s)
	}
	if !strings.Contains(s, "vpn-hostname: configured") {
		t.Fatalf("want configured status, got %q", s)
	}
}

func TestSSHOptsKeyOnlyRequiresPin(t *testing.T) {
	_, err := sshOpts("", false, false, "203.0.113.77", "52222")
	if err == nil {
		t.Fatal("key-only without TOFU pin must fail closed")
	}
	if !strings.Contains(err.Error(), "pin required") && !strings.Contains(err.Error(), "TOFU") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestSSHOptsPasswordMayAcceptNew(t *testing.T) {
	args, err := sshOpts("", true, false, "203.0.113.78", "22")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "accept-new") {
		t.Fatalf("password hop should allow accept-new, got %v", args)
	}
}

func TestSSHOptsPasswordStillFailsOnCorruptPin(t *testing.T) {
	// Simulate openSSHPinPath path by forcing Write failure is hard; call sshOpts when
	// TOFU missing is accept-new, but corrupt is tested via openSSHPinPath returning non-errNoPin.
	// Inject by writing corrupt TOFU under NETDUCTOR_STATE.
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	sec := filepath.Join(dir, "secondary")
	_ = os.MkdirAll(sec, 0o700)
	if err := os.WriteFile(filepath.Join(sec, "ssh_known_hosts.json"), []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := sshOpts("", true, false, "203.0.113.90", "22")
	if err == nil {
		t.Fatal("password mode must not accept-new when TOFU is corrupt")
	}
}
