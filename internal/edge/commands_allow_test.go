package edge

import "testing"

func TestAllowedEdgeActionsArchitecture(t *testing.T) {
	// read/soft must be present
	for _, a := range []string{"ping", "status", "metrics", "uci_get", "apply_template", "luci_status"} {
		if !allowedEdgeActions[a] {
			t.Fatalf("missing allow %s", a)
		}
	}
	// destructive still allowed for operator session (single-op model) — document via presence
	for _, a := range []string{"uci_set", "sysupgrade", "reboot", "agent_update"} {
		if !allowedEdgeActions[a] {
			t.Fatalf("missing destructive allow %s", a)
		}
	}
	// unknown must not
	if allowedEdgeActions["rm_rf"] || allowedEdgeActions["shell"] {
		t.Fatal("dangerous action must not be allowed")
	}
	// MikroTik RSC is not an OpenWrt agent action
	if allowedEdgeActions["apply_rsc"] {
		t.Fatal("apply_rsc must not be queued to OpenWrt agent")
	}
}

func TestEnqueueCmdRejectsUnknown(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	t.Setenv("NETDUCTOR_EDGE_DIR", dir)
	Enroll(map[string]any{"device_id": "d1"})
	_, _ = Approve("d1")
	if id := EnqueueCmd("d1", "not_a_real_action", ""); id != "" {
		t.Fatalf("unknown action returned id %s", id)
	}
	if id := EnqueueCmd("d1", "ping", ""); id == "" {
		t.Fatal("ping should enqueue")
	}
}

func TestCommandLeaseUntilResult(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	t.Setenv("NETDUCTOR_EDGE_DIR", dir)
	Enroll(map[string]any{"device_id": "lease1"})
	_, _ = Approve("lease1")
	id := EnqueueCmd("lease1", "ping", "")
	if id == "" {
		t.Fatal("enqueue")
	}
	got := PollCommands("lease1")
	if len(got) != 1 || got[0]["id"] != id {
		t.Fatalf("poll1: %#v", got)
	}
	// Within lease: not re-delivered
	got2 := PollCommands("lease1")
	if len(got2) != 0 {
		t.Fatalf("within lease should be empty: %#v", got2)
	}
	CmdResult(map[string]any{"device_id": "lease1", "cmd_id": id, "result": "pong"})
	got3 := PollCommands("lease1")
	if len(got3) != 0 {
		t.Fatalf("after result: %#v", got3)
	}
}
