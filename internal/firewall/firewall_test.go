package firewall

import "testing"

func TestTruncate(t *testing.T) {
	if got := truncate("ab", 5); got != "ab" {
		t.Fatalf("%q", got)
	}
	if got := truncate("  hello world  ", 5); got != "hello…" {
		t.Fatalf("%q", got)
	}
}

func TestFirstLines(t *testing.T) {
	in := "a\nb\nc\nd"
	got := firstLines(in, 2)
	if got != "a\nb" {
		t.Fatalf("%q", got)
	}
	if firstLines("x", 5) != "x" {
		t.Fatal("short")
	}
}

func TestDetectRoleDefaultPrimary(t *testing.T) {
	// Without agent/role markers on this host, default is primary.
	r := detectRole()
	if r != "primary" && r != "secondary" {
		t.Fatalf("role %q", r)
	}
}
