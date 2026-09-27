package ndconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkipsForbidden(t *testing.T) {
	dir := t.TempDir()
	conf := filepath.Join(dir, "netductor.conf")
	_ = os.WriteFile(conf, []byte("DOMAIN=ok.example\nNETDUCTOR_PLAIN_AGENT=1\nLE_EMAIL=a@b.c\n"), 0o600)
	_ = os.Setenv("NETDUCTOR_CONF", conf)
	_ = os.Unsetenv("NETDUCTOR_PLAIN_AGENT")
	_ = os.Unsetenv("NETDUCTOR_DOMAIN")
	_ = os.Unsetenv("NETDUCTOR_LE_EMAIL")
	t.Cleanup(func() {
		_ = os.Unsetenv("NETDUCTOR_CONF")
		_ = os.Unsetenv("NETDUCTOR_DOMAIN")
		_ = os.Unsetenv("NETDUCTOR_LE_EMAIL")
	})
	Load()
	if os.Getenv("NETDUCTOR_PLAIN_AGENT") != "" {
		t.Fatal("forbidden key loaded")
	}
	if os.Getenv("NETDUCTOR_DOMAIN") != "ok.example" {
		t.Fatalf("DOMAIN not loaded: %q", os.Getenv("NETDUCTOR_DOMAIN"))
	}
}
