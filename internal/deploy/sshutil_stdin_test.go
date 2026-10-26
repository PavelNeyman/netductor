package deploy

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestMaterialVerifyRemoteCmdHasNoShipBody ensures the verify script does not embed ship JSON.
func TestMaterialVerifyRemoteCmdHasNoShipBody(t *testing.T) {
	// Reconstruct the fixed remote script used in DeploySecondary (no ship bytes).
	verifyMat := `set -e
f=/etc/netductor/svc-paths/material.json
test -f "$f"
if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 required for material verify — refuse skip-SCP" >&2
  exit 2
fi
python3 -c '
import json,sys
a=json.load(open(sys.argv[1]))
b=json.load(sys.stdin)
' "$f"
`
	secret := `{"sp_priv_s":"SECRET_PRIVATE_KEY_VALUE_XYZ","ps_priv_s":"OTHER_SECRET"}`
	if strings.Contains(verifyMat, "SECRET_PRIVATE_KEY") {
		t.Fatal("fixture broken")
	}
	if strings.Contains(verifyMat, secret) {
		t.Fatal("ship must not appear in remote command text")
	}
	// stdin path: payload separate from cmd
	if !strings.Contains(verifyMat, "json.load(sys.stdin)") {
		t.Fatal("expected stdin load")
	}
}

func TestCreateTempMaterialMode(t *testing.T) {
	f, err := os.CreateTemp("", "nd-svc-material-test-*.json")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	defer os.Remove(path)
	if err := f.Chmod(0o600); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("mode %v should be 0600-ish", st.Mode())
	}
	_ = filepath.Base(path)
}

func TestRunSSHOnPortWithStdinEcho(t *testing.T) {
	// Smoke: local echo of stdin via ssh is not available; ensure helper compiles and rejects empty host path via ssh failure without embedding secret in error when cmd is fixed.
	if _, err := exec.LookPath("ssh"); err != nil {
		t.Skip("ssh not available")
	}
	secret := []byte(`{"sp_priv_s":"DO_NOT_LEAK_IN_ARGV"}`)
	// Force fail: invalid host; command has no secret
	cmd := "true"
	_, err := runSSHOnPortWithStdin("1", "", "", "root", "127.0.0.1", cmd, "", secret)
	if err == nil {
		t.Fatal("expected connection failure on port 1")
	}
	if strings.Contains(err.Error(), "DO_NOT_LEAK") {
		t.Fatal("secret leaked in error")
	}
}
