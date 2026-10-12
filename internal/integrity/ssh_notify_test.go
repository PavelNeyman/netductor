package integrity

import (
	"strings"
	"testing"
)

func TestSummarizeAuthorizedKeysForNotifyNoFullKey(t *testing.T) {
	blob := "AAAAC3NzaC1lZDI1NTE5AAAAIGxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxYY"
	body := "ssh-ed25519 " + blob + " netductor-primary\n"
	s := SummarizeAuthorizedKeysForNotify(body)
	if s == "" {
		t.Fatal("empty summary")
	}
	if strings.Contains(s, blob) {
		t.Fatalf("summary still contains full key material: %q", s)
	}
	if !strings.Contains(s, "netductor-primary") {
		t.Fatalf("expected comment in summary: %q", s)
	}
}
