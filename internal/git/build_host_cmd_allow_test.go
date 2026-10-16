package gitstore

import "testing"

func TestAllowedBuildHostCmd(t *testing.T) {
	if !allowedBuildHostCmd("backup_pull") || !allowedBuildHostCmd("recovery_arm") {
		t.Fatal("want allow")
	}
	if allowedBuildHostCmd("shell") || allowedBuildHostCmd("") || allowedBuildHostCmd("reboot") {
		t.Fatal("want deny")
	}
}
