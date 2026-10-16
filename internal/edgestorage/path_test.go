package edgestorage

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeDeviceID(t *testing.T) {
	if SafeDeviceID("../etc/passwd") != "passwd" && SafeDeviceID("../etc/passwd") != "etcpasswd" {
		// Base of ../etc/passwd is passwd
		got := SafeDeviceID("../etc/passwd")
		if got != "passwd" {
			t.Fatalf("got %q", got)
		}
	}
	got := SafeDeviceID("tr1200-shop_1")
	if got != "tr1200-shop_1" {
		t.Fatal(got)
	}
	if strings.Contains(Dir("../../../tmp/x"), "/tmp/") {
		t.Fatalf("Dir escaped: %s", Dir("../../../tmp/x"))
	}
	if !strings.HasPrefix(Dir("ok-id"), "/var/lib/netductor/edge-backups/") {
		t.Fatal(Dir("ok-id"))
	}
}

func TestClampOfflineDest(t *testing.T) {
	d := clampOfflineDest("/etc/evil", "dev1")
	if d != filepath.Join("/var/lib/netductor/edge-offline", "dev1") {
		t.Fatal(d)
	}
	ok := filepath.Join("/var/lib/netductor/edge-offline", "dev1", "extra")
	if clampOfflineDest(ok, "dev1") != ok {
		t.Fatal(clampOfflineDest(ok, "dev1"))
	}
}
