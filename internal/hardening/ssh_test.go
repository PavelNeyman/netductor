package hardening

import (
	"strings"
	"testing"
)

func TestDropInConf(t *testing.T) {
	if !strings.Contains(DropInConf(), "PasswordAuthentication no") {
		t.Fatal("key-only")
	}
}

func TestPrepareOrderAndFlock(t *testing.T) {
	s := RemoteHardenPrepareScript()
	if !strings.Contains(s, "previous SSH rollback is not terminal") ||
		!strings.Contains(s, "previous SSH rollback has no terminal marker") {
		t.Fatal("must refuse prepare while a prior rollback is unresolved")
	}
	if !strings.Contains(s, "flock 9") {
		t.Fatal("prepare must hold the persistent backup lock")
	}
	cleanup := strings.Index(s, `find "$ND_BAK" -mindepth 1 -maxdepth 1 ! -name lock -exec rm -rf -- {} +`)
	backup := strings.Index(s, "# --- Mandatory SSH backup ---")
	if cleanup < 0 || backup < 0 || cleanup > backup {
		t.Fatal("must clear only terminal transaction payload before taking a new snapshot")
	}
	if strings.Contains(s, `rm -rf "$ND_BAK"`) {
		t.Fatal("must preserve the persistent backup lock inode")
	}
	if !strings.Contains(s, "Mutations only after rollback is armed") {
		t.Fatal("mutation marker")
	}
	idxTimer := strings.Index(s, "systemctl start ${ND_RB_TIMER}")
	idxMut := strings.Index(s, "Mutations only after rollback is armed")
	if idxTimer < 0 || idxMut < 0 || idxTimer > idxMut {
		t.Fatal("arm rollback timer before mutating SSH or firewall")
	}
	if !strings.Contains(s, "user6.rules") {
		t.Fatal("user6 snapshot path")
	}
	if strings.Contains(s, "ufw allow 22/tcp comment netductor-ssh-bootstrap || true") {
		t.Fatal("bootstrap SSH allow must not suppress failure")
	}
}

func TestCancelUsesFlockAndStamp(t *testing.T) {
	s := RemoteHardenCancelRollbackScript()
	if !strings.Contains(s, "flock 9") {
		t.Fatal("cancel flock")
	}
	if !strings.Contains(s, "CANCELLED") {
		t.Fatal("CANCELLED")
	}
	if !strings.Contains(s, RollbackUnit) {
		t.Fatal("stop unit")
	}
	if !strings.Contains(s, "still active") {
		t.Fatal("verify inactive")
	}
}

func TestFinalizeClean(t *testing.T) {
	s := RemoteHardenFinalizeScript()
	if strings.Contains(s, "apt-get remove") || strings.Contains(s, "CANCELLED") {
		t.Fatal("finalize scope")
	}
}

func TestLegacyDisabled(t *testing.T) {
	if !strings.Contains(RemoteHardenScript(), "exit 1") {
		t.Fatal("legacy")
	}
}

func TestSSHPortPositive(t *testing.T) {
	if SSHPort() < 1 || SSHPort() > 65535 {
		t.Fatalf("%d", SSHPort())
	}
}

func TestRollbackLockIsBoundedAndRetriesAfterContention(t *testing.T) {
	s := RemoteHardenPrepareScript()
	if !strings.Contains(s, "flock -w 60 9") {
		t.Fatal("rollback must not wait indefinitely for the transaction lock")
	}
	if !strings.Contains(s, "netductor-ssh-rollback-emergency") {
		t.Fatal("lock timeout must preserve emergency management ingress")
	}
	if !strings.Contains(s, "ROLLBACK_FAILED") {
		t.Fatal("lock timeout must remain explicitly recoverable")
	}
	if !strings.Contains(s, "Restart=on-failure") || !strings.Contains(s, "RestartSec=15s") ||
		!strings.Contains(s, "TimeoutStartSec=90s") || !strings.Contains(s, "StartLimitIntervalSec=0") {
		t.Fatal("rollback unit must retry bounded attempts until the lock clears")
	}
}
