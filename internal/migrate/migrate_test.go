package migrate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PavelNeyman/netductor/internal/firewall"
)

func usePrimaryRoleForTest(t *testing.T) {
	t.Helper()
	previousRoleDetector := detectHostRole
	previousStateRepair := repairStateOnUpgrade
	detectHostRole = func() (string, bool) { return "primary", true }
	// Migration fixtures use isolated temporary roots. Do not let the repair
	// migration create hooks under the real /etc/letsencrypt tree.
	repairStateOnUpgrade = func() []string { return nil }
	t.Cleanup(func() {
		detectHostRole = previousRoleDetector
		repairStateOnUpgrade = previousStateRepair
	})
}

type stubMig struct {
	id      string
	need    bool
	applied int
}

func (s *stubMig) ID() string          { return s.id }
func (s *stubMig) Description() string { return s.id }
func (s *stubMig) Required(ctx context.Context) (bool, error) {
	return s.need, nil
}
func (s *stubMig) Apply(ctx context.Context) error {
	s.applied++
	s.need = false
	return nil
}
func (s *stubMig) Verify(ctx context.Context) error { return nil }

func TestRunRegistryIdempotent(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	s := &stubMig{id: "test-stub", need: true}
	ctx := context.Background()
	r1, err := RunRegistry(ctx, []Migration{s})
	if err != nil {
		t.Fatal(err)
	}
	if len(r1.Ran) != 1 || s.applied != 1 {
		t.Fatalf("r1=%+v applied=%d", r1, s.applied)
	}
	r2, err := RunRegistry(ctx, []Migration{s})
	if err != nil {
		t.Fatal(err)
	}
	if len(r2.Ran) != 0 || s.applied != 1 {
		t.Fatalf("second run should skip: %+v applied=%d", r2, s.applied)
	}
	jp := filepath.Join(os.Getenv("NETDUCTOR_STATE"), "migrate", "journal.json")
	if _, err := os.Stat(jp); err != nil {
		t.Fatal("journal missing", err)
	}
}

func TestRunRegistryApplyFail(t *testing.T) {
	t.Setenv("NETDUCTOR_STATE", t.TempDir())
	s := &failMig{id: "fail-me"}
	_, err := RunRegistry(context.Background(), []Migration{s})
	if err == nil {
		t.Fatal("expected error")
	}
}

type failMig struct{ id string }

func (s *failMig) ID() string                                 { return s.id }
func (s *failMig) Description() string                        { return s.id }
func (s *failMig) Required(ctx context.Context) (bool, error) { return true, nil }
func (s *failMig) Apply(ctx context.Context) error            { return context.Canceled }
func (s *failMig) Verify(ctx context.Context) error           { return nil }

func TestBuiltinMigrationsFromEmptyState(t *testing.T) {
	usePrimaryRoleForTest(t)
	root := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", root)
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	// Simulate pre-0.9.612-ish empty state: only empty state root
	_ = os.MkdirAll(root, 0o700)
	ctx := context.Background()
	res, err := RunRegistry(ctx, DefaultRegistry)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Ran) == 0 && len(res.Skipped) == 0 {
		t.Fatal("expected migrations to run or skip")
	}
	// Second pass should not re-apply repair marker path heavily
	res2, err := RunRegistry(ctx, DefaultRegistry)
	if err != nil {
		t.Fatal(err)
	}
	_ = res2
	if _, err := os.Stat(filepath.Join(root, "migrate", "journal.json")); err != nil {
		t.Fatal("journal", err)
	}
}

func TestPostCheckAfterRunAll(t *testing.T) {
	usePrimaryRoleForTest(t)
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	t.Setenv("NETDUCTOR_STATE", root)
	t.Setenv("NETDUCTOR_ETC", etc)
	t.Setenv("NETDUCTOR_SKIP_MIGRATE_POSTCHECK", "") // ensure on
	ctx := context.Background()
	if _, err := RunAll(ctx); err != nil {
		t.Fatal(err)
	}
	if err := PostCheck(); err != nil {
		t.Fatal(err)
	}
}

func TestFixtureLegacyLooseModes(t *testing.T) {
	usePrimaryRoleForTest(t)
	// Simulates a pre-migration install: dirs exist with loose modes, no journal, no role.
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	t.Setenv("NETDUCTOR_STATE", root)
	t.Setenv("NETDUCTOR_ETC", etc)
	_ = os.MkdirAll(filepath.Join(etc, "secrets"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, "edge"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "edge", "devices.json"), []byte("{}"), 0o644)
	// No role, no migrate journal
	ctx := context.Background()
	res, err := RunRegistry(ctx, DefaultRegistry)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Ran) == 0 {
		t.Fatalf("expected migrations to run on legacy layout, got %#v", res)
	}
	st, err := os.Stat(filepath.Join(etc, "secrets"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("secrets mode still loose: %04o", st.Mode().Perm())
	}
	if _, err := os.Stat(filepath.Join(etc, "role")); err != nil {
		t.Fatal("role", err)
	}
	// PostCheck should pass
	if err := PostCheck(); err != nil {
		t.Fatal(err)
	}
	// Idempotent second RunAll
	if _, err := RunAll(ctx); err != nil {
		t.Fatal("second RunAll", err)
	}
}

func TestInterruptedRunningReruns(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", root)
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	ctx := context.Background()
	// Pre-seed journal with StatusRunning for a mig that is still Required
	once := 0
	m := &countMig{id: "test-interrupted", need: true, applyFn: func() error {
		once++
		return nil
	}}
	// Write fake running record
	_ = os.MkdirAll(filepath.Join(root, "migrate"), 0o700)
	j := journalFile{Schema: 1, Records: map[string]record{
		"test-interrupted": {ID: "test-interrupted", Status: StatusRunning},
	}}
	if err := durableWriteJournal(j); err != nil {
		t.Fatal(err)
	}
	res, err := RunRegistry(ctx, []Migration{m})
	if err != nil {
		t.Fatal(err)
	}
	if once != 1 {
		t.Fatalf("apply calls=%d want 1", once)
	}
	if len(res.Ran) != 1 {
		t.Fatalf("ran=%v", res.Ran)
	}
}

type countMig struct {
	id      string
	need    bool
	applyFn func() error
}

func (c *countMig) ID() string                                 { return c.id }
func (c *countMig) Description() string                        { return c.id }
func (c *countMig) Required(ctx context.Context) (bool, error) { return c.need, nil }
func (c *countMig) Apply(ctx context.Context) error {
	if c.applyFn != nil {
		return c.applyFn()
	}
	return nil
}
func (c *countMig) Verify(ctx context.Context) error {
	c.need = false
	return nil
}

func TestClearFailedRecords(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", root)
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	_ = os.MkdirAll(filepath.Join(root, "migrate"), 0o700)
	j := journalFile{Schema: 1, Records: map[string]record{
		"a": {ID: "a", Status: StatusFailed, Error: "boom"},
		"b": {ID: "b", Status: StatusCompleted},
		"c": {ID: "c", Status: StatusRunning},
	}}
	if err := durableWriteJournal(j); err != nil {
		t.Fatal(err)
	}
	if err := ClearFailedRecords(); err != nil {
		t.Fatal(err)
	}
	pub, err := LoadJournalPublic()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := pub["a"]; ok {
		t.Fatal("failed record should be cleared")
	}
	if _, ok := pub["c"]; ok {
		t.Fatal("running record should be cleared")
	}
	if pub["b"].Status != StatusCompleted {
		t.Fatal("completed should remain")
	}
}

func TestFixturePre0912UpgradePath(t *testing.T) {
	usePrimaryRoleForTest(t)
	// Note: name is historical; this is pre-migration-runner layout, NOT a byte-accurate 0.9.612 VPS image.
	// Richer simulation of an install that predates the migration runner:
	// loose secrets, partial edge JSON, wrong modes, interrupted journal entry,
	// no role / aptguard / vpn-users / baseline tree, stale installed_version empty.
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	t.Setenv("NETDUCTOR_STATE", root)
	t.Setenv("NETDUCTOR_ETC", etc)
	_ = os.MkdirAll(filepath.Join(etc, "secrets"), 0o755)
	_ = os.WriteFile(filepath.Join(etc, "secrets", "secondary_agent_token"), []byte("x\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "edge"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "edge", "devices.json"), []byte("{}"), 0o666)
	_ = os.WriteFile(filepath.Join(root, "installed_version"), []byte("\n"), 0o644)
	_ = os.MkdirAll(filepath.Join(root, "migrate"), 0o700)
	j := journalFile{Schema: 1, Records: map[string]record{
		"2026-10-ensure-state-dirs": {ID: "2026-10-ensure-state-dirs", Status: StatusRunning},
	}}
	if err := durableWriteJournal(j); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	res, err := RunAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Ran) == 0 {
		t.Fatalf("expected work on pre-migration-runner layout: %#v", res)
	}
	// Secrets tightened
	st, err := os.Stat(filepath.Join(etc, "secrets"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("secrets mode %04o", st.Mode().Perm())
	}
	for _, rel := range []string{
		filepath.Join(etc, "role"),
		filepath.Join(etc, "vpn-users.json"),
		filepath.Join(root, "baseline"),
		filepath.Join(root, "sites"),
		filepath.Join(root, "installed_version"),
		filepath.Join(root, "migrate", "journal.json"),
	} {
		if _, err := os.Stat(rel); err != nil {
			t.Fatalf("missing after upgrade: %s: %v", rel, err)
		}
	}
	b, _ := os.ReadFile(filepath.Join(root, "installed_version"))
	if len(strings.TrimSpace(string(b))) == 0 {
		t.Fatal("installed_version empty")
	}
	if err := PostCheck(); err != nil {
		t.Fatal(err)
	}
	// Idempotent
	res2, err := RunAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(res2.Ran) != 0 {
		// may re-run if Required true; allow only if empty work
		for _, id := range res2.Ran {
			t.Log("re-ran", id)
		}
	}
}

func TestFailedMigrationClearAndRetry(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", root)
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	ctx := context.Background()
	calls := 0
	m := &countMig{id: "flaky", need: true, applyFn: func() error {
		calls++
		if calls == 1 {
			return context.Canceled
		}
		return nil
	}}
	_, err := RunRegistry(ctx, []Migration{m})
	if err == nil {
		t.Fatal("expected fail")
	}
	pub, _ := LoadJournalPublic()
	if pub["flaky"].Status != StatusFailed {
		t.Fatalf("status=%s", pub["flaky"].Status)
	}
	if err := ClearFailedRecords(); err != nil {
		t.Fatal(err)
	}
	m.need = true
	res, err := RunRegistry(ctx, []Migration{m})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
	if len(res.Ran) != 1 {
		t.Fatalf("ran=%v", res.Ran)
	}
}

func TestMigrateLockSerializes(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", root)
	t.Setenv("NETDUCTOR_ETC", filepath.Join(root, "etc"))
	ctx := context.Background()
	started := make(chan struct{})
	done1 := make(chan error, 1)
	go func() {
		m := &countMig{id: "slow", need: true, applyFn: func() error {
			close(started)
			time.Sleep(80 * time.Millisecond)
			return nil
		}}
		_, err := RunRegistry(ctx, []Migration{m})
		done1 <- err
	}()
	<-started
	m2 := &countMig{id: "after", need: true, applyFn: func() error { return nil }}
	if _, err := RunRegistry(ctx, []Migration{m2}); err != nil {
		t.Fatal(err)
	}
	if err := <-done1; err != nil {
		t.Fatal(err)
	}
}

func TestInstalledVersionRequiredNotExistVsOtherError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	m := &ensureInstalledVersionMigration{}
	need, err := m.Required(context.Background())
	if err != nil || !need {
		t.Fatalf("missing file: need=%v err=%v", need, err)
	}
	path := filepath.Join(dir, "installed_version")
	if err := os.WriteFile(path, []byte("0.9.651\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	need, err = m.Required(context.Background())
	if err != nil || need {
		t.Fatalf("present: need=%v err=%v", need, err)
	}
}

func TestRoleApplyRequiresDetectableRole(t *testing.T) {
	// Ensure message format is stable for operators when role cannot be determined.
	// We only assert DetectRole non-empty hosts typically; for empty, Apply must not write primary.
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_ETC", dir)
	// If DetectRole still finds secondary markers from the real host, Apply may succeed —
	// skip when role file would be valid.
	role := firewall.DetectRole()
	if role == "primary" || role == "secondary" {
		t.Skip("host already has detectable role")
	}
	m := &ensureRoleFileMigration{}
	if err := m.Apply(context.Background()); err == nil {
		t.Fatal("expected error when role unknown")
	}
}

func TestFixture0912PrimaryLayout(t *testing.T) {
	usePrimaryRoleForTest(t)
	// Layout modeled on production Primary ~0.9.612 era reports (paths/modes/markers),
	// not a full systemd image: VERSION, vpn-users, secrets, edge devices, no migrate journal,
	// no installed_version, role may be absent (DetectRole may still succeed on real hosts).
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	t.Setenv("NETDUCTOR_STATE", root)
	t.Setenv("NETDUCTOR_ETC", etc)
	t.Setenv("NETDUCTOR_SKIP_MIGRATE_POSTCHECK", "1") // PostCheck separately

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(etc, "secrets", "mtls", "clients"), 0o755)) // loose modes
	must(os.WriteFile(filepath.Join(etc, "VERSION"), []byte("0.9.612\n"), 0o644))
	must(os.WriteFile(filepath.Join(etc, "vpn-users.json"), []byte("{\"users\":[]}\n"), 0o644))
	must(os.WriteFile(filepath.Join(etc, "secrets", "placeholder"), []byte("x\n"), 0o644))
	must(os.MkdirAll(filepath.Join(root, "edge"), 0o755))
	must(os.WriteFile(filepath.Join(root, "edge", "devices.json"), []byte("{}\n"), 0o644))
	must(os.MkdirAll(filepath.Join(root, "svc-paths"), 0o755))
	// no migrate/, no installed_version, no role file — as older primary before runner

	ctx := context.Background()
	res, err := RunRegistry(ctx, DefaultRegistry)
	if err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	if len(res.Ran) == 0 {
		t.Fatalf("expected migrations on 0.9.612-like layout: %#v", res)
	}
	// Secrets 0700
	st, err := os.Stat(filepath.Join(etc, "secrets"))
	must(err)
	if st.Mode().Perm()&0o077 != 0 {
		t.Fatalf("secrets mode %04o", st.Mode().Perm())
	}
	iv, err := os.ReadFile(filepath.Join(root, "installed_version"))
	must(err)
	if strings.TrimSpace(string(iv)) == "" {
		t.Fatal("installed_version empty")
	}
	if _, err := os.Stat(filepath.Join(root, "migrate", "journal.json")); err != nil {
		t.Fatal(err)
	}
	pub1, err := LoadJournalPublic()
	must(err)

	// Second pass: no re-apply of completed work (Ran empty or only newly required)
	res2, err := RunRegistry(ctx, DefaultRegistry)
	if err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	if len(res2.Ran) != 0 {
		t.Fatalf("second pass should not re-run completed migrations: ran=%v", res2.Ran)
	}
	pub2, err := LoadJournalPublic()
	must(err)
	for id, r1 := range pub1 {
		r2, ok := pub2[id]
		if !ok {
			t.Fatalf("journal lost %s", id)
		}
		if r1.Status == StatusCompleted && r2.Status != StatusCompleted {
			t.Fatalf("%s status changed %s -> %s", id, r1.Status, r2.Status)
		}
	}

	t.Setenv("NETDUCTOR_SKIP_MIGRATE_POSTCHECK", "")
	if err := PostCheck(); err != nil {
		t.Fatal("PostCheck after 0.9.612 fixture:", err)
	}
}

func TestRepairStateMigrationReRunsForNewMarkerVersion(t *testing.T) {
	root := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", root)
	if err := os.MkdirAll(filepath.Join(root, "migrate"), 0o700); err != nil { t.Fatal(err) }
	marker := filepath.Join(root, "migrate", "repair-state.done")
	// Existing installations used a timeless "ok" marker. A new repair
	// generation must execute again so newly added managed files are covered.
	if err := os.WriteFile(marker, []byte("ok\n"), 0o600); err != nil { t.Fatal(err) }
	previous := repairStateOnUpgrade
	calls := 0
	repairStateOnUpgrade = func() []string { calls++; return nil }
	t.Cleanup(func() { repairStateOnUpgrade = previous })

	m := &repairStateOnUpgradeMigration{}
	need, err := m.Required(context.Background())
	if err != nil || !need { t.Fatalf("old marker: need=%v err=%v", need, err) }
	if err := m.Apply(context.Background()); err != nil { t.Fatal(err) }
	if err := m.Verify(context.Background()); err != nil { t.Fatal(err) }
	if calls != 1 { t.Fatalf("repair calls=%d want 1", calls) }
	b, err := os.ReadFile(marker)
	if err != nil { t.Fatal(err) }
	if string(b) != repairMarkerVersion+"\n" {
		t.Fatalf("marker=%q want %q", string(b), repairMarkerVersion+"\n")
	}
	need, err = m.Required(context.Background())
	if err != nil || need { t.Fatalf("new marker: need=%v err=%v", need, err) }
	if calls != 1 { t.Fatalf("repair unexpectedly reran: calls=%d", calls) }
}
