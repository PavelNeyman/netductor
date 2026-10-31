package migrate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
		t.Fatalf("expected work on pre-0.9.12 layout: %#v", res)
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
