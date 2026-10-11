package version

import "testing"

func TestReleaseNonEmpty(t *testing.T) {
	if Release == "" {
		t.Fatal("empty Release")
	}
}
