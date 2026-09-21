package guest

import (
	"path/filepath"
	"testing"
	"time"
)

func TestGrantClampAndExpire(t *testing.T) {
	dir := t.TempDir()
	s, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	e, err := s.Grant("AA-BB-CC-DD-EE-FF", 48*time.Hour, "test")
	if err != nil {
		t.Fatal(err)
	}
	if e.ExpiresAt.Sub(time.Now()) > MaxGrant+time.Minute {
		t.Fatalf("clamp failed: %v", e.ExpiresAt)
	}
	if !s.IsAllowed("aa:bb:cc:dd:ee:ff") {
		t.Fatal("allowed")
	}
	_ = s.Revoke("aa:bb:cc:dd:ee:ff")
	if s.IsAllowed("aa:bb:cc:dd:ee:ff") {
		t.Fatal("revoked")
	}
}

func TestSessionCode(t *testing.T) {
	dir := t.TempDir()
	s, _ := OpenStore(dir)
	sess, err := s.EnsureSession("11:22:33:44:55:66")
	if err != nil || sess.Code == "" || sess.Token == "" {
		t.Fatalf("%v %+v", err, sess)
	}
	sess2, _ := s.EnsureSession("11:22:33:44:55:66")
	if sess2.Code != sess.Code {
		t.Fatal("should reuse session")
	}
	got, ok := s.LookupCode(sess.Code)
	if !ok || got.MAC != "11:22:33:44:55:66" {
		t.Fatal(got, ok)
	}
	_, err = s.Grant(got.MAC, DefaultGrant, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.LookupCode(sess.Code); ok {
		t.Fatal("pending should clear after grant")
	}
}

func TestPIN(t *testing.T) {
	c := Config{DeskPINHash: HashPIN("secret")}
	if !c.PINOK("secret") || c.PINOK("wrong") {
		t.Fatal("pin")
	}
}

func TestJoinQR(t *testing.T) {
	q := JoinQR("Shop", "pass;word", true)
	if q != `WIFI:T:WPA;S:Shop;P:pass\;word;H:true;;` {
		t.Fatal(q)
	}
}

func TestConfigRoundtrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "g.json")
	c := Config{Enabled: true, SSID: "G", PSK: "x", DeskPIN: "1234"}
	if err := SaveConfig(p, c); err != nil {
		t.Fatal(err)
	}
	c2, err := LoadConfig(p)
	if err != nil || c2.DeskPIN != "" || c2.DeskPINHash == "" || !c2.PINOK("1234") {
		t.Fatalf("%+v %v", c2, err)
	}
}
