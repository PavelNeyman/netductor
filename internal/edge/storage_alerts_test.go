package edge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestUSBPrevPersistSurvivesReload(t *testing.T) {
	dir := t.TempDir()
	usbPrevPath = filepath.Join(dir, "_usb_prev.json")
	resetUSBPrevForTest()

	usbPrevMu.Lock()
	loadUSBPrevLocked()
	usbPrev["r1"] = "missing"
	saveUSBPrevLocked()
	usbPrevMu.Unlock()

	resetUSBPrevForTest()
	usbPrevMu.Lock()
	loadUSBPrevLocked()
	got := usbPrev["r1"]
	usbPrevMu.Unlock()
	if got != "missing" {
		t.Fatalf("want missing after reload, got %q", got)
	}
	b, _ := os.ReadFile(usbPrevPath)
	var m map[string]string
	if err := json.Unmarshal(b, &m); err != nil || m["r1"] != "missing" {
		t.Fatalf("file: %s err=%v", b, err)
	}
}
