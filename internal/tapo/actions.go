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
	case dir == "presets":
		return resultJSON(cl.GetPresets())
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
