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
	if _, err := MapUnameToGoArch("armv6l"); err == nil {
		t.Fatal("armv6 must not map to GOARM=7 arm")
	}
	if _, err := MapUnameToGoArch("mips"); err == nil {
		t.Fatal("bare mips must not guess endian")
	}
}

func TestMapOpenWrtArch(t *testing.T) {
	cases := map[string]string{
		"aarch64_cortex-a53":       "arm64",
		"x86_64":                   "amd64",
		"arm_cortex-a7_neon-vfpv4": "arm",
		"mipsel_24kc":              "mipsle",
	}
	if _, err := mapOpenWrtArch("mips_24kc"); err == nil {
		t.Fatal("big-endian mips_24kc must not become mipsle")
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
