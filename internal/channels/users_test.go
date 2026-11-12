package channels

import "testing"

func TestCollectUsersWhenNoAPI(t *testing.T) {
	r := CollectUsers()
	// Without local clash_api and without secondary traffic in registry, unavailable.
	if r.Available && len(r.Users) == 0 && r.Source == "" {
		// ambiguous
	}
	if !r.Available && r.Note == "" {
		t.Fatal("expected note when unavailable")
	}
	s := FormatUsersText(r)
	if s == "" {
		t.Fatal("format empty")
	}
}
