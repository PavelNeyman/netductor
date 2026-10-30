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
	if !strings.Contains(s, "Port ") {
		t.Fatal("missing Port")
	}
}

func TestRemoteHardenPrepareFailClosed(t *testing.T) {
	s := RemoteHardenPrepareScript()
	if !strings.Contains(s, "set -euo pipefail") {
		t.Fatal("want strict shell")
	}
	if !strings.Contains(s, "sshd -t") {
		t.Fatal("expected sshd -t")
	}
	if !strings.Contains(s, RollbackTimer) {
		t.Fatal("expected rollback timer")
	}
	if !strings.Contains(s, "rm -rf \"$ND_BAK\"") && !strings.Contains(s, "rm -rf \"$ND_BAK\"") {
		// template uses $ND_BAK
		if !strings.Contains(s, "rm -rf") {
			t.Fatal("fresh backup dir")
		}
	}
	if strings.Contains(s, "pkill") {
		t.Fatal("no pkill")
	}
	if strings.Contains(s, "deny 22") {
		t.Fatal("prepare must not deny port 22")
	}
	// restart must not be swallowed with || true at the end
	if strings.Contains(s, "restart ssh 2>/dev/null || true") || strings.Contains(s, "restart sshd 2>/dev/null || true") {
		t.Fatal("restart errors must not be swallowed")
	}
	if !strings.Contains(s, "exit 1") {
		t.Fatal("expected explicit restart failure path")
	}
	if !strings.Contains(s, "ufw allow") {
		t.Fatal("expected ufw allow new port")
	}
	if !strings.Contains(s, "sshd_config.bak") {
		t.Fatal("mandatory ssh backup")
	}
}

func TestRemoteHardenFinalizeDoesNotCancelRollback(t *testing.T) {
	s := RemoteHardenFinalizeScript()
	if strings.Contains(s, "systemctl stop "+RollbackTimer) {
		t.Fatal("finalize must not cancel rollback timer")
	}
	if strings.Contains(s, "rm -f /etc/systemd/system/"+RollbackUnit) {
		t.Fatal("finalize must not remove rollback unit")
	}
	if !strings.Contains(s, "PasswordAuthentication no") {
		t.Fatal("password off")
	}
	if !strings.Contains(s, "deny 22") {
		t.Fatal("finalize should deny 22")
	}
	if strings.Contains(s, "restart sshd 2>/dev/null || true") {
		t.Fatal("restart must not be swallowed")
	}
	if strings.Contains(s, "pkill") {
		t.Fatal("no pkill")
	}
}

func TestRemoteHardenCancelRollbackOnly(t *testing.T) {
	s := RemoteHardenCancelRollbackScript()
	if !strings.Contains(s, "systemctl stop "+RollbackTimer) {
		t.Fatal("cancel must stop timer")
	}
	if !strings.Contains(s, RollbackUnit) {
		t.Fatal("remove unit")
	}
}

func TestRollbackUnitRestoresUFW(t *testing.T) {
	s := RemoteHardenPrepareScript()
	if !strings.Contains(s, "ufw-22-state") {
		t.Fatal("ufw 22 state snapshot")
	}
	if !strings.Contains(s, "user.rules") {
		t.Fatal("ufw rules restore in rollback")
	}
}

func TestSSHPortPositive(t *testing.T) {
	if SSHPort() < 1 || SSHPort() > 65535 {
		t.Fatalf("%d", SSHPort())
	}
}
