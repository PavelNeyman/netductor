package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyHostComponentSkipDownloadHappy(t *testing.T) {
	DefaultHostRegistry = NewJobRegistry()
	dir := t.TempDir()
	dest := filepath.Join(dir, "bin")
	_ = os.WriteFile(dest, []byte("x"), 0o755)
	j, err := ApplyHostComponent(HostApplyOpts{
		Component:      ComponentTG,
		InstanceID:     "test-host",
		Tag:            "v0.9.596",
		DestPath:       dest,
		Restart:        false,
		SkipDownload:   true,
		IdempotencyKey: "t1",
		JobID:          "job-t1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if j.State != StateSucceeded {
		t.Fatalf("state %s", j.State)
	}
	// second concurrent different key should conflict while non-terminal — already succeeded so new begin ok
	DefaultHostRegistry = NewJobRegistry()
	j2, err := ApplyHostComponent(HostApplyOpts{
		Component: ComponentTG, InstanceID: "test-host", Tag: "v0.9.596",
		DestPath: dest, SkipDownload: true, JobID: "job-t2", IdempotencyKey: "t2",
	})
	if err != nil || j2.State != StateSucceeded {
		t.Fatalf("%v %+v", err, j2)
	}
}

func TestApplyHostConflict(t *testing.T) {
	DefaultHostRegistry = NewJobRegistry()
	_, err := DefaultHostRegistry.Begin(&UpdateJob{
		JobID: "a", Component: ComponentAgent, InstanceID: "x", TargetVersion: "0.9.596",
		Asset: "netductor-agent-linux-amd64", State: StateDownloading, IdempotencyKey: "k",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = ApplyHostComponent(HostApplyOpts{
		Component: ComponentAgent, InstanceID: "x", Tag: "v0.9.596",
		DestPath: "/tmp/nope", SkipDownload: true, JobID: "b", IdempotencyKey: "other",
	})
	if err == nil {
		t.Fatal("expected conflict")
	}
}
