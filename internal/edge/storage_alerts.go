package edge

import (
	"os"
	"path/filepath"

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

func usbAlertsMuted(deviceID string) bool {
	_, err := os.Stat(MuteUSBAlertsPath(deviceID))
	return err == nil
}

func EvaluateStorageAlerts(deviceID string, st StorageFacts) {
	if deviceID == "" || usbAlertsMuted(deviceID) || !st.ExpectUSB {
		return
	}
	key := "edge-usb:" + deviceID
	if !st.HasUsableUSB() {
		notify.AlertOnce(key+":missing", "⚠️ Edge USB missing: "+deviceID+" (storage expected, no disk)")
		return
	}
	if !st.ExtrootActive {
		notify.AlertOnce(key+":no-extroot", "⚠️ Edge USB present but extroot not active: "+deviceID)
	}
}
