package secondary

import "testing"

func TestAllowedSecondaryCmd(t *testing.T) {
	for _, c := range []string{"reboot", "upgrade", "upgrade:v0.9.404", "mtls_refresh", "backup_pull", "ssh_keys", "ssh_allowlist:xxx", "restart:sing-box", "journal", "metrics", "backup_local"} {
		if !allowedSecondaryCmd(c) {
			t.Fatalf("want allow %q", c)
		}
	}
	for _, c := range []string{"", "shell", "rm -rf", "restart:sshd", "curl http://evil"} {
		if allowedSecondaryCmd(c) {
			t.Fatalf("must deny %q", c)
		}
	}
}
