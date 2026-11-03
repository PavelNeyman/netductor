package secondary

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForceReinstallStopsOnForgetError(t *testing.T) {
	isolateState(t)
	// Corrupt TOFU so ForgetSSHHost fails on load
	if err := os.MkdirAll(filepath.Dir(relayKnownPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(relayKnownPath(), []byte("{bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ProvisionFromCore(ProvisionIn{
		Host:           "203.0.113.80",
		ForceReinstall: true,
		MTLSCA:         []byte("ca"),
		MTLSCert:       []byte("cert"),
		MTLSKey:        []byte("key"),
	}, `{}`)
	if err == nil {
		t.Fatal("want ForceReinstall error")
	}
	if !strings.Contains(err.Error(), "ForceReinstall") && !strings.Contains(err.Error(), "SSH trust") {
		// must mention trust reset failure
		if !strings.Contains(err.Error(), "TOFU") && !strings.Contains(err.Error(), "corrupt") {
			t.Fatalf("unexpected: %v", err)
		}
	}
}

func TestValidateOperatorPubKeyRejectsMultilineShellInjection(t *testing.T) {
	for _, input := range []string{
		"ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIG5vdC1hLXZhbGlkLWtleQ==\n; touch /root/owned",
		"\n; id",
		"",
	} {
		if _, err := validateOperatorPubKey(input); err == nil {
			t.Fatalf("expected invalid public key to be rejected: %q", input)
		}
	}
}
