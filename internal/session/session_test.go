package session

import (
	"time"
	"os"
	"path/filepath"
	"testing"
)

func setup(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	_ = os.Setenv("NETDUCTOR_ETC", dir)
	t.Cleanup(func() { _ = os.Unsetenv("NETDUCTOR_ETC") })
	_ = os.MkdirAll(filepath.Join(dir, "sessions"), 0o700)
}

func TestCreateValidRevoke(t *testing.T) {
	setup(t)
	tok, exp, err := Create(1, "test", "127.0.0.1")
	if err != nil || tok == "" || exp == 0 {
		t.Fatal(err, tok, exp)
	}
	if !Valid(tok) {
		t.Fatal("should be valid")
	}
	ents, _ := os.ReadDir(Dir())
	for _, e := range ents {
		if e.Name() == tok {
			t.Fatal("token must not be filename")
		}
	}
	Revoke(tok)
	if Valid(tok) {
		t.Fatal("revoked")
	}
}

func TestClampHours(t *testing.T) {
	if clampHours(0) != DefaultHours {
		t.Fatal()
	}
	if clampHours(999) != MaxHours {
		t.Fatal()
	}
}

func TestTokenFromAuth(t *testing.T) {
	if TokenFromAuth("Bearer abc", "") != "abc" {
		t.Fatal()
	}
	if TokenFromAuth("", "nd_session=xyz; Path=/") != "xyz" {
		t.Fatal(TokenFromAuth("", "nd_session=xyz; Path=/"))
	}
	if TokenFromAuth("", "") != "" {
		t.Fatal()
	}
}

func TestRevokeAll(t *testing.T) {
	setup(t)
	t1, _, _ := Create(1, "a", "")
	t2, _, _ := Create(1, "b", "")
	if err := RevokeAll(); err != nil {
		t.Fatal(err)
	}
	if Valid(t1) || Valid(t2) {
		t.Fatal()
	}
}

func TestMaxHoursEnforced(t *testing.T) {
	setup(t)
	_, exp, err := Create(999, "", "")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	max := now + int64(MaxHours)*3600
	if exp < now || exp > max+120 {
		t.Fatalf("exp=%d now=%d max=%d", exp, now, max)
	}
}
