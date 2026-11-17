package cluster

import (
	"testing"

	"github.com/PavelNeyman/netductor/internal/controlplane"
)

func TestBuildSelfLeader(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("NETDUCTOR_STATE", dir)
	r := controlplane.RoleActive
	controlplane.SetTestRole(&r)
	defer controlplane.SetTestRole(nil)
	v := Build()
	if !v.WriteAllowed || len(v.Members) < 1 {
		t.Fatalf("%+v", v)
	}
	if v.Members[0].Kind != "control" || !v.Members[0].IsLeader {
		t.Fatalf("self %+v", v.Members[0])
	}
}
