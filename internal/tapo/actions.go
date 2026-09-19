package tapo

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// GetBasicInfo returns device basic_info (pytapo getBasicInfo).
func (c *Client) GetBasicInfo() (map[string]any, error) {
	return c.Execute("getDeviceInfo", map[string]any{
		"device_info": map[string]any{"name": []string{"basic_info"}},
	})
}

// MoveMotorStep relative angle 0..359 (pytapo relativeMove).
func (c *Client) MoveMotorStep(angle int) (map[string]any, error) {
	if angle < 0 || angle >= 360 {
		return nil, fmt.Errorf("angle must be 0 <= angle < 360")
	}
	return c.Execute("relativeMove", map[string]any{
		"motor": map[string]any{
			"movestep": map[string]any{"direction": strconv.Itoa(angle)},
		},
	})
}

// CalibrateMotor runs manualCalibrate.
func (c *Client) CalibrateMotor() (map[string]any, error) {
	return c.Execute("manualCalibrate", map[string]any{
		"motor": map[string]any{"manual_cali": ""},
	})
}

// CruiseStop stops cruise/patrol.
func (c *Client) CruiseStop() (map[string]any, error) {
	return c.Execute("cruiseStop", map[string]any{
		"motor": map[string]any{"cruise_stop": map[string]any{}},
	})
}

// GetPresets raw getPresetConfig.
func (c *Client) GetPresets() (map[string]any, error) {
	return c.Execute("getPresetConfig", map[string]any{
		"preset": map[string]any{"name": []string{"preset"}},
	})
}

// SavePreset adds current PTZ position (typo in API: addMotorPostion).
func (c *Client) SavePreset(name string) (map[string]any, error) {
	return c.Execute("addMotorPostion", map[string]any{
		"preset": map[string]any{
			"set_preset": map[string]any{"name": name, "save_ptz": "1"},
		},
	})
}

// GotoPreset moves to preset id.
func (c *Client) GotoPreset(id string) (map[string]any, error) {
	return c.Execute("motorMoveToPreset", map[string]any{
		"preset": map[string]any{"goto_preset": map[string]any{"id": id}},
	})
}

// DeletePreset removes preset id.
func (c *Client) DeletePreset(id string) (map[string]any, error) {
	return c.Execute("deletePreset", map[string]any{
		"preset": map[string]any{"remove_preset": map[string]any{"id": []string{id}}},
	})
}

// SetLED enables status LED.
func (c *Client) SetLED(on bool) (map[string]any, error) {
	en := "off"
	if on {
		en = "on"
	}
	return c.Execute("setLedStatus", map[string]any{
		"led": map[string]any{"config": map[string]any{"enabled": en}},
	})
}

// GetAlarmConfig motion/siren config.
func (c *Client) GetAlarmConfig() (map[string]any, error) {
	return c.Execute("getAlarmConfig", map[string]any{"msg_alarm": map[string]any{}})
}

// Reboot device.
func (c *Client) Reboot() (map[string]any, error) {
	return c.Execute("rebootDevice", map[string]any{"system": map[string]any{"reboot": "null"}})
}

func resultJSON(res map[string]any, err error) string {
	if err != nil {
		return "tapo-go:err:" + err.Error()
	}
	b, _ := json.Marshal(res)
	return "tapo-go:" + string(b)
}

// Control one-shot helper (extended).
// dir:
//
//	left|right|up|down|stop
//	step:<angle>
//	calibrate
//	cruise_stop
//	night:on|off|auto
//	privacy:on|off
//	led:on|off
//	info
//	presets
//	preset_save:<name>
//	preset_goto:<id>
//	preset_del:<id>
func Control(host, user, password, dir string, step int) string {
	if step <= 0 {
		step = 10
	}
	cl := New(host, user, password)
	if err := cl.Login(); err != nil {
		return "tapo-go:login:" + err.Error()
	}
	switch {
	case dir == "info":
		return resultJSON(cl.GetBasicInfo())
	case dir == "alarm":
		return resultJSON(cl.GetAlarmConfig())
	case dir == "reboot":
		return resultJSON(cl.Reboot())
	case dir == "presets":
		return resultJSON(cl.GetPresets())
	case dir == "privacy_get":
		return resultJSON(cl.GetPrivacy())
	case dir == "motion_get":
		return resultJSON(cl.GetMotionDetection())
	case strings.HasPrefix(dir, "motion:"):
		// motion:on|off or motion:on:high
		parts := strings.Split(strings.TrimPrefix(dir, "motion:"), ":")
		on := parts[0] == "on" || parts[0] == "1"
		sens := ""
		if len(parts) > 1 {
			sens = parts[1]
		}
		return resultJSON(cl.SetMotionDetection(on, sens))
	case strings.HasPrefix(dir, "alarm:"):
		on := strings.TrimPrefix(dir, "alarm:") == "on"
		return resultJSON(cl.SetAlarm(on, true, true))
	case dir == "children":
		return resultJSON(cl.GetChildDeviceList())
	case dir == "smart_track":
		return resultJSON(cl.GetSmartTrack())
	case dir == "calibrate":
		return resultJSON(cl.CalibrateMotor())
	case dir == "cruise_stop":
		return resultJSON(cl.CruiseStop())
	case strings.HasPrefix(dir, "step:"):
		ang, _ := strconv.Atoi(strings.TrimPrefix(dir, "step:"))
		return resultJSON(cl.MoveMotorStep(ang))
	case strings.HasPrefix(dir, "preset_save:"):
		return resultJSON(cl.SavePreset(strings.TrimPrefix(dir, "preset_save:")))
	case strings.HasPrefix(dir, "preset_goto:"):
		return resultJSON(cl.GotoPreset(strings.TrimPrefix(dir, "preset_goto:")))
	case strings.HasPrefix(dir, "preset_del:"):
		return resultJSON(cl.DeletePreset(strings.TrimPrefix(dir, "preset_del:")))
	case strings.HasPrefix(dir, "night:"):
		return resultJSON(cl.SetDayNight(strings.TrimPrefix(dir, "night:")))
	case strings.HasPrefix(dir, "privacy:"):
		on := strings.TrimPrefix(dir, "privacy:")
		return resultJSON(cl.SetPrivacy(on == "on" || on == "1" || on == "true"))
	case strings.HasPrefix(dir, "led:"):
		on := strings.TrimPrefix(dir, "led:")
		return resultJSON(cl.SetLED(on == "on" || on == "1" || on == "true"))
	case dir == "left":
		return resultJSON(cl.MoveMotor(-step, 0))
	case dir == "right":
		return resultJSON(cl.MoveMotor(step, 0))
	case dir == "up":
		return resultJSON(cl.MoveMotor(0, step))
	case dir == "down":
		return resultJSON(cl.MoveMotor(0, -step))
	case dir == "stop":
		_, _ = cl.CruiseStop()
		return "tapo-go:stop:ok"
	default:
		return "tapo-go:error:dir"
	}
}

// GetPrivacy lens mask status.
func (c *Client) GetPrivacy() (map[string]any, error) {
	return c.Execute("getLensMaskConfig", map[string]any{
		"lens_mask": map[string]any{"name": []string{"lens_mask_info"}},
	})
}

// GetMotionDetection config.
func (c *Client) GetMotionDetection() (map[string]any, error) {
	return c.Execute("getDetectionConfig", map[string]any{
		"motion_detection": map[string]any{"name": []string{"motion_det"}},
	})
}

// SetMotionDetection enabled on/off; sensitivity "" | "low"|"medium"|"high" maps digital_sensitivity.
func (c *Client) SetMotionDetection(enabled bool, sensitivity string) (map[string]any, error) {
	det := map[string]any{"enabled": "off"}
	if enabled {
		det["enabled"] = "on"
	}
	switch strings.ToLower(sensitivity) {
	case "low", "l":
		det["digital_sensitivity"] = "20"
		det["sensitivity"] = "low"
	case "medium", "med", "m":
		det["digital_sensitivity"] = "50"
		det["sensitivity"] = "normal"
	case "high", "h":
		det["digital_sensitivity"] = "80"
		det["sensitivity"] = "high"
	}
	return c.Execute("setDetectionConfig", map[string]any{
		"motion_detection": map[string]any{"motion_det": det},
	})
}

// SetAlarm enables siren/light alarm (C200-style).
func (c *Client) SetAlarm(enabled, sound, light bool) (map[string]any, error) {
	modes := []string{}
	if sound {
		modes = append(modes, "sound")
	}
	if light {
		modes = append(modes, "light")
	}
	if len(modes) == 0 {
		modes = []string{"sound"}
	}
	en := "off"
	if enabled {
		en = "on"
	}
	return c.Perform(map[string]any{
		"method": "set",
		"msg_alarm": map[string]any{
			"chn1_msg_alarm_info": map[string]any{
				"alarm_type": "0",
				"enabled":    en,
				"light_type": "0",
				"alarm_mode": modes,
			},
		},
	})
}

// GetChildDeviceList for hubs.
func (c *Client) GetChildDeviceList() (map[string]any, error) {
	return c.Execute("getChildDeviceList", map[string]any{
		"childControl": map[string]any{"start_index": 0},
	})
}

// GetSmartTrack config.
func (c *Client) GetSmartTrack() (map[string]any, error) {
	return c.Execute("getSmartTrackConfig", map[string]any{
		"smart_track": map[string]any{"name": "smart_track_info"},
	})
}
