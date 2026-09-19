package nvr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClipTokenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("NETDUCTOR_STATE", dir)
	defer os.Unsetenv("NETDUCTOR_STATE")
	// paths.StateDir uses env
	p := filepath.Join(dir, "nvr", "segments", "c1", "a.mp4")
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	_ = os.WriteFile(p, []byte("x"), 0o600)
	tok, err := IssueClipToken("c1", p, 60)
	if err != nil || tok == "" {
		t.Fatal(err, tok)
	}
	got, ok := RedeemClipToken(tok)
	if !ok || got != p {
		t.Fatalf("%v %s", ok, got)
	}
	if _, ok := RedeemClipToken(tok); ok {
		t.Fatal("one-shot")
	}
}
