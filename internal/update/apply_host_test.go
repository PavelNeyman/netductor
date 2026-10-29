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
		Component:        ComponentTG,
		InstanceID:       "test-host",
		Tag:              "v0.9.596",
		DestPath:         dest,
		Restart:          false,
		SkipDownload:     true,
		SkipVersionProbe: true,
		IdempotencyKey:   "t1",
		JobID:            "job-t1",
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
		DestPath: dest, SkipDownload: true, SkipVersionProbe: true, JobID: "job-t2", IdempotencyKey: "t2",
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
		DestPath: "/tmp/nope", SkipDownload: true, SkipVersionProbe: true, JobID: "b", IdempotencyKey: "other",
	})
	if err == nil {
		t.Fatal("expected conflict")
	}
}

func TestAtomicReplaceKeepsPrev(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "bin")
	_ = os.WriteFile(dest, []byte("old"), 0o755)
	newPath := dest + ".new"
	prevPath := dest + ".prev"
	_ = os.WriteFile(newPath, []byte("new"), 0o755)
	if err := atomicReplaceBinary(dest, newPath, prevPath); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(dest)
	if string(b) != "new" {
		t.Fatalf("dest=%s", b)
	}
	p, _ := os.ReadFile(prevPath)
	if string(p) != "old" {
		t.Fatalf("prev=%s", p)
	}
}

func TestProbeBinaryVersion(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake")
	script := "#!/bin/sh\necho netductor 0.9.596\n"
	_ = os.WriteFile(bin, []byte(script), 0o755)
	if err := probeBinaryVersion(bin, "0.9.596"); err != nil {
		t.Fatal(err)
	}
	if err := probeBinaryVersion(bin, "0.9.999"); err == nil {
		t.Fatal("expected mismatch")
	}
}
