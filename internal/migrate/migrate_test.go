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
