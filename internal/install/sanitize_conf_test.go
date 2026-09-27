package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeNetductorConf(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_ETC", dir)
	t.Cleanup(func() { _ = os.Unsetenv("NETDUCTOR_ETC") })
	conf := filepath.Join(dir, "netductor.conf")
	body := "DOMAIN=example.com\nLE_EMAIL=a@b.c\nNETDUCTOR_PLAIN_AGENT=1\nAPI_ALLOW_PUBLIC=1\nREDIRECT_BASE=https://i.example.com:8443\n"
	if err := os.WriteFile(conf, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	_ = os.Setenv("NETDUCTOR_PLAIN_AGENT", "1")
	removed := SanitizeNetductorConf()
	if len(removed) < 1 {
		t.Fatalf("expected removals, got %v", removed)
	}
	b, _ := os.ReadFile(conf)
	s := string(b)
	if strings.Contains(s, "PLAIN_AGENT") || strings.Contains(s, "ALLOW_PUBLIC") {
		t.Fatalf("footguns remain: %s", s)
	}
	if !strings.Contains(s, "DOMAIN=example.com") {
		t.Fatalf("lost DOMAIN: %s", s)
	}
	if os.Getenv("NETDUCTOR_PLAIN_AGENT") != "" {
		t.Fatal("env not cleared")
	}
}
