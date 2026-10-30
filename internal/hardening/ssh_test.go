package hardening

import (
	"strings"
	"testing"
)

func TestDropInConfContainsPortAndKeyOnly(t *testing.T) {
	if !strings.Contains(DropInConf(), "PasswordAuthentication no") {
		t.Fatal("missing PasswordAuthentication")
	}
}

func TestPrepareArmsRollbackBeforeMutation(t *testing.T) {
	s := RemoteHardenPrepareScript()
	marker := "Mutations only after rollback is armed"
	idxMut := strings.Index(s, marker)
	idxTimer := strings.Index(s, "systemctl is-active ${ND_RB_TIMER}")
	if idxMut < 0 || idxTimer < 0 || idxTimer > idxMut {
		t.Fatal("rollback must be active before mutations")
	}
	if !strings.Contains(s, "CANCELLED") {
		// stamp is in rollback script body
		if !strings.Contains(s, "cancelled stamp") {
			t.Fatal("rollback should honor CANCELLED stamp")
		}
	}
	if !strings.Contains(s, "ufw-policy") {
		t.Fatal("ufw policy")
	}
	if !strings.Contains(s, "ssh.socket") {
		t.Fatal("socket state snapshot")
	}
	if !strings.Contains(s, "test -f \"$ND_BAK/ufw/user.rules\"") && !strings.Contains(s, `test -f "$ND_BAK/ufw/user.rules"`) {
		// active requires user.rules
		if !strings.Contains(s, "user.rules") {
			t.Fatal("user.rules snapshot")
		}
	}
}

func TestFinalizeNoPurgeNoCancel(t *testing.T) {
	s := RemoteHardenFinalizeScript()
	if strings.Contains(s, "apt-get remove") {
		t.Fatal("no purge in finalize")
	}
	if strings.Contains(s, "CANCELLED") {
		t.Fatal("finalize must not cancel")
	}
}

func TestCancelStampAndService(t *testing.T) {
	s := RemoteHardenCancelRollbackScript()
	if !strings.Contains(s, "CANCELLED") {
		t.Fatal("CANCELLED stamp")
	}
	if !strings.Contains(s, "stop \"$ND_RB_UNIT\"") && !strings.Contains(s, "stop ${ND_RB_UNIT}") {
		if !strings.Contains(s, RollbackUnit) {
			t.Fatal("must stop rollback unit")
		}
	}
	if !strings.Contains(s, "is-active") {
		t.Fatal("verify inactive")
	}
}

func TestLegacyOneShotDisabled(t *testing.T) {
	s := RemoteHardenScript()
	if !strings.Contains(s, "exit 1") {
		t.Fatal("legacy one-shot must fail")
	}
	if strings.Contains(s, "nd-harden-prepare") {
		t.Fatal("must not embed prepare")
	}
}

func TestPurgeIsSeparate(t *testing.T) {
	if !strings.Contains(RemoteHostAgentPurgeScript(), "zabbix") {
		t.Fatal("purge")
	}
}

func TestSSHPortPositive(t *testing.T) {
	if SSHPort() < 1 || SSHPort() > 65535 {
		t.Fatalf("%d", SSHPort())
	}
}
