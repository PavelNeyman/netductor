package hardening

import (
	"strings"
	"testing"
)

func TestFinalizeParserIsMawkSafe(t *testing.T) {
	s := RemoteHardenFinalizeScript()
	if !strings.Contains(s, "ufw_first22()") {
		t.Fatal("missing ufw_first22 helper")
	}
	if strings.Contains(s, "[^]]") {
		t.Fatal("old mawk-hostile class still in finalize")
	}
	if !strings.Contains(s, "printf '%s\\n'") && !strings.Contains(s, `printf '%s\n'`) {
		t.Fatal("printf percent was eaten by Sprintf")
	}
	if !strings.Contains(s, "ufw status numbered -----") && !strings.Contains(s, "----- ufw status numbered -----") {
		t.Fatal("failure path does not dump ufw status")
	}
}
