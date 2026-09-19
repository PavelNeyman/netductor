package nvr

import "testing"

func TestPathUnderRoot(t *testing.T) {
	if !PathUnderRoot("/var/nvr", "/var/nvr/a.mp4") {
		t.Fatal("child")
	}
	if PathUnderRoot("/var/nvr", "/var/nvr-evil/a.mp4") {
		t.Fatal("prefix trap")
	}
	if PathUnderRoot("/var/nvr", "/etc/passwd") {
		t.Fatal("outside")
	}
}
