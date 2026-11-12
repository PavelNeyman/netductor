package channels

import "testing"

func TestCollectUsersUnsupported(t *testing.T) {
	r := CollectUsers()
	if r.Available {
		t.Skip("local stats API unexpectedly up")
	}
	if r.Note == "" {
		t.Fatal("expected unsupported note")
	}
	s := FormatUsersText(r)
	if s == "" {
		t.Fatal("format empty")
	}
}
