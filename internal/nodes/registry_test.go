package nodes

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeHostname(t *testing.T) {
	got := NormalizeHostname("ND Core 01!")
	if got != "nd-core-01" {
		t.Fatalf("got %q", got)
	}
	if NormalizeRole("core") != "primary" || NormalizeRole("relay") != "secondary" {
		t.Fatalf("NormalizeRole core/relay")
	}
}

func TestStableIDRename(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_STATE", dir)
	_ = os.Setenv("NETDUCTOR_ETC", filepath.Join(dir, "etc"))
	t.Cleanup(func() {
		_ = os.Unsetenv("NETDUCTOR_STATE")
		_ = os.Unsetenv("NETDUCTOR_ETC")
	})
	_ = os.MkdirAll(filepath.Join(dir, "etc"), 0o755)

	id := LocalStableID()
	if id == "" {
		t.Fatal("empty id")
	}
	if LocalStableID() != id {
		t.Fatal("id must be stable")
	}
	_ = SelfRegisterLocal("nd-primary-1", "primary", "1.2.3.4")
	_, err := SetDesiredHostname(id, "nd-primary-nl01")
	if err != nil {
		t.Fatal(err)
	}
	WriteLocalHostname("nd-primary-nl01")
	if err := SyncLocalHostname(); err != nil {
		t.Fatal(err)
	}
	list, _ := List()
	if len(list) != 1 {
		t.Fatalf("%+v", list)
	}
	if list[0].ID != id || list[0].Hostname != "nd-primary-nl01" {
		t.Fatalf("%+v", list[0])
	}
}

func TestSetCapability(t *testing.T) {
	// uses default paths — skip if no writable state in CI; Upsert then set
	id := "cap-test-node"
	_, err := UpsertFromDevice(Node{ID: id, Hostname: "nd-egress-ru-msk", Role: "secondary", Kind: "vps"})
	if err != nil {
		t.Skip(err)
	}
	n, err := SetCapability(id, "vpn_ingress", true)
	if err != nil {
		t.Fatal(err)
	}
	if n.Labels["cap.vpn_ingress"] != "1" {
		t.Fatalf("%v", n.Labels)
	}
	caps, err := CapabilityList(id)
	if err != nil || len(caps) == 0 {
		t.Fatal(caps, err)
	}
}
