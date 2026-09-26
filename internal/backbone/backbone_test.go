package backbone

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInitPrimaryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_BACKBONE_DIR", dir)
	s, err := InitPrimary(51820, "1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if s.PrimaryPublicKey == "" || s.SecondaryPrivate == "" {
		t.Fatal("keys empty")
	}
	if _, err := base64.StdEncoding.DecodeString(s.PrimaryPrivate); err != nil {
		t.Fatal(err)
	}
	raw, err := ExportForSecondary()
	if err != nil {
		t.Fatal(err)
	}
	var exported State
	if err := json.Unmarshal(raw, &exported); err != nil {
		t.Fatal(err)
	}
	if exported.PrimaryPrivate != "" {
		t.Fatal("primary private must not be exported")
	}
	if exported.SecondaryPrivate == "" {
		t.Fatal("secondary private required on export")
	}
	dir2 := t.TempDir()
	t.Setenv("NETDUCTOR_BACKBONE_DIR", dir2)
	if err := InitSecondaryFromState(&exported, "1.2.3.4:51820"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir2, InterfaceName+".conf")); err != nil {
		t.Fatal(err)
	}
}
