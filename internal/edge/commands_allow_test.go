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
