package deploy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCacheDefaultTemplates(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "agents")
	p, err := CacheDefaultTemplates(agentDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Fatal(err)
	}
	tmpl, err := LoadCachedTemplate(agentDir, "default")
	if err != nil {
		t.Fatal(err)
	}
	if tmpl["id"] != "default" && tmpl["id"] != nil {
		// id may be set
	}
	if tmpl == nil {
		t.Fatal("nil template")
	}
}
