package svcpaths

import (
	"fmt"
	"os"
	"errors"
	"strings"
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
		{name: "no broad rule", status: "Status: active\nTo                         Action      From\n8445/tcp                   ALLOW       203.0.113.10\n"},
		{name: "untagged IPv4 administrator rule", status: "Status: active\nTo                         Action      From\n8445/tcp                   ALLOW       Anywhere\n", wantErr: true},
		{name: "untagged IPv6 administrator rule", status: "Status: active\nTo                         Action      From\n8445/tcp (v6)              ALLOW       Anywhere (v6)\n", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateNoGlobalWSSAllows(tc.status, []int{8445})
			if (err != nil) != tc.wantErr {
				t.Fatalf("validateNoGlobalWSSAllows() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

 
// Transaction tests use injected actions and never execute host firewall commands.
func TestApplyFirewallMutationsRollsBackOnlyCreatedRules(t *testing.T) {
	var state []string
	mk := func(name string, fail bool) firewallMutation {
		return firewallMutation{
			apply: func() (bool, error) {
				if fail { return false, errors.New("injected failure") }
				state = append(state, name); return true, nil
			},
			rollback: func() error {
				for i := len(state)-1; i >= 0; i-- {
					if state[i] == name { state = append(state[:i], state[i+1:]...); return nil }
				}
				return nil
			},
		}
	}
	err := applyFirewallMutations([]firewallMutation{mk("peer-b-wss", false), mk("peer-b-svc", true)}, nil)
	if err == nil { t.Fatal("expected injected failure") }
	if len(state) != 0 { t.Fatalf("partial rule leaked after rollback: %v", state) }
}
func TestApplyFirewallMutationsPreservesExistingAndUnrelated(t *testing.T) {
	state := []string{"peer-a-rule", "peer-b-existing"}
	mutation := firewallMutation{apply: func() (bool,error) { return false,nil }, rollback: func() error { state=nil; return nil }}
	if err := applyFirewallMutations([]firewallMutation{mutation},nil); err != nil { t.Fatal(err) }
	if len(state)!=2 || state[0]!="peer-a-rule" || state[1]!="peer-b-existing" { t.Fatalf("existing rules changed: %v",state) }
}
func TestApplyFirewallMutationsRollbackAndRepersistOnPersistenceFailure(t *testing.T) {
	created := false
	persistCalls := 0
	mutation := firewallMutation{
		apply: func() (bool, error) { created = true; return true, nil },
		rollback: func() error { created = false; return nil },
	}
	err := applyFirewallMutations([]firewallMutation{mutation}, func() error {
		persistCalls++
		if persistCalls == 1 { return errors.New("injected initial save failure") }
		return nil
	})
	if err == nil { t.Fatal("expected initial persistence failure") }
	if created { t.Fatal("rule not rolled back") }
	if persistCalls != 2 { t.Fatalf("persist called %d times, want initial save plus restored-state save", persistCalls) }
}

func TestParseNetductorInputChainStrict(t *testing.T) {
	for _, tc := range []struct{name, input, want string; wantErr bool}{
		{name:"single A", input:"-P INPUT ACCEPT\n-A INPUT -j NETDUCTOR_A\n", want:"NETDUCTOR_A"},
		{name:"single B", input:"-A INPUT -j NETDUCTOR_B\n", want:"NETDUCTOR_B"},
		{name:"missing", input:"-P INPUT ACCEPT\n", wantErr:true},
		{name:"duplicate jump", input:"-A INPUT -j NETDUCTOR_A\n-A INPUT -j NETDUCTOR_A\n", wantErr:true},
		{name:"ambiguous chains", input:"-A INPUT -j NETDUCTOR_A\n-A INPUT -j NETDUCTOR_B\n", wantErr:true},
	} {
		t.Run(tc.name,func(t *testing.T){
			got,err:=parseNetductorInputChain(tc.input)
			if (err!=nil)!=tc.wantErr { t.Fatalf("error=%v wantErr=%v",err,tc.wantErr) }
			if err==nil && got!=tc.want { t.Fatalf("chain=%q want %q",got,tc.want) }
		})
	}
}

func TestUFWStatusHasPeerRuleMatchesExactNumberedRule(t *testing.T) {
	for _, tc := range []struct { name, status string; want bool }{
		{name: "numbered split bracket", status: "Status: active\n[ 2] 8444/tcp ALLOW IN 203.0.113.7 # nd-wss-sp\n", want: true},
		{name: "numbered compact bracket", status: "[2] 8444/tcp ALLOW IN 203.0.113.7 # nd-wss-sp\n", want: true},
		{name: "wrong source", status: "[ 2] 8444/tcp ALLOW IN 203.0.113.70 # nd-wss-sp\n"},
		{name: "wrong port", status: "[ 2] 18444/tcp ALLOW IN 203.0.113.7 # nd-wss-sp\n"},
		{name: "wrong comment", status: "[ 2] 8444/tcp ALLOW IN 203.0.113.7 # nd-wss-ps\n"},
		{name: "wrong action", status: "[ 2] 8444/tcp DENY IN 203.0.113.7 # nd-wss-sp\n"},
		{name: "wrong direction", status: "[ 2] 8444/tcp ALLOW OUT 203.0.113.7 # nd-wss-sp\n"},
		{name: "extra trailing fields", status: "[ 2] 8444/tcp ALLOW IN 203.0.113.7 # nd-wss-sp extra\n"},
		{name: "unnumbered status", status: "8444/tcp ALLOW IN 203.0.113.7 # nd-wss-sp\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ufwStatusHasPeerRule(tc.status, "203.0.113.7", 8444, "nd-wss-sp"); got != tc.want {
				t.Fatalf("ufwStatusHasPeerRule() = %v, want %v", got, tc.want)
			}
		})
	}
}


func TestBootstrapTransactionWritersSnapshotBeforeMutation(t *testing.T) {
	root := t.TempDir()
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	oldJournalRoot := bootstrapJournalRoot
	bootstrapJournalRoot = filepath.Join(root, "journal")
	t.Cleanup(func() { bootstrapJournalRoot = oldJournalRoot })

	existing := filepath.Join(root, "existing.conf")
	if err := os.WriteFile(existing, []byte("before"), 0o640); err != nil { t.Fatal(err) }
	created := filepath.Join(root, "created.conf")
	tx, err := beginBootstrapFileTransaction("secondary", "peer-b", 1)
	if err != nil { t.Fatal(err) }
	if err := tx.writeFile(existing, "after", 0o600); err != nil { t.Fatal(err) }
	if err := tx.writeFile(created, "new", 0o600); err != nil { t.Fatal(err) }
	if err := tx.Rollback(); err != nil { t.Fatal(err) }

	got, err := os.ReadFile(existing)
	if err != nil { t.Fatal(err) }
	if string(got) != "before" { t.Fatalf("existing content = %q, want before", got) }
	info, err := os.Stat(existing)
	if err != nil { t.Fatal(err) }
	if info.Mode().Perm() != 0o640 { t.Fatalf("existing mode = %o, want 640", info.Mode().Perm()) }
	if _, err := os.Lstat(created); !os.IsNotExist(err) { t.Fatalf("created file remains after rollback: %v", err) }
}

func TestBootstrapTransactionSnapshotsAndRestoresSystemdState(t *testing.T) {
	root := t.TempDir()
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	oldJournalRoot := bootstrapJournalRoot
	bootstrapJournalRoot = filepath.Join(root, "journal")
	t.Cleanup(func() { bootstrapJournalRoot = oldJournalRoot })
	oldSystemctl := bootstrapSystemctl
	defer func() { bootstrapSystemctl = oldSystemctl }()
	var calls []string
	bootstrapSystemctl = func(args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) > 0 && args[0] == "show" {
			return []byte("LoadState=loaded\nUnitFileState=enabled\nActiveState=active\n"), nil
		}
		return nil, nil
	}
	tx, err := beginBootstrapFileTransaction("secondary", "peer-b", 1)
	if err != nil { t.Fatal(err) }
	if err := tx.SnapshotUnits("nd-test.service", "nd-test.service"); err != nil { t.Fatal(err) }
	if err := tx.MarkUnitsTouched("nd-test.service"); err != nil { t.Fatal(err) }
	if len(tx.journal.Units) != 1 { t.Fatalf("unit snapshots = %d, want deduplicated 1", len(tx.journal.Units)) }
	if err := tx.Rollback(); err != nil { t.Fatal(err) }
	joined := strings.Join(calls, "\n")
	for _, want := range []string{"show --property=LoadState --property=UnitFileState --property=ActiveState nd-test.service", "daemon-reload", "enable nd-test.service", "restart nd-test.service"} {
		if !strings.Contains(joined, want) { t.Fatalf("systemctl calls missing %q:\n%s", want, joined) }
	}
}

func TestBootstrapTransactionRestoresPreviouslyAbsentUnit(t *testing.T) {
	root := t.TempDir()
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	oldJournalRoot := bootstrapJournalRoot
	bootstrapJournalRoot = filepath.Join(root, "journal")
	t.Cleanup(func() { bootstrapJournalRoot = oldJournalRoot })
	oldSystemctl := bootstrapSystemctl
	defer func() { bootstrapSystemctl = oldSystemctl }()
	var calls []string
	bootstrapSystemctl = func(args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) > 0 && args[0] == "show" {
			return []byte("LoadState=not-found\n"), nil
		}
		return nil, nil
	}
	tx, err := beginBootstrapFileTransaction("secondary", "peer-b", 1)
	if err != nil { t.Fatal(err) }
	if err := tx.SnapshotUnits("nd-new.service"); err != nil { t.Fatal(err) }
	if err := tx.MarkUnitsTouched("nd-new.service"); err != nil { t.Fatal(err) }
	if err := tx.Rollback(); err != nil { t.Fatal(err) }
	joined := strings.Join(calls, "\n")
	if !strings.Contains(joined, "stop nd-new.service") || !strings.Contains(joined, "disable nd-new.service") {
		t.Fatalf("previously absent unit was not returned to absent/inactive state:\n%s", joined)
	}
}

func TestValidateBootstrapJournalAcceptsLegacyV1(t *testing.T) {
	now := time.Now().UTC()
	j := bootstrapJournal{
		Version: 1, TxID: "0123456789abcdef0123456789abcdef",
		Role: "secondary", PeerID: "peer-b", Slot: 1, Phase: "applying",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := validateBootstrapJournal(j, j.TxID+".json"); err != nil {
		t.Fatalf("legacy version-1 journal must remain recoverable: %v", err)
	}
}

func TestBootstrapUnitSnapshotsAreAtomicOnPartialReadFailure(t *testing.T) {
	root := t.TempDir()
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	oldJournalRoot := bootstrapJournalRoot
	bootstrapJournalRoot = filepath.Join(root, "journal")
	t.Cleanup(func() { bootstrapJournalRoot = oldJournalRoot })
	oldSystemctl := bootstrapSystemctl
	defer func() { bootstrapSystemctl = oldSystemctl }()
	calls := 0
	bootstrapSystemctl = func(args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "show" {
			calls++
			if calls == 1 { return []byte("LoadState=loaded\nUnitFileState=enabled\nActiveState=active\n"), nil }
			return []byte("systemctl unavailable"), errors.New("injected failure")
		}
		return nil, nil
	}
	tx, err := beginBootstrapFileTransaction("secondary", "peer-b", 1)
	if err != nil { t.Fatal(err) }
	if err := tx.SnapshotUnits("nd-first.service", "nd-second.service"); err == nil {
		t.Fatal("expected snapshot failure")
	}
	if len(tx.journal.Units) != 0 {
		t.Fatalf("partial systemd snapshot persisted: %+v", tx.journal.Units)
	}
	if err := tx.Rollback(); err != nil { t.Fatal(err) }
}

func TestBootstrapRollbackDoesNotRestartUntouchedUnits(t *testing.T) {
	root := t.TempDir()
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	oldJournalRoot := bootstrapJournalRoot
	bootstrapJournalRoot = filepath.Join(root, "journal")
	t.Cleanup(func() { bootstrapJournalRoot = oldJournalRoot })
	oldSystemctl := bootstrapSystemctl
	defer func() { bootstrapSystemctl = oldSystemctl }()
	var calls []string
	bootstrapSystemctl = func(args ...string) ([]byte, error) {
		calls = append(calls, strings.Join(args, " "))
		if len(args) > 0 && args[0] == "show" {
			return []byte("LoadState=loaded\nUnitFileState=enabled\nActiveState=active\n"), nil
		}
		return nil, nil
	}
	tx, err := beginBootstrapFileTransaction("secondary", "peer-b", 1)
	if err != nil { t.Fatal(err) }
	if err := tx.SnapshotUnits("nd-untouched.service"); err != nil { t.Fatal(err) }
	if err := tx.Rollback(); err != nil { t.Fatal(err) }
	joined := strings.Join(calls, "\n")
	if strings.Contains(joined, "restart nd-untouched.service") || strings.Contains(joined, "enable nd-untouched.service") {
		t.Fatalf("rollback mutated an untouched unit:\n%s", joined)
	}
}

func TestBootstrapRollbackRemovesOnlyTransactionCreatedUFWRule(t *testing.T) {
	root := t.TempDir()
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	oldJournalRoot := bootstrapJournalRoot
	bootstrapJournalRoot = filepath.Join(root, "journal")
	t.Cleanup(func() { bootstrapJournalRoot = oldJournalRoot })
	oldLock, oldExec := bootstrapFirewallLock, bootstrapFirewallExec
	defer func() { bootstrapFirewallLock, bootstrapFirewallExec = oldLock, oldExec }()
	bootstrapFirewallLock = func() (func(), error) { return func(){}, nil }
	status := "Status: active\n[ 1] 8445/tcp ALLOW IN 203.0.113.20 # nd-wss-ps\n[ 2] 8444/tcp ALLOW IN 203.0.113.10 # nd-wss-sp\n"
	var calls []string
	bootstrapFirewallExec = func(name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name != "ufw" { return nil, fmt.Errorf("unexpected command %s", name) }
		if len(args) == 2 && args[0] == "status" && args[1] == "numbered" { return []byte(status), nil }
		if len(args) > 0 && args[0] == "--force" {
			if !strings.Contains(strings.Join(args, " "), "203.0.113.20") { return nil, fmt.Errorf("wrong rule deletion: %v", args) }
			status = "Status: active\n[ 1] 8444/tcp ALLOW IN 203.0.113.10 # nd-wss-sp\n"
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected ufw args: %v", args)
	}
	tx, err := beginBootstrapFileTransaction("secondary", "peer-b", 1)
	if err != nil { t.Fatal(err) }
	if err := tx.SnapshotFirewallRule(bootstrapFirewallImage{Backend:"ufw", Source:"203.0.113.20", Port:8445, Comment:"nd-wss-ps"}, false); err != nil { t.Fatal(err) }
	if err := tx.Rollback(); err != nil { t.Fatal(err) }
	if strings.Contains(status, "203.0.113.20") { t.Fatalf("transaction-created rule remains: %s", status) }
	if !strings.Contains(status, "203.0.113.10") { t.Fatalf("unrelated existing peer rule was removed: %s", status) }
	if len(calls) != 3 { t.Fatalf("firewall calls = %v, want inspect/delete/verify", calls) }
}

func TestBootstrapRollbackPreservesPreexistingFirewallRule(t *testing.T) {
	root := t.TempDir()
	oldAllowed := bootstrapAllowedFileRoots
	bootstrapAllowedFileRoots = append(append([]string(nil), oldAllowed...), root)
	t.Cleanup(func() { bootstrapAllowedFileRoots = oldAllowed })
	oldJournalRoot := bootstrapJournalRoot
	bootstrapJournalRoot = filepath.Join(root, "journal")
	t.Cleanup(func() { bootstrapJournalRoot = oldJournalRoot })
	oldLock, oldExec := bootstrapFirewallLock, bootstrapFirewallExec
	defer func() { bootstrapFirewallLock, bootstrapFirewallExec = oldLock, oldExec }()
	bootstrapFirewallLock = func() (func(), error) { return func(){}, nil }
	calls := 0
	bootstrapFirewallExec = func(name string, args ...string) ([]byte, error) { calls++; return nil, nil }
	tx, err := beginBootstrapFileTransaction("secondary", "peer-b", 1)
	if err != nil { t.Fatal(err) }
	if err := tx.SnapshotFirewallRule(bootstrapFirewallImage{Backend:"ufw", Source:"203.0.113.20", Port:8445, Comment:"nd-wss-ps"}, true); err != nil { t.Fatal(err) }
	if err := tx.Rollback(); err != nil { t.Fatal(err) }
	if calls != 0 { t.Fatalf("pre-existing firewall rule caused %d rollback commands", calls) }
}
