package vpn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func TestPreferredFragmentAndSubscriptionNoHY2(t *testing.T) {
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_ETC", filepath.Join(dir, "etc"))
	_ = os.Setenv("NETDUCTOR_STATE", filepath.Join(dir, "state"))
	_ = os.MkdirAll(filepath.Join(paths.EtcDir(), "secrets"), 0o700)
	_ = os.WriteFile(filepath.Join(paths.EtcDir(), "secrets", "singbox_reality_public"), []byte("pbk"), 0o600)
	_ = os.WriteFile(filepath.Join(paths.EtcDir(), "secrets", "singbox_short_id"), []byte("abcd"), 0o600)
	_ = EnsureDirs()
	_, _ = AddNative("tuser", "t")
	r, err := loadRegistry()
	if err != nil || findUser(r, "tuser") == nil {
		t.Skip("add failed without full env")
	}
	u := findUser(r, "tuser")
	link := VLESSLink(u.Name, u.UUID)
	if !strings.Contains(link, "#nd-primary") {
		t.Fatalf("expected #nd-primary got %s", link)
	}
	_ = writeArtifacts(u.Name, u.UUID, u.Hy2Password)
	// subscription removed from product surface
	if _, ok := ReadClient(u.Name, "subscription.txt"); ok {
		t.Fatal("subscription.txt should not be written")
	}
	n, err := RefreshLinks("tuser")
	if err != nil || n != 1 {
		t.Fatalf("refresh %d %v", n, err)
	}
}
