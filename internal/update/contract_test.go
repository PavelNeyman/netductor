package update

import "testing"

func TestValidComponent(t *testing.T) {
	for _, id := range []ComponentID{ComponentNode, ComponentTG, ComponentAgent, ComponentOp} {
		if !ValidComponent(id) {
			t.Fatalf("expected valid %s", id)
		}
	}
	if ValidComponent("relay") || ValidComponent("") {
		t.Fatal("unexpected valid component")
	}
}

func TestAssetName(t *testing.T) {
	cases := []struct {
		comp ComponentID
		p    Platform
		want string
	}{
		{ComponentNode, Platform{GOOS: "linux", GOARCH: "amd64"}, "netductor-linux-amd64"},
		{ComponentNode, Platform{GOOS: "darwin", GOARCH: "amd64"}, ""},
		{ComponentTG, Platform{GOOS: "linux", GOARCH: "arm64"}, "netductor-tg-linux-arm64"},
		{ComponentAgent, Platform{GOOS: "linux", GOARCH: "mipsle"}, "netductor-agent-linux-mipsle"},
		{ComponentAgent, Platform{GOOS: "linux", GOARCH: "arm"}, "netductor-agent-linux-arm"},
		{ComponentOp, Platform{GOOS: "darwin", GOARCH: "arm64"}, "netductor-op-darwin-arm64"},
		{ComponentOp, Platform{GOOS: "linux", GOARCH: "amd64"}, ""},
	}
	for _, c := range cases {
		got := AssetName(c.comp, c.p)
		if got != c.want {
			t.Fatalf("%s %+v: got %q want %q", c.comp, c.p, got, c.want)
		}
	}
}

func TestHappyPathTransitions(t *testing.T) {
	path := []JobState{
		StateIdle, StateChecking, StateAvailable, StateQueued, StateDownloading,
		StateVerifying, StateInstalling, StateRestarting, StateHealthChecking, StateSucceeded, StateIdle,
	}
	for i := 0; i < len(path)-1; i++ {
		if err := ValidateTransition(path[i], path[i+1]); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
}

func TestIllegalTransitions(t *testing.T) {
	bad := [][2]JobState{
		{StateDownloading, StateSucceeded}, // skip verify/install
		{StateIdle, StateInstalling},
		{StateSucceeded, StateDownloading},
		{StateRolledBack, StateInstalling},
		{StateUnsupported, StateInstalling},
	}
	for _, b := range bad {
		if CanTransition(b[0], b[1]) {
			t.Fatalf("expected illegal %s → %s", b[0], b[1])
		}
	}
}

func TestRollbackPath(t *testing.T) {
	if err := ValidateTransition(StateInstalling, StateRollbackInProgress); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTransition(StateRollbackInProgress, StateRolledBack); err != nil {
		t.Fatal(err)
	}
	if err := ValidateTransition(StateRolledBack, StateChecking); err != nil {
		t.Fatal(err)
	}
}

func TestSelfTransitionOK(t *testing.T) {
	if !CanTransition(StateDownloading, StateDownloading) {
		t.Fatal("self transition should be allowed for progress heartbeats")
	}
}
