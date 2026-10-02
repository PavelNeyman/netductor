package vpn

import (
	"testing"
)

func TestSubTokenIssueResolve(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	tok, err := IssueSubToken("alice")
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) < 32 {
		t.Fatalf("short token %q", tok)
	}
	if ResolveSubToken(tok) != "alice" {
		t.Fatalf("resolve")
	}
	if ResolveSubToken("too-short") != "" {
		t.Fatal("short must miss")
	}
	// rotate
	tok2, err := IssueSubToken("alice")
	if err != nil {
		t.Fatal(err)
	}
	if tok2 == tok {
		t.Fatal("rotate should change token")
	}
	if ResolveSubToken(tok) != "" {
		t.Fatal("old token must die")
	}
	if ResolveSubToken(tok2) != "alice" {
		t.Fatal("new token")
	}
}
