package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SetBackupPeer writes backup.offsite for SCP push to peer VPS.
// Example: root@92.255.77.253:/var/lib/netductor/backups/peers/core/
func SetBackupPeer(scpTarget string, scpOpts string) error {
	scpTarget = strings.TrimSpace(scpTarget)
	if scpTarget == "" || !strings.Contains(scpTarget, ":") {
		return fmt.Errorf("target like root@host:/var/lib/netductor/backups/peers/core/")
	}
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	nl := string([]byte{10})
	body := "# Cross-VPS backup peer (scp)" + nl + "method=scp" + nl + "target=" + scpTarget + nl
	if scpOpts != "" {
		body += "scp_opts=" + scpOpts + nl
	} else {
		body += "scp_opts=-o StrictHostKeyChecking=accept-new -o BatchMode=yes" + nl
	}
	return os.WriteFile(filepath.Join(paths.EtcDir(), "backup.offsite"), []byte(body), 0o600)
}

func BackupPeerStatus() string {
	m, t, e := loadOffsite()
	if m == "" {
		return "offsite: not configured"
	}
	return fmt.Sprintf("offsite method=%s target=%s extra=%s", m, t, e)
}
