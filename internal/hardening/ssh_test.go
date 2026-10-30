package hardening

import (
	"strings"
	"testing"
)

func TestDropInConfContainsPortAndKeyOnly(t *testing.T) {
	s := DropInConf()
	if !strings.Contains(s, "PasswordAuthentication no") {
		t.Fatal("missing PasswordAuthentication")
	}
}

func TestPrepareArmsRollbackBeforeMutation(t *testing.T) {
	s := RemoteHardenPrepareScript()
	marker := "Mutations only after rollback is armed"
	idxMutSection := strings.Index(s, marker)
	idxTimer := strings.Index(s, "systemctl is-active ${ND_RB_TIMER}")
	if idxMutSection < 0 {
		t.Fatal("missing mutation section marker")
	}
	if idxTimer < 0 || idxTimer > idxMutSection {
		t.Fatal("rollback timer must be active before mutations")
	}
	// First ufw allow of ND_PORT in mutation section
	rest := s[idxMutSection:]
	if !strings.Contains(rest, "ufw allow") {
		t.Fatal("mutation section must allow ports")
	}
	if !strings.Contains(s, "ufw-policy") {
		t.Fatal("ufw policy snapshot")
	}
	if strings.Contains(s, "restart sshd 2>/dev/null || true") {
		t.Fatal("no swallowed restart")
	}
	if strings.Contains(s, "pkill") {
		t.Fatal("no pkill")
	}
}

func TestFinalizeNoPurgeNoCancel(t *testing.T) {
	s := RemoteHardenFinalizeScript()
	if strings.Contains(s, "apt-get remove") {
		t.Fatal("purge must not be in finalize")
	}
	if strings.Contains(s, "systemctl stop "+RollbackTimer) {
		t.Fatal("finalize must not cancel timer")
	}
	if !strings.Contains(s, "deny 22") {
		t.Fatal("deny 22")
	}
}

func TestCancelVerifiesInactive(t *testing.T) {
	s := RemoteHardenCancelRollbackScript()
	if !strings.Contains(s, "is-active") {
		t.Fatal("must check timer inactive")
	}
	if !strings.Contains(s, "systemctl stop "+RollbackTimer) {
		t.Fatal("stop timer")
	}
}

func TestPurgeIsSeparate(t *testing.T) {
	s := RemoteHostAgentPurgeScript()
	if !strings.Contains(s, "zabbix") {
		t.Fatal("zabbix purge")
	}
}

func TestSSHPortPositive(t *testing.T) {
	if SSHPort() < 1 || SSHPort() > 65535 {
		t.Fatalf("%d", SSHPort())
	}
}
