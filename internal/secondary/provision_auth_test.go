package secondary

import (
	"strings"
	"testing"
)

func TestSSHClientBootstrapPasswordOnlyWhenPasswordSet(t *testing.T) {
	// Structural: when Password set, bootstrap path must not require private key file.
	in := ProvisionIn{Host: "203.0.113.50", Port: 22, User: "root", Password: "x"}
	// Dial will fail (no server); we only assert helper selection does not demand key path first.
	_, err := sshClientBootstrap(in)
	if err == nil {
		t.Fatal("expected dial fail without server")
	}
	if strings.Contains(err.Error(), "private key path required") {
		t.Fatalf("bootstrap with password should not require key path: %v", err)
	}
}

func TestSSHClientKeyOnlyRequiresKey(t *testing.T) {
	_, err := sshClientKeyOnly(ProvisionIn{Host: "203.0.113.50", Port: 22, User: "root", Password: "x"})
	if err == nil || !strings.Contains(err.Error(), "private key") {
		t.Fatalf("want key required, got %v", err)
	}
}
