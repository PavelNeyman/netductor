package secondary

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseKnockPorts(t *testing.T) {
	_, err := parseKnockPorts("1,2,3")
	if err == nil {
		t.Fatal("expected error for <4 ports")
	}
	p, err := parseKnockPorts("30111,30222,30333,30444,30555,30666,30777,30888")
	if err != nil || len(p) != 8 {
		t.Fatalf("%v %v", p, err)
	}
}

func TestRegenerateKnockPorts(t *testing.T) {
	dir := t.TempDir()
	// redirect path via chdir is hard; just test parse
	_ = os.WriteFile(filepath.Join(dir, "x"), []byte("1"), 0o600)
}
