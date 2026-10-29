package update

import "testing"

func TestNormalizeEdgeGOARCH(t *testing.T) {
	ok := map[string]string{
		"x86_64": "amd64", "amd64": "amd64",
		"aarch64": "arm64", "arm64": "arm64",
		"armv7l": "arm", "arm": "arm",
		"mipsel": "mipsle", "mipsle": "mipsle",
		"riscv64": "riscv64",
	}
	for in, want := range ok {
		g, err := NormalizeEdgeGOARCH(in)
		if err != nil || g != want {
			t.Fatalf("%s: got %s %v", in, g, err)
		}
	}
	for _, bad := range []string{"", "auto", "mips", "armv6", "ppc64"} {
		if _, err := NormalizeEdgeGOARCH(bad); err == nil {
			t.Fatalf("expected reject %q", bad)
		}
	}
}

func TestAgentAssetNameNormalized(t *testing.T) {
	n, err := AgentAssetNameNormalized("mipsel_24kc")
	if err != nil || n != "netductor-agent-linux-mipsle" {
		t.Fatalf("%s %v", n, err)
	}
}

func TestEdgePathsStable(t *testing.T) {
	if EdgeBinaryPrimary != "/usr/sbin/netductor-agent" {
		t.Fatal("binary path changed without migration plan")
	}
	if EdgeConfigDir != "/etc/netductor-agent" {
		t.Fatal("config dir changed without migration plan")
	}
	if ComponentAgent != "agent" {
		t.Fatal("logical component id")
	}
}
