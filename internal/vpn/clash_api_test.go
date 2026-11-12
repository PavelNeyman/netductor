package vpn

import (
	"os"
	"testing"
)

func TestClashAPIDefaultLoopback(t *testing.T) {
	_ = os.Unsetenv("NETDUCTOR_CLASH_API")
	if ClashAPIListenAddr() != DefaultClashAPIAddr {
		t.Fatal(ClashAPIListenAddr())
	}
	exp := ExperimentalClashAPI()
	if exp == nil {
		t.Fatal("expected experimental")
	}
	ca := exp["clash_api"].(map[string]any)
	if ca["external_controller"] != DefaultClashAPIAddr {
		t.Fatal(ca)
	}
}

func TestClashAPIOff(t *testing.T) {
	t.Setenv("NETDUCTOR_CLASH_API", "off")
	if ClashAPIListenAddr() != "" {
		t.Fatal("expected empty")
	}
	if ExperimentalClashAPI() != nil {
		t.Fatal("expected nil")
	}
}

func TestClashAPIRefusePublicBind(t *testing.T) {
	t.Setenv("NETDUCTOR_CLASH_API", "0.0.0.0:9090")
	a := ClashAPIListenAddr()
	if a != "127.0.0.1:9090" {
		t.Fatal(a)
	}
}

func TestAttachExperimental(t *testing.T) {
	_ = os.Unsetenv("NETDUCTOR_CLASH_API")
	cfg := map[string]any{"log": map[string]any{"level": "info"}}
	AttachExperimentalClashAPI(cfg)
	if cfg["experimental"] == nil {
		t.Fatal("missing experimental")
	}
}
