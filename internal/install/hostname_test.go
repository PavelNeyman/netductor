package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyHostnameEnv(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_ETC", dir)
	_ = os.Setenv("NETDUCTOR_HOSTNAME", "nd-primary-test99")
	t.Cleanup(func() {
		_ = os.Unsetenv("NETDUCTOR_ETC")
		_ = os.Unsetenv("NETDUCTOR_HOSTNAME")
	})
	applyHostname("primary")
	b, err := os.ReadFile(filepath.Join(dir, "node_id"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "nd-primary-test99") {
		t.Fatalf("%q", b)
	}
}

func TestHostsHasNameEmptyLines(t *testing.T) {
	hosts := "\n# comment only\n\n127.0.0.1 localhost\n"
	if hostsHasName(hosts, "nd-secondary-00") {
		t.Fatal("should not match")
	}
	if !hostsHasName(hosts+"127.0.1.1 nd-secondary-00\n", "nd-secondary-00") {
		t.Fatal("should match")
	}
}
