package nvr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ScheduleWindow is local-time HH:MM range (inclusive start, exclusive end).
// Days: 0=Sun … 6=Sat; empty = every day.
type ScheduleWindow struct {
	Start string `json:"start"` // "09:00"
	End   string `json:"end"`   // "18:00"
	Days  []int  `json:"days,omitempty"`
}

// MotionConfig — phase-1: when alerts/recording windows are active (not CV yet).
type MotionConfig struct {
	Enabled   bool             `json:"enabled"`
	Timezone  string           `json:"timezone"` // e.g. Europe/Moscow; empty = local
	Windows   []ScheduleWindow `json:"windows"`
	// AlertOnSegment: emit event when new segment arrives during window.
	AlertOnSegment bool `json:"alert_on_segment"`
}

func motionPath() string {
	return filepath.Join(dir(), "motion.json")
}

func eventsPath() string {
	return filepath.Join(dir(), "events.jsonl")
}

// LoadMotion returns motion/schedule config.
func LoadMotion() MotionConfig {
	mu.Lock()
	defer mu.Unlock()
	c := MotionConfig{Enabled: false, AlertOnSegment: true}
	b, err := os.ReadFile(motionPath())
	if err != nil {
		return c
	}
	_ = json.Unmarshal(b, &c)
	return c
}

// SaveMotion persists motion config.
func SaveMotion(c MotionConfig) error {
	mu.Lock()
	defer mu.Unlock()
	if err := ensure(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := motionPath() + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, motionPath())
}

// InMotionWindow reports whether now falls into a configured window.
// If disabled or no windows → true (always on).
func InMotionWindow(c MotionConfig, now time.Time) bool {
	if !c.Enabled {
		return true
	}
	if len(c.Windows) == 0 {
		return true
	}
	if c.Timezone != "" {
		if loc, err := time.LoadLocation(c.Timezone); err == nil {
			now = now.In(loc)
		}
	}
	wd := int(now.Weekday())
	hm := now.Hour()*60 + now.Minute()
	for _, w := range c.Windows {
		if len(w.Days) > 0 {
			ok := false
			for _, d := range w.Days {
				if d == wd {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		s := parseHHMM(w.Start)
		e := parseHHMM(w.End)
		if s < 0 || e < 0 {
			continue
		}
		if s <= e {
			if hm >= s && hm < e {
				return true
			}
		} else {
			// overnight
			if hm >= s || hm < e {
				return true
			}
		}
	}
	return false
}

func parseHHMM(s string) int {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return -1
	}
	h, m := 0, 0
	for _, c := range parts[0] {
		if c < '0' || c > '9' {
			return -1
		}
		h = h*10 + int(c-'0')
	}
	for _, c := range parts[1] {
		if c < '0' || c > '9' {
			return -1
		}
		m = m*10 + int(c-'0')
	}
	if h > 23 || m > 59 {
		return -1
	}
	return h*60 + m
}

// Event is an NVR activity line.
type Event struct {
	TS       int64  `json:"ts"`
	Type     string `json:"type"`
	CameraID string `json:"camera_id,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

// AppendEvent writes one event (best-effort).
func AppendEvent(typ, cameraID, detail string) {
	mu.Lock()
	defer mu.Unlock()
	_ = ensure()
	ev := Event{TS: Now().Unix(), Type: typ, CameraID: cameraID, Detail: detail}
	b, _ := json.Marshal(ev)
	f, err := os.OpenFile(eventsPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(b, '\n'))
}

// ListEventsTail returns last n events.
func ListEventsTail(n int) []Event {
	mu.Lock()
	defer mu.Unlock()
	b, err := os.ReadFile(eventsPath())
	if err != nil {
		return nil
	}
	lines := strings.Split(string(b), "\n")
	var out []Event
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e Event
		if json.Unmarshal([]byte(line), &e) == nil {
			out = append(out, e)
		}
	}
	if n <= 0 || len(out) <= n {
		return out
	}
	return out[len(out)-n:]
}
