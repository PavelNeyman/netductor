package svcpaths

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateUnitExecStart(t *testing.T) {
	tests := []struct {
		name      string
		unit      string
		execStart string
		wantErr   bool
	}{
		{
			name:      "valid absolute command with arguments",
			unit:      "nd-wss-sp-server-nd-svc-sp1",
			execStart: "/usr/local/bin/wstunnel server wss://0.0.0.0:8454 --restrict-to 127.0.0.1:51840",
		},
		{name: "empty command", unit: "nd-wss", execStart: "  ", wantErr: true},
		{name: "relative executable", unit: "nd-wss", execStart: "wstunnel server", wantErr: true},
		{name: "newline injection", unit: "nd-wss", execStart: "/usr/bin/true\nExecStart=/bin/sh", wantErr: true},
		{name: "carriage return", unit: "nd-wss", execStart: "/usr/bin/true\rExecStart=/bin/sh", wantErr: true},
		{name: "nul byte", unit: "nd-wss", execStart: "/usr/bin/true\x00", wantErr: true},
		{name: "systemd specifier", unit: "nd-wss", execStart: "/usr/bin/echo %n", wantErr: true},
		{name: "unit path traversal", unit: "../nd-wss", execStart: "/usr/bin/true", wantErr: true},
		{name: "unit slash", unit: "nd/wss", execStart: "/usr/bin/true", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUnitExecStart(tt.unit, tt.execStart)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateUnitExecStart() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}


func TestBootstrapFileTransactionRollbackRestoresBeforeImages(t *testing.T) {
	root := t.TempDir()
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	oldRoot := bootstrapJournalRoot
	bootstrapJournalRoot = filepath.Join(root, "journal")
	defer func() { bootstrapJournalRoot = oldRoot }()

	existing := filepath.Join(root, "existing.conf")
	if err := os.WriteFile(existing, []byte("before"), 0o640); err != nil { t.Fatal(err) }
	created := filepath.Join(root, "created.conf")

	tx, err := beginBootstrapFileTransaction("secondary", "peer-b", 1)
	if err != nil { t.Fatal(err) }
	if err := tx.SnapshotFile(existing); err != nil { t.Fatal(err) }
	if err := tx.SnapshotFile(created); err != nil { t.Fatal(err) }
	if err := os.WriteFile(existing, []byte("after"), 0o600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(created, []byte("new"), 0o600); err != nil { t.Fatal(err) }

	if err := tx.Rollback(); err != nil { t.Fatal(err) }
	got, err := os.ReadFile(existing)
	if err != nil { t.Fatal(err) }
	if string(got) != "before" { t.Fatalf("existing content = %q, want before", got) }
	info, err := os.Stat(existing)
	if err != nil { t.Fatal(err) }
	if info.Mode().Perm() != 0o640 { t.Fatalf("existing mode = %o, want 640", info.Mode().Perm()) }
	if _, err := os.Lstat(created); !os.IsNotExist(err) { t.Fatalf("created file remains after rollback: %v", err) }
	entries, err := os.ReadDir(bootstrapJournalRoot)
	if err != nil { t.Fatal(err) }
	if len(entries) != 0 { t.Fatalf("rollback left journal entries: %v", entries) }
}

func TestRecoverBootstrapFileTransactionsIsIdempotent(t *testing.T) {
	for _, phase := range []string{"applying", "rolling_back"} {
		t.Run(phase, func(t *testing.T) {
			root := t.TempDir()
			oldAllowed := bootstrapAllowedFileRoots
			bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
			t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
			oldRoot := bootstrapJournalRoot
			bootstrapJournalRoot = filepath.Join(root, "journal")
			t.Cleanup(func() { bootstrapJournalRoot = oldRoot })

			target := filepath.Join(root, "peer.conf")
			if err := os.WriteFile(target, []byte("before"), 0o600); err != nil {
				t.Fatal(err)
			}
			tx, err := beginBootstrapFileTransaction("primary", "peer-b", 1)
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.SnapshotFile(target); err != nil {
				t.Fatal(err)
			}
			// Persist the exact durable phase at which the process is assumed to crash.
			tx.journal.Phase = phase
			if err := tx.persist(); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, []byte("after"), 0o600); err != nil {
				t.Fatal(err)
			}

			if err := recoverBootstrapFileTransactions(); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != "before" {
				t.Fatalf("recovered content = %q, want before", got)
			}
			if err := recoverBootstrapFileTransactions(); err != nil {
				t.Fatalf("second recovery must be idempotent: %v", err)
			}
		})
	}
}

func TestBootstrapFileTransactionRejectsSymlinkTarget(t *testing.T) {
	root := t.TempDir()
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	oldRoot := bootstrapJournalRoot
	bootstrapJournalRoot = filepath.Join(root, "journal")
	defer func() { bootstrapJournalRoot = oldRoot }()
	target := filepath.Join(root, "link")
	if err := os.Symlink(filepath.Join(root, "missing"), target); err != nil { t.Fatal(err) }
	tx, err := beginBootstrapFileTransaction("secondary", "peer-b", 1)
	if err != nil { t.Fatal(err) }
	if err := tx.SnapshotFile(target); err == nil { t.Fatal("expected symlink target rejection") }
	if err := tx.Rollback(); err != nil { t.Fatal(err) }
}

func TestBootstrapSnapshotRejectsOutsideManagedRoots(t *testing.T) {
	if err := validateBootstrapSnapshotPath("/etc/passwd"); err == nil { t.Fatal("expected unmanaged path rejection") }
}

func TestBootstrapSnapshotRejectsSymlinkParent(t *testing.T) {
	root := t.TempDir()
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	real := filepath.Join(root, "real")
	if err := os.Mkdir(real, 0o700); err != nil { t.Fatal(err) }
	link := filepath.Join(root, "link")
	if err := os.Symlink(real, link); err != nil { t.Fatal(err) }
	if err := validateBootstrapSnapshotPath(filepath.Join(link, "target.conf")); err == nil { t.Fatal("expected symlink parent rejection") }
}

func TestValidateBootstrapJournalRejectsMalformedMetadata(t *testing.T) {
	now := time.Now().UTC()
	valid := bootstrapJournal{
		Version: bootstrapJournalVersion,
		TxID: "0123456789abcdef0123456789abcdef",
		Role: "secondary",
		PeerID: "peer-b",
		Slot: 1,
		Phase: "applying",
		CreatedAt: now,
		UpdatedAt: now,
		Files: []bootstrapFileImage{{Path: "/etc/netductor/keys/material.json", Exists: true, Data: []byte("before"), Mode: 0o600, UID: 0, GID: 0}},
	}
	tests := []struct {
		name string
		mutate func(*bootstrapJournal)
	}{
		{name: "malformed transaction id", mutate: func(j *bootstrapJournal) { j.TxID = "not-hex" }},
		{name: "future creation time", mutate: func(j *bootstrapJournal) { j.CreatedAt = now.Add(time.Hour) }},
		{name: "updated before created", mutate: func(j *bootstrapJournal) { j.UpdatedAt = now.Add(-time.Second) }},
		{name: "negative uid", mutate: func(j *bootstrapJournal) { j.Files[0].UID = -1 }},
		{name: "negative gid", mutate: func(j *bootstrapJournal) { j.Files[0].GID = -1 }},
		{name: "oversized image", mutate: func(j *bootstrapJournal) { j.Files[0].Data = make([]byte, maxBootstrapImageBytes+1) }},
		{name: "invalid mode", mutate: func(j *bootstrapJournal) { j.Files[0].Mode = os.ModeSymlink | 0o600 }},
		{name: "unknown phase", mutate: func(j *bootstrapJournal) { j.Phase = "finished-ish" }},
		{name: "invalid peer identity", mutate: func(j *bootstrapJournal) { j.PeerID = "../peer" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := valid
			j.Files = append([]bootstrapFileImage(nil), valid.Files...)
			tt.mutate(&j)
			if err := validateBootstrapJournal(j, j.TxID+".json"); err == nil {
				t.Fatal("expected malformed journal rejection")
			}
		})
	}
	if err := validateBootstrapJournal(valid, valid.TxID+".json"); err != nil {
		t.Fatalf("valid journal rejected: %v", err)
	}
}


func TestEnsureBootstrapJournalDirectoryCreatesAndReusesPrivateRoot(t *testing.T) {
	parent := t.TempDir()
	journal := filepath.Join(parent, "journal")
	if err := ensureBootstrapJournalDirectory(journal); err != nil {
		t.Fatalf("create journal directory: %v", err)
	}
	if err := ensureBootstrapJournalDirectory(journal); err != nil {
		t.Fatalf("repeat ensure must be idempotent: %v", err)
	}
	info, err := os.Lstat(journal)
	if err != nil { t.Fatal(err) }
	if !info.IsDir() || info.Mode().Perm() != 0o700 {
		t.Fatalf("journal directory mode/type = %v, want private directory 0700", info.Mode())
	}
}

func TestEnsureBootstrapJournalDirectoryRejectsSymlinkParent(t *testing.T) {
	root := t.TempDir()
	realParent := filepath.Join(root, "real")
	if err := os.Mkdir(realParent, 0o700); err != nil { t.Fatal(err) }
	linkParent := filepath.Join(root, "parent-link")
	if err := os.Symlink(realParent, linkParent); err != nil { t.Fatal(err) }
	if err := ensureBootstrapJournalDirectory(filepath.Join(linkParent, "journal")); err == nil {
		t.Fatal("expected symlink parent rejection")
	}
	if _, err := os.Lstat(filepath.Join(realParent, "journal")); !os.IsNotExist(err) {
		t.Fatalf("journal was created through symlink parent: %v", err)
	}
}

func TestEnsureBootstrapJournalDirectoryRejectsSymlinkRoot(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil { t.Fatal(err) }
	link := filepath.Join(root, "journal-link")
	if err := os.Symlink(target, link); err != nil { t.Fatal(err) }
	if err := ensureBootstrapJournalDirectory(link); err == nil {
		t.Fatal("expected symlink journal root rejection")
	}
}

func TestEnsureBootstrapJournalDirectoryAllowsNonWritableSharedParent(t *testing.T) {
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o755); err != nil { t.Fatal(err) }
	if err := ensureBootstrapJournalDirectory(filepath.Join(parent, "journal")); err != nil {
		t.Fatalf("read/execute-only parent should be accepted: %v", err)
	}
}

func TestEnsureBootstrapJournalDirectoryRejectsWritableParent(t *testing.T) {
	parent := t.TempDir()
	if err := os.Chmod(parent, 0o770); err != nil { t.Fatal(err) }
	if err := ensureBootstrapJournalDirectory(filepath.Join(parent, "journal")); err == nil {
		t.Fatal("expected group-writable parent rejection")
	}
	if _, err := os.Lstat(filepath.Join(parent, "journal")); !os.IsNotExist(err) {
		t.Fatalf("journal was created under unsafe parent: %v", err)
	}
}


func TestValidateNoGlobalWSSAllows(t *testing.T) {
	tests := []struct {
		name string
		status string
		wantErr bool
	}{
		{name: "no broad rules", status: "Status: active\n8444/tcp ALLOW IN 10.0.0.1\n"},
		{name: "ipv4 anywhere", status: "Status: active\n8444/tcp ALLOW IN Anywhere\n", wantErr: true},
		{name: "ipv6 anywhere", status: "Status: active\n8444/tcp (v6) ALLOW IN Anywhere (v6)\n", wantErr: true},
		{name: "untagged administrator rule", status: "Status: active\n8454/tcp ALLOW IN Anywhere\n", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoGlobalWSSAllows(tt.status, []int{8444, 8454})
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateNoGlobalWSSAllows() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}


func TestValidateNoGlobalWSSAllowsProtectsAdministratorRules(t *testing.T) {
	for _, tc := range []struct {
		name string
		status string
		wantErr bool
	}{
		{name: "no broad rule", status: "Status: active\\nTo                         Action      From\\n8445/tcp                   ALLOW       203.0.113.10\\n"},
		{name: "untagged IPv4 administrator rule", status: "Status: active\\nTo                         Action      From\\n8445/tcp                   ALLOW       Anywhere\\n", wantErr: true},
		{name: "untagged IPv6 administrator rule", status: "Status: active\\nTo                         Action      From\\n8445/tcp (v6)              ALLOW       Anywhere (v6)\\n", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNoGlobalWSSAllows(tc.status, []int{8445})
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateNoGlobalWSSAllows() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
