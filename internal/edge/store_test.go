package edge

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func setup(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	os.Setenv("NETDUCTOR_EDGE_DIR", dir)
	os.Setenv("NETDUCTOR_ETC", dir)
	_ = os.MkdirAll(filepath.Join(dir, "secrets"), 0o700)
	_ = os.WriteFile(filepath.Join(dir, "secrets", "edge_bootstrap_token"), []byte("boot-token-at-least-32-characters-xx\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "secrets", "edge_token"), []byte("global\n"), 0o600)
	t.Cleanup(func() {
		os.Unsetenv("NETDUCTOR_EDGE_DIR")
		os.Unsetenv("NETDUCTOR_ETC")
	})
}

func TestEnrollApproveFlow(t *testing.T) {
	setup(t)
	st, tok, isNew := Enroll(map[string]any{"device_id": "site1", "board": "Cudy", "wan_ip": "1.2.3.4"})
	if st != StatusPending || tok != "" || !isNew {
		t.Fatalf("%s %q %v", st, tok, isNew)
	}
	if len(ListPending()) != 1 {
		t.Fatal(ListPending())
	}
	if RequireApproved("Bearer nope", "site1") {
		t.Fatal("should fail")
	}
	dt, err := Approve("site1")
	if err != nil || dt == "" {
		t.Fatal(err, dt)
	}
	if !RequireApproved("Bearer "+dt, "site1") {
		t.Fatal("device token")
	}
	if !ValidBearer("Bearer " + dt) {
		t.Fatal("valid bearer")
	}
	// commands only approved
	id := EnqueueCmd("site1", "ping", "")
	if id == "" {
		t.Fatal("enqueue")
	}
	cmds := PollCommands("site1")
	if len(cmds) != 1 {
		t.Fatal(cmds)
	}
	Deny("site2") // unknown err
	_ = Deny("site1")
	if RequireApproved("Bearer "+dt, "site1") {
		t.Fatal("denied")
	}
}

func TestRevoke(t *testing.T) {
	setup(t)
	Enroll(map[string]any{"device_id": "r1"})
	tok, _ := Approve("r1")
	_ = Revoke("r1")
	if RequireApproved("Bearer "+tok, "r1") {
		t.Fatal()
	}
}

func TestBackupPathTraversal(t *testing.T) {
	setup(t)
	_, err := SaveBackup("s", bytes.NewReader([]byte("x")))
	if err != nil {
		t.Fatal(err)
	}
	list := ListBackups("s")
	if len(list) != 1 {
		t.Fatal(list)
	}
	_, err = BackupPath("s", "../x")
	if err == nil {
		t.Fatal()
	}
}

func TestBootstrap(t *testing.T) {
	setup(t)
	if !ValidBootstrap("Bearer boot-token-at-least-32-characters-xx") {
		t.Fatal()
	}
	if ValidBootstrap("Bearer wrong") {
		t.Fatal()
	}
}

func TestEnrollApprovedNoTokenLeak(t *testing.T) {
	setup(t)
	st, tok, _ := Enroll(map[string]any{"device_id": "leak1", "board": "x"})
	if st != StatusPending || tok != "" {
		t.Fatalf("pending: %s %q", st, tok)
	}
	dt, err := Approve("leak1")
	if err != nil || dt == "" {
		t.Fatal(err, dt)
	}
	// Enroll itself never returns personal token (R-05).
	st2, tok2, isNew := Enroll(map[string]any{"device_id": "leak1", "board": "x"})
	if st2 != StatusApproved || isNew || tok2 != "" {
		t.Fatalf("enroll must not return token: status=%s tok=%q isNew=%v", st2, tok2, isNew)
	}
	// PeekPickupToken is armed until Confirm (recovery/operator path only).
	if pt := PeekPickupToken("leak1"); pt != dt {
		t.Fatalf("peek pickup: %q", pt)
	}
	if err := ConfirmTokenPickup("leak1", dt); err != nil {
		t.Fatal(err)
	}
	if pt := PeekPickupToken("leak1"); pt != "" {
		t.Fatalf("after ack peek must be empty: %q", pt)
	}
	st3, tok3, _ := Enroll(map[string]any{"device_id": "leak1", "board": "x"})
	if st3 != StatusApproved || tok3 != "" {
		t.Fatalf("re-enroll must not return token: status=%s tok=%q", st3, tok3)
	}
	if !RequireApproved("Bearer "+dt, "leak1") {
		t.Fatal("original token still valid")
	}
}

func TestEnsureBootstrapToken(t *testing.T) {
	dir := t.TempDir()
	os.Setenv("NETDUCTOR_ETC", dir)
	t.Cleanup(func() { os.Unsetenv("NETDUCTOR_ETC") })
	tok, err := EnsureBootstrapToken()
	if err != nil || len(tok) < 32 {
		t.Fatal(err, len(tok))
	}
	tok2, err := EnsureBootstrapToken()
	if err != nil || tok2 != tok {
		t.Fatal("must not rotate existing", err, tok2)
	}
}

func TestReissueTokenPickup(t *testing.T) {
	setup(t)
	Enroll(map[string]any{"device_id": "ri1"})
	dt, _ := Approve("ri1")
	_ = ConfirmTokenPickup("ri1", dt)
	st, tok, _ := Enroll(map[string]any{"device_id": "ri1"})
	if st != StatusApproved || tok != "" {
		t.Fatalf("after ack: %s %q", st, tok)
	}
	dt2, err := ReissueToken("ri1")
	if err != nil || dt2 != dt {
		t.Fatal(err, dt2)
	}
	st2, tok2, _ := Enroll(map[string]any{"device_id": "ri1"})
	if st2 != StatusApproved || tok2 != "" {
		t.Fatalf("enroll after reissue must not return token: %s %q", st2, tok2)
	}
	if pt := PeekPickupToken("ri1"); pt != dt {
		t.Fatalf("peek after reissue: %q", pt)
	}
}

func TestPeekRecoveryDoesNotConsume(t *testing.T) {
	setup(t)
	code, _, err := IssueRecoveryCode("site-a", "n", 0)
	if err != nil {
		t.Fatal(err)
	}
	sid, ok := PeekRecoveryCode(code)
	if !ok || sid != "site-a" {
		t.Fatal(sid, ok)
	}
	sid2, ok2 := PeekRecoveryCode(code)
	if !ok2 || sid2 != "site-a" {
		t.Fatal("peek must not consume")
	}
	_, ok3 := ConsumeRecoveryCode(code)
	if !ok3 {
		t.Fatal("consume")
	}
	if _, ok4 := PeekRecoveryCode(code); ok4 {
		t.Fatal("after consume peek must fail")
	}
}

func TestRotateDeviceToken(t *testing.T) {
	setup(t)
	Enroll(map[string]any{"device_id": "rot1"})
	old, _ := Approve("rot1")
	_ = ConfirmTokenPickup("rot1", old)
	neu, err := RotateDeviceToken("rot1")
	if err != nil || neu == "" || neu == old {
		t.Fatal(err, neu, old)
	}
	if RequireApproved("Bearer "+old, "rot1") {
		t.Fatal("old token must not work")
	}
	st, tok, _ := Enroll(map[string]any{"device_id": "rot1"})
	if st != StatusApproved || tok != "" {
		t.Fatalf("enroll after rotate must not return token: %s %q", st, tok)
	}
	if pt := PeekPickupToken("rot1"); pt != neu {
		t.Fatalf("peek after rotate: %q want %q", pt, neu)
	}
}
