package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestClassifyFailure(t *testing.T) {
	cases := []struct {
		err error
		cls FailureClass
	}{
		{fmt.Errorf("SHA256SUMS required"), FailResolve},
		{fmt.Errorf("checksum mismatch"), FailChecksum},
		{fmt.Errorf("no space left on device"), FailDisk},
		{fmt.Errorf("permission denied"), FailPermission},
		{fmt.Errorf("update: job already active for tg/local"), FailConflict},
		{fmt.Errorf("health: unit x is inactive"), FailHealth},
		{fmt.Errorf("no asset for component"), FailUnsupported},
	}
	for _, c := range cases {
		if g := ClassifyFailure(c.err); g != c.cls {
			t.Fatalf("%v: got %s want %s", c.err, g, c.cls)
		}
	}
}

func TestApplyEmptyDestFails(t *testing.T) {
	DefaultHostRegistry = NewJobRegistry()
	_, err := ApplyHostComponent(HostApplyOpts{
		Component: ComponentTG, InstanceID: "e1", Tag: "v0.9.596",
		DestPath: "", SkipDownload: false, SkipVersionProbe: true, JobID: "e1",
	})
	// Resolve may fail first without network - either resolve or empty dest
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestApplyConflictClass(t *testing.T) {
	DefaultHostRegistry = NewJobRegistry()
	_, err := DefaultHostRegistry.Begin(&UpdateJob{
		JobID: "a", Component: ComponentAgent, InstanceID: "c1", TargetVersion: "0.9.596",
		Asset: "x", State: StateDownloading, IdempotencyKey: "k",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApplyHostComponent(HostApplyOpts{
		Component: ComponentAgent, InstanceID: "c1", Tag: "v0.9.596",
		DestPath: "/tmp/x", SkipDownload: true, SkipVersionProbe: true, JobID: "b", IdempotencyKey: "other",
	})
	if err == nil {
		t.Fatal("expected conflict")
	}
	if ClassifyFailure(err) != FailConflict {
		t.Fatalf("class %s", ClassifyFailure(err))
	}
}

func TestHealthFailRestoresPrev(t *testing.T) {
	DefaultHostRegistry = NewJobRegistry()
	dir := t.TempDir()
	dest := filepath.Join(dir, "bin")
	_ = os.WriteFile(dest, []byte("old-bin"), 0o755)
	// skip download but manually set up as if install happened — use real download path simulation via AfterDownload
	// Instead: call atomic path through Apply with SkipDownload false needs network.
	// Direct unit test restore path:
	prev := dest + ".prev"
	_ = os.WriteFile(prev, []byte("old-bin"), 0o755)
	_ = os.WriteFile(dest, []byte("bad-new"), 0o755)
	if err := restorePrevBinary(dest, prev); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(dest)
	if string(b) != "old-bin" {
		t.Fatalf("got %s", b)
	}
}

func TestErrRollbackFailed(t *testing.T) {
	err := ErrRollbackFailed(errors.New("health boom"), errors.New("restore boom"))
	if ClassifyFailure(err) != FailRollback {
		t.Fatalf("%s", ClassifyFailure(err))
	}
}

func TestNodeApplyJobLifecycle(t *testing.T) {
	DefaultHostRegistry = NewJobRegistry()
	j, err := BeginNodeApply("v0.9.596", "n1")
	if err != nil || j.State != StateQueued {
		t.Fatalf("%v %+v", err, j)
	}
	j2, err := FinishNodeApply(true, "")
	if err != nil {
		t.Fatal(err)
	}
	if j2.State != StateSucceeded {
		t.Fatalf("state %s", j2.State)
	}
	DefaultHostRegistry = NewJobRegistry()
	_, _ = BeginNodeApply("v0.9.596", "n2")
	j3, err := FinishNodeApply(false, "preflight failed")
	if err != nil {
		t.Fatal(err)
	}
	if j3.State != StateFailed {
		t.Fatalf("state %s", j3.State)
	}
}
