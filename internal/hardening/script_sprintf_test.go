package hardening

import (
	"strings"
	"testing"
)

func TestHardenScriptsDoNotSpliceFormat(t *testing.T) {
	prep := RemoteHardenPrepareScript()
	if !strings.Contains(prep, `printf '%s\n' "$ST"`) {
		t.Fatalf("prepare HEAD printf was eaten by Sprintf:\n%s", prep[strings.Index(prep, "HEAD="):strings.Index(prep, "HEAD=")+120])
	}
	if strings.Contains(prep, "Port 52222") && strings.Contains(prep[strings.Index(prep, "HEAD="):], "Port 52222") {
		// drop-in is appended later; HEAD line itself must not contain it
		head := prep[strings.Index(prep, "HEAD=") : strings.Index(prep, "HEAD=")+80]
		if strings.Contains(head, "Port ") {
			t.Fatalf("drop-in spliced into HEAD: %s", head)
		}
	}
	fin := RemoteHardenFinalizeScript()
	if strings.Contains(fin, "printf '") && !strings.Contains(fin, `printf '%s\n'`) {
		t.Fatalf("finalize printf malformed")
	}
	can := RemoteHardenCancelRollbackScript()
	if !strings.Contains(can, `printf '%s\n' "$ufw_status"`) {
		t.Fatalf("cancel printf was eaten by Sprintf")
	}
}
