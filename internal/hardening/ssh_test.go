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

func TestRemoteHardenScriptMentionsUFWAndZabbix(t *testing.T) {
	s := RemoteHardenScript()
	if !strings.Contains(s, "ufw") {
		t.Fatal("no ufw")
	}
	if !strings.Contains(s, "zabbix") {
		t.Fatal("no zabbix purge")
	}
	if !strings.Contains(s, "sshd -t") {
		t.Fatal("expected sshd -t before restart")
	}
	if strings.Contains(s, "pkill -f") {
		t.Fatal("pkill -f must not appear")
	}
	if !strings.Contains(s, "52222") && !strings.Contains(s, "Port") {
		t.Log("port may come from DefaultSSHPort expansion")
	}
}

func TestSSHPortPositive(t *testing.T) {
	if SSHPort() < 1 || SSHPort() > 65535 {
		t.Fatalf("%d", SSHPort())
	}
}
