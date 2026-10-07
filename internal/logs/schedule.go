package logs

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// Schedule is local time-of-day for log rotation timer (same shape as backup).
// Default: 01:00 UTC ≈ 04:00 MSK.
type Schedule struct {
	Hour   int  `json:"hour"`
	Minute int  `json:"minute"`
	UTC    bool `json:"utc"`
	// KeepHours: journal window after vacuum (default 24).
	KeepHours int `json:"keep_hours,omitempty"`
}

func schedulePath() string {
	return filepath.Join(paths.StateDir(), "logs_schedule.json")
}

func DefaultSchedule() Schedule {
	return Schedule{Hour: 1, Minute: 0, UTC: true, KeepHours: 24}
}

func LoadSchedule() Schedule {
	b, err := os.ReadFile(schedulePath())
	if err != nil {
		return DefaultSchedule()
	}
	var s Schedule
	if json.Unmarshal(b, &s) != nil {
		return DefaultSchedule()
	}
	if s.Hour < 0 || s.Hour > 23 {
		s.Hour = 1
	}
	if s.Minute < 0 || s.Minute > 59 {
		s.Minute = 0
	}
	if s.KeepHours < 1 {
		s.KeepHours = 24
	}
	if s.KeepHours > 168 {
		s.KeepHours = 168
	}
	return s
}

func SaveSchedule(s Schedule) error {
	if s.KeepHours < 1 {
		s.KeepHours = 24
	}
	_ = os.MkdirAll(filepath.Dir(schedulePath()), 0o700)
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(schedulePath(), append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return ApplySchedule(s)
}

// ApplySchedule writes systemd timer drop-in and enables unit.
func ApplySchedule(s Schedule) error {
	if err := ensureUnits(); err != nil {
		return err
	}
	dir := "/etc/systemd/system/netductor-logs-rotate.timer.d"
	_ = os.MkdirAll(dir, 0o755)
	cal := fmt.Sprintf("*-*-* %02d:%02d:00", s.Hour, s.Minute)
	content := "[Timer]\nOnCalendar=\nOnCalendar=" + cal + "\nPersistent=true\n"
	if err := os.WriteFile(filepath.Join(dir, "schedule.conf"), []byte(content), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", "--now", "netductor-logs-rotate.timer").Run()
	return nil
}

func FormatSchedule(s Schedule) string {
	note := "host local"
	if s.UTC {
		note = "UTC (~MSK=UTC+3)"
	}
	return fmt.Sprintf("%02d:%02d (%s) keep=%dh", s.Hour, s.Minute, note, s.KeepHours)
}

func ensureUnits() error {
	svc := `[Unit]
Description=netductor log rotation (journal vacuum + export cleanup)
[Service]
Type=oneshot
ExecStart=/usr/local/bin/netductor logs rotate
`
	tmr := `[Unit]
Description=netductor log rotation timer
[Timer]
OnCalendar=*-*-* 01:00:00
Persistent=true
[Install]
WantedBy=timers.target
`
	if err := os.WriteFile("/etc/systemd/system/netductor-logs-rotate.service", []byte(svc), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("/etc/systemd/system/netductor-logs-rotate.timer", []byte(tmr), 0o644); err != nil {
		return err
	}
	return nil
}

// EnsureTimer installs units + applies saved schedule (idempotent).
func EnsureTimer() error {
	s := LoadSchedule()
	return ApplySchedule(s)
}

func runOut(name string, args ...string) string {
	out, _ := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out))
}
