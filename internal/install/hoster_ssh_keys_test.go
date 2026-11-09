package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanHosterSSHAuthorizedKeysMarkers(t *testing.T) {
	dir := t.TempDir()
	ak := filepath.Join(dir, "authorized_keys")
	content := `ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakeOperatorKey operator@mac
from="85.198.118.171,85.198.75.83" ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC supportAccessKey
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOtherKey user@box
`
	if err := os.WriteFile(ak, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	// Call logic by reading same markers against temp file — unit via helper
	b, _ := os.ReadFile(ak)
	var hits int
	for _, line := range strings.Split(string(b), "\n") {
		low := strings.ToLower(line)
		for _, m := range hosterSSHKeyMarkers {
			if strings.Contains(low, strings.ToLower(m)) {
				hits++
				break
			}
		}
	}
	if hits < 1 {
		t.Fatal("expected hoster key hit")
	}
}
