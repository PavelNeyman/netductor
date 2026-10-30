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

func TestRemoteHardenPrepareHasRollbackAndSshdT(t *testing.T) {
	s := RemoteHardenPrepareScript()
	if !strings.Contains(s, "sshd -t") {
		t.Fatal("expected sshd -t")
	}
	if !strings.Contains(s, "netductor-ssh-harden-rollback") {
		t.Fatal("expected rollback unit/timer")
	}
	if strings.Contains(s, "pkill") {
		t.Fatal("no pkill")
	}
	if strings.Contains(s, "deny 22") {
		t.Fatal("prepare must not deny port 22")
	}
	if !strings.Contains(s, "ufw allow") {
		t.Fatal("expected ufw allow new port")
	}
}

func TestRemoteHardenFinalizeDisablesPassword(t *testing.T) {
	s := RemoteHardenFinalizeScript()
	if !strings.Contains(s, "PasswordAuthentication no") {
		t.Fatal("password off")
	}
	if !strings.Contains(s, "deny 22") {
		t.Fatal("finalize should deny 22")
	}
	if !strings.Contains(s, "systemctl stop") {
		t.Fatal("cancel rollback timer")
	}
	if strings.Contains(s, "pkill") {
		t.Fatal("no pkill")
	}
}

func TestRemoteHardenScriptCombined(t *testing.T) {
	s := RemoteHardenScript()
	if !strings.Contains(s, "sshd -t") {
		t.Fatal("sshd -t")
	}
}

func TestSSHPortPositive(t *testing.T) {
	if SSHPort() < 1 || SSHPort() > 65535 {
		t.Fatalf("%d", SSHPort())
	}
}
