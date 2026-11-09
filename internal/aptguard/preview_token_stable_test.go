package aptguard

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreviewTokenStableAcrossRebuilds(t *testing.T) {
	root := t.TempDir()
	srcDir := filepath.Join(root, "etc", "apt", "sources.list.d")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// three unknown mirrors → multiple quarantine changes; map order must not matter
	files := map[string]string{
		"Zzz.sources": "Types: deb\nURIs: http://mirror.example/zzz\nSuites: trixie\nComponents: main\n",
		"Aaa.sources": "Types: deb\nURIs: http://mirror.example/aaa\nSuites: trixie\nComponents: main\n",
		"Mmm.sources": "Types: deb\nURIs: http://mirror.example/mmm\nSuites: trixie\nComponents: main\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(srcDir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	os.WriteFile(filepath.Join(root, "etc", "os-release"), []byte("VERSION_CODENAME=trixie\n"), 0o644)

	pol := DefaultPolicy()
	var tokens []string
	for i := 0; i < 20; i++ {
		pv, err := BuildPreview("test-host", root, pol)
		if err != nil {
			t.Fatal(err)
		}
		if len(pv.Changes) < 3 {
			t.Fatalf("expected >=3 changes, got %d: %+v", len(pv.Changes), pv.Changes)
		}
		tokens = append(tokens, pv.Token)
	}
	for i := 1; i < len(tokens); i++ {
		if tokens[i] != tokens[0] {
			t.Fatalf("token unstable: first=%s iter[%d]=%s", tokens[0], i, tokens[i])
		}
	}
}
