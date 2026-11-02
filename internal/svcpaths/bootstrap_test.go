package svcpaths

import (
	"os"
	"path/filepath"
	"testing"
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
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	if err := validateBootstrapSnapshotPath(filepath.Join(link, "target.conf")); err == nil { t.Fatal("expected symlink parent rejection") }
}
