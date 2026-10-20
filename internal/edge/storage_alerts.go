package edge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/PavelNeyman/netductor/internal/notify"
)

func MuteUSBAlertsPath(deviceID string) string {
	return filepath.Join("/var/lib/netductor/edge-storage", deviceID, "mute_alerts")
}

func SetUSBAlertMute(deviceID string, mute bool) error {
	p := MuteUSBAlertsPath(deviceID)
	if mute {
		_ = os.MkdirAll(filepath.Dir(p), 0o755)
		return os.WriteFile(p, []byte("1\n"), 0o644)
	}
	_ = os.Remove(p)
	return nil
}

func USBAlertsMuted(deviceID string) bool {
	_, err := os.Stat(MuteUSBAlertsPath(deviceID))
	return err == nil
}

var usbPrevPath = "/var/lib/netductor/edge-storage/_usb_prev.json"

var (
	usbPrevMu   sync.Mutex
	usbPrev     map[string]string // deviceID -> "ok"|"missing"|"no-extroot"
	usbPrevLoad sync.Once
)

func loadUSBPrevLocked() {
	usbPrevLoad.Do(func() {
		usbPrev = map[string]string{}
		b, err := os.ReadFile(usbPrevPath)
		if err != nil {
			return
		}
		_ = json.Unmarshal(b, &usbPrev)
		if usbPrev == nil {
			usbPrev = map[string]string{}
		}
	})
}

func saveUSBPrevLocked() {
	_ = os.MkdirAll(filepath.Dir(usbPrevPath), 0o755)
	b, err := json.Marshal(usbPrev)
	if err != nil {
		return
	}
	_ = os.WriteFile(usbPrevPath, b, 0o600)
}

// EvaluateStorageAlerts from device facts after heartbeat.
func EvaluateStorageAlerts(deviceID string, st StorageFacts) {
	if deviceID == "" || USBAlertsMuted(deviceID) || !st.ExpectUSB {
		return
	}
	key := "edge-usb:" + deviceID
	state := "ok"
	if !st.HasUsableUSB() {
		state = "missing"
		notify.AlertOnce(key+":missing", "⚠️ Edge USB missing: "+deviceID+" (storage expected, no disk)")
	} else if !st.ExtrootActive {
		state = "no-extroot"
		notify.AlertOnce(key+":no-extroot", "⚠️ Edge USB present but extroot not active: "+deviceID)
	}

	usbPrevMu.Lock()
	loadUSBPrevLocked()
	prev := usbPrev[deviceID]
	if prev != state {
		usbPrev[deviceID] = state
		saveUSBPrevLocked()
	}
	usbPrevMu.Unlock()

	if state == "ok" && prev != "" && prev != "ok" {
		notify.AlertRefresh(key+":recovered", "✅ Edge USB restored: "+deviceID+" (storage OK)")
	}
}

func resetUSBPrevForTest() {
	usbPrevMu.Lock()
	defer usbPrevMu.Unlock()
	usbPrevLoad = sync.Once{}
	usbPrev = nil
}
