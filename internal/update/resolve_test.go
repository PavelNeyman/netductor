package update

import (
	"runtime"
	"testing"
)

func TestResolveReleaseAssetLocal(t *testing.T) {
	sums := map[string]string{
		"netductor-linux-amd64":        "aaa",
		"netductor-agent-linux-mipsle": "bbb",
	}
	r, err := ResolveReleaseAssetLocal(ComponentNode, Platform{GOOS: "linux", GOARCH: "amd64"}, "v0.9.596", sums)
	if err != nil || r.Name != "netductor-linux-amd64" || r.SHA256 != "aaa" {
		t.Fatalf("got %+v err=%v", r, err)
	}
	_, err = ResolveReleaseAssetLocal(ComponentNode, Platform{GOOS: "darwin", GOARCH: "amd64"}, "v0.9.596", sums)
	if err == nil {
		t.Fatal("expected reject darwin node")
	}
	_, err = ResolveReleaseAssetLocal(ComponentAgent, Platform{GOOS: "linux", GOARCH: "mipsle"}, "v0.9.596", sums)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ResolveReleaseAssetLocal(ComponentAgent, Platform{GOOS: "linux", GOARCH: "mipsle"}, "v0.9.596", map[string]string{})
	if err == nil {
		t.Fatal("expected missing sums")
	}
}

func TestJobRegistryIdempotencyAndConflict(t *testing.T) {
	r := NewJobRegistry()
	j1 := &UpdateJob{JobID: "1", Component: ComponentNode, InstanceID: "primary", TargetVersion: "0.9.596", Asset: "netductor-linux-amd64", IdempotencyKey: "k1", State: StateQueued}
	got, err := r.Begin(j1)
	if err != nil || got.JobID != "1" {
		t.Fatalf("begin1 %v %v", got, err)
	}
	j1b := &UpdateJob{JobID: "2", Component: ComponentNode, InstanceID: "primary", TargetVersion: "0.9.596", Asset: "netductor-linux-amd64", IdempotencyKey: "k1", State: StateQueued}
	got2, err := r.Begin(j1b)
	if err != nil || got2.JobID != "1" {
		t.Fatalf("idempotent want job 1, got %+v err=%v", got2, err)
	}
	j2 := &UpdateJob{JobID: "3", Component: ComponentNode, InstanceID: "primary", TargetVersion: "0.9.597", Asset: "netductor-linux-amd64", IdempotencyKey: "k2", State: StateQueued}
	_, err = r.Begin(j2)
	if err == nil {
		t.Fatal("expected conflict")
	}
	if _, err := r.Transition(ComponentNode, "primary", StateDownloading, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Transition(ComponentNode, "primary", StateSucceeded, ""); err == nil {
		t.Fatal("expected illegal jump to succeeded")
	}
}

func TestAssetNameLegacyCLIComponents(t *testing.T) {
	// node on this builder OS — may be empty if not linux; only assert linux contract
	if AssetName(ComponentNode, Platform{GOOS: "linux", GOARCH: "amd64"}) != "netductor-linux-amd64" {
		t.Fatal("node linux amd64")
	}
	if n := assetName("node"); runtime.GOOS == "linux" && runtime.GOARCH == "amd64" && n != "netductor-linux-amd64" {
		t.Fatalf("assetName node: %s", n)
	}
}
