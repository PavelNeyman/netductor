package svcpaths

import "testing"

func TestTickBuildPathsFailoverEmptyWithoutMaterial(t *testing.T) {
	// CI/sandbox: no /etc/netductor/svc-paths/build-material.json → no alert noise.
	if BuildPathsExpected() {
		t.Skip("material present on this host")
	}
	if sum := TickBuildPathsFailover(); sum != "" {
		t.Fatalf("want empty summary without material, got %q", sum)
	}
}
