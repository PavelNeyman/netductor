package vpn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidName(t *testing.T) {
	if !ValidName("alice") || !ValidName("a_b-1") {
		t.Fatal("expected valid")
	}
	if ValidName("") || ValidName("bad name") || ValidName("../x") {
		t.Fatal("expected invalid")
	}
}

func TestLoadRegistryRejectsMissingOrNullUsersArray(t *testing.T) {
	for _, raw := range []string{`{}`, `{"users":null}`, `{"users":{}}`,
	} {
		dir := t.TempDir()
		t.Setenv("NETDUCTOR_ETC", dir)
		if err := os.WriteFile(filepath.Join(dir, "vpn-users.json"), []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadRegistry(); err == nil {
			t.Fatalf("loadRegistry accepted malformed registry %s", raw)
		}
	}
}

func TestRegistryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("NETDUCTOR_ETC", dir)
	defer os.Unsetenv("NETDUCTOR_ETC")
	_ = os.MkdirAll(filepath.Join(dir, "secrets"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, "secrets", "singbox_reality_public"), []byte("pub\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "secrets", "singbox_reality_private"), []byte("priv\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "secrets", "singbox_short_id"), []byte("abcd\n"), 0o600)
	// ApplyConfig needs sing-box optional — AddNative may fail apply; still writes user
	out, err := AddNative("tester", "note")
	if out == "" && err != nil {
		// if apply fails still may return name uuid
	}
	users, err := ListNative()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, u := range users {
		if u.Name == "tester" {
			found = true
		}
	}
	if !found {
		t.Fatalf("user not found after add: %v out=%s err=%v", users, out, err)
	}
}
