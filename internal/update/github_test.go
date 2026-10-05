package update

import "testing"

func TestNewer(t *testing.T) {
	if !Newer("v0.8.0", "0.7.39-dev") {
		t.Fatal("0.8 > 0.7")
	}
	if Newer("0.8.0", "0.8.0") {
		t.Fatal("equal")
	}
	if Newer("0.7.9", "0.8.0") {
		t.Fatal("older")
	}
}

func TestParseSHA256SUMS(t *testing.T) {
	body := []byte("abc123  netductor-agent-linux-arm64\ndef456 *netductor-agent-linux-amd64\n# comment\n\n")
	m, err := ParseSHA256SUMS(body)
	if err != nil {
		t.Fatal(err)
	}
	if m["netductor-agent-linux-arm64"] != "abc123" {
		t.Fatalf("arm64: %v", m)
	}
	if m["netductor-agent-linux-amd64"] != "def456" {
		t.Fatalf("amd64: %v", m)
	}
	if _, err := ParseSHA256SUMS([]byte("# only\n")); err == nil {
		t.Fatal("expected empty error")
	}
}
