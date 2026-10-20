package vpn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPruneOrphanSubTokens(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	// write store with live + orphan
	s := subStore{Entries: []subEntry{
		{Token: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", User: "Pavel"},
		{Token: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", User: "Ghost"},
	}}
	if err := saveSub(s); err != nil {
		t.Fatal(err)
	}
	orphans := ListOrphanSubUsers([]string{"Pavel"})
	if len(orphans) != 1 || orphans[0] != "Ghost" {
		t.Fatalf("orphans=%v", orphans)
	}
	n, err := PruneOrphanSubTokens([]string{"Pavel"})
	if err != nil || n != 1 {
		t.Fatalf("n=%d err=%v", n, err)
	}
	if got := ListOrphanSubUsers([]string{"Pavel"}); len(got) != 0 {
		t.Fatalf("still orphans %v", got)
	}
	// file exists
	if _, err := os.Stat(filepath.Join(dir, "vpn_sub_tokens.json")); err != nil {
		t.Fatal(err)
	}
}
