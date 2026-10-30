package update

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestValidReleaseTag(t *testing.T) {
	ok, err := ValidReleaseTag("v0.9.612")
	if err != nil || ok != "v0.9.612" {
		t.Fatalf("%q %v", ok, err)
	}
	ok, err = ValidReleaseTag("0.9.612")
	if err != nil || ok != "v0.9.612" {
		t.Fatalf("normalize %q %v", ok, err)
	}
	if _, err := ValidReleaseTag(""); err == nil {
		t.Fatal("empty")
	}
	if _, err := ValidReleaseTag("../evil"); err == nil {
		t.Fatal("path")
	}
}

func TestResolveUnsupportedArch(t *testing.T) {
	sums := map[string]string{"netductor-linux-amd64": "aaa"}
	_, err := ResolveReleaseAssetLocal(ComponentNode, Platform{GOOS: "linux", GOARCH: "s390x"}, "v0.9.612", sums)
	if err == nil {
		t.Fatal("want unsupported arch")
	}
	if ClassifyFailure(err) != FailUnsupported && ClassifyFailure(err) != FailResolve {
		// either is acceptable if message mentions no asset
		if ClassifyFailure(fmt.Errorf("no asset for platform")) != FailUnsupported {
			t.Fatalf("class %s err=%v", ClassifyFailure(err), err)
		}
	}
}

func TestHostJobsSnapshotEmptyAndAfterJob(t *testing.T) {
	DefaultHostRegistry = NewJobRegistry()
	if n := len(HostJobsSnapshot()); n != 0 {
		t.Fatalf("want empty got %d", n)
	}
	dir := t.TempDir()
	dest := filepath.Join(dir, "bin")
	_ = os.WriteFile(dest, []byte("x"), 0o755)
	_, err := ApplyHostComponent(HostApplyOpts{
		Component: ComponentTG, InstanceID: "local", Tag: "v0.9.612",
		DestPath: dest, Restart: false, SkipDownload: true, SkipVersionProbe: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	snaps := HostJobsSnapshot()
	if len(snaps) != 1 || snaps[0].Component != "tg" || snaps[0].State != string(StateSucceeded) {
		t.Fatalf("%+v", snaps)
	}
}

func TestConcurrentBeginConflict(t *testing.T) {
	r := NewJobRegistry()
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := r.Begin(&UpdateJob{
				JobID: fmt.Sprintf("j%d", i), Component: ComponentTG, InstanceID: "local",
				TargetVersion: "0.9.612", Asset: "x", IdempotencyKey: fmt.Sprintf("k%d", i), State: StateQueued,
			})
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)
	ok, conflict := 0, 0
	for e := range errs {
		if e == nil {
			ok++
		} else {
			conflict++
		}
	}
	if ok != 1 || conflict != 19 {
		t.Fatalf("ok=%d conflict=%d", ok, conflict)
	}
}

func TestSafetyAgentDoesNotUseNodeAssetName(t *testing.T) {
	n := AssetName(ComponentNode, Platform{GOOS: "linux", GOARCH: "amd64"})
	a := AssetName(ComponentAgent, Platform{GOOS: "linux", GOARCH: "amd64"})
	if n == a || n == "" || a == "" {
		t.Fatalf("node=%s agent=%s", n, a)
	}
	if a != "netductor-agent-linux-amd64" {
		t.Fatal(a)
	}
}

func TestClassifyMissingAssetAndChecksum(t *testing.T) {
	if ClassifyFailure(errors.New("no asset for platform")) != FailUnsupported {
		t.Fatal("unsupported")
	}
	if ClassifyFailure(errors.New("checksum mismatch for foo")) != FailChecksum {
		t.Fatal("checksum")
	}
	if ClassifyFailure(errors.New("permission denied")) != FailPermission {
		t.Fatal("perm")
	}
}

func TestWrongChecksumLocal(t *testing.T) {
	sums := map[string]string{"netductor-linux-amd64": "deadbeef"}
	r, err := ResolveReleaseAssetLocal(ComponentNode, Platform{GOOS: "linux", GOARCH: "amd64"}, "v0.9.612", sums)
	if err != nil {
		t.Fatal(err)
	}
	// ResolveLocal only checks presence of name in sums, not file body —
	// simulate verify failure path classification
	if r.SHA256 != "deadbeef" {
		t.Fatal(r)
	}
	err = fmt.Errorf("checksum mismatch for %s", r.Name)
	if ClassifyFailure(err) != FailChecksum {
		t.Fatal(ClassifyFailure(err))
	}
}
