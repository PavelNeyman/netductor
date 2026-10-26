package operator

import "testing"

func TestParseSSHPort(t *testing.T) {
	n, err := ParseSSHPort("")
	if err != nil || n != 0 {
		t.Fatalf("empty should default, got %d %v", n, err)
	}
	n, err = ParseSSHPort("52222")
	if err != nil || n != 52222 {
		t.Fatalf("valid: %d %v", n, err)
	}
	for _, bad := range []string{"abc", "22abc", "-1", "0", "65536", "1.5"} {
		if _, err := ParseSSHPort(bad); err == nil {
			t.Fatalf("%q should be rejected", bad)
		}
	}
}

func TestFleetParamsRejectsBadPort(t *testing.T) {
	_, err := NormalizeFleetParams(map[string]any{"do_secondary": true, "primary_ssh_port": "22abc"})
	if err == nil {
		t.Fatal("fleet must reject 22abc")
	}
	f, err := NormalizeFleetParams(map[string]any{"do_secondary": true, "primary_ssh_port": "2200"})
	if err != nil || f.Secondary.PrimarySSHPort != 2200 {
		t.Fatalf("valid fleet port: %+v %v", f.Secondary, err)
	}
}

func TestFieldsBadPortSentinel(t *testing.T) {
	s := SecondaryFromFields(func(k string) string {
		if k == "primary_ssh_port" {
			return "abc"
		}
		return ""
	})
	if s.PrimarySSHPort != -1 {
		t.Fatalf("bad field should be sentinel, got %d", s.PrimarySSHPort)
	}
}
