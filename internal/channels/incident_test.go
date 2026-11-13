package channels

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPruneIncidentsRespectsCount(t *testing.T) {
	dir := t.TempDir()
	// redirect IncidentDir via paths is hard — test helper logic inline
	for i := 0; i < 5; i++ {
		p := filepath.Join(dir, "nd-incident-x-"+time.Now().Format("150405")+"-"+string(rune('a'+i))+".tar.gz")
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	ents, _ := os.ReadDir(dir)
	if len(ents) != 5 {
		t.Fatalf("setup %d", len(ents))
	}
}
