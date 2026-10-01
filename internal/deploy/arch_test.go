package deploy

import "testing"

func TestMapUnameToGoArch(t *testing.T) {
	cases := map[string]string{
		"x86_64":      "amd64",
		"amd64":       "amd64",
		"aarch64":     "arm64",
		"arm64":       "arm64",
		"armv7l":      "arm",
		"armv7":       "arm",
		"arm":         "arm",
		"mips":        "mipsle",
		"mipsel":      "mipsle",
		"riscv64":     "riscv64",
		"  aarch64\n": "arm64",
	}
	for in, want := range cases {
		got, err := MapUnameToGoArch(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got != want {
			t.Fatalf("%q → %q want %q", in, got, want)
		}
	}
	if _, err := MapUnameToGoArch("powerpc"); err == nil {
		t.Fatal("expected error for powerpc")
	}
}

func TestMapOpenWrtArch(t *testing.T) {
	cases := map[string]string{
		"aarch64_cortex-a53": "arm64",
		"x86_64":             "amd64",
		"arm_cortex-a7_neon-vfpv4": "arm",
		"mipsel_24kc":        "mipsle",
	}
	for in, want := range cases {
		got, err := mapOpenWrtArch(in)
		if err != nil || got != want {
			t.Fatalf("%q → %q (%v) want %q", in, got, err, want)
		}
	}
}

func TestValidAgentArch(t *testing.T) {
	for _, s := range []string{"", "auto", "amd64", "arm64", "arm", "mipsle", "riscv64"} {
		if !ValidAgentArch(s) {
			t.Fatalf("%q should be valid", s)
		}
	}
	if ValidAgentArch("../x") {
		t.Fatal("path injection must fail")
	}
}
