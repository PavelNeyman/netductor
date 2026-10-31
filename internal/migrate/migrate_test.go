package migrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
