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
	if !strings.Contains(s, "flock 9") {
		t.Fatal("rollback must use flock")
	}
	if !strings.Contains(s, "Mutations only after rollback is armed") {
		t.Fatal("mutation marker")
	}
	idxTimer := strings.Index(s, "systemctl is-active ${ND_RB_TIMER}")
	idxMut := strings.Index(s, "Mutations only after rollback is armed")
	if idxTimer < 0 || idxMut < 0 || idxTimer > idxMut {
		t.Fatal("arm before mutate")
	}
	if !strings.Contains(s, "user6.rules") {
		t.Fatal("user6 snapshot path")
	}
	if strings.Contains(s, "ufw allow 22/tcp comment netductor-ssh-bootstrap || true") &&
		strings.Contains(s[strings.Index(s, "NDRBSH"):], "ufw allow 22/tcp comment netductor-ssh-bootstrap") {
		// heuristic allow in rollback body is forbidden; prepare mutation may still allow 22
		rb := s[strings.Index(s, "NDRBSH") : strings.LastIndex(s, "NDRBSH")+6]
		if strings.Contains(rb, "netductor-ssh-bootstrap") {
			t.Fatal("rollback must not heuristic-allow 22")
		}
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
