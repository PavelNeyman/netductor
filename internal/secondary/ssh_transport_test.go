package secondary

import "testing"

func TestSSHTransportPasswordOnlyInBootstrap(t *testing.T) {
	tr := NewSSHTransportBootstrap("203.0.113.10", 22, "root", "secret", "/tmp/id", "")
	pw, err := tr.BootstrapPassword()
	if err != nil || pw != "secret" {
		t.Fatalf("bootstrap password: %q %v", pw, err)
	}
	if err := tr.PromoteToKeyVerified(); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.BootstrapPassword(); err == nil {
		t.Fatal("password must be unavailable after key_verified")
	}
	if tr.bootstrapPass != "" {
		t.Fatal("password must be zeroed")
	}
	if err := tr.PromoteToHardened(52222); err != nil {
		t.Fatal(err)
	}
	if tr.Mode != SSHTransportHardened || tr.Port != 52222 {
		t.Fatalf("%s port=%d", tr.Mode, tr.Port)
	}
}

func TestSSHTransportCannotPromoteWithoutKey(t *testing.T) {
	tr := NewSSHTransportBootstrap("h", 22, "root", "p", "", "")
	if err := tr.PromoteToKeyVerified(); err == nil {
		t.Fatal("want error")
	}
}

func TestSSHTransportNoSkipToHardened(t *testing.T) {
	tr := NewSSHTransportBootstrap("h", 22, "root", "p", "/k", "")
	if err := tr.PromoteToHardened(52222); err == nil {
		t.Fatal("must not skip key_verified")
	}
}
