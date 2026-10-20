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
	st2, tok2, isNew := Enroll(map[string]any{"device_id": "leak1", "board": "x"})
	if st2 != StatusApproved || isNew {
		t.Fatalf("status=%s isNew=%v", st2, isNew)
	}
	if tok2 != "" {
		t.Fatal("H-1: bootstrap re-enroll must not return existing device_token")
	}
	if !RequireApproved("Bearer "+dt, "leak1") {
		t.Fatal("original token still valid")
	}
}
