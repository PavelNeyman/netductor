package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// BackupSchedule is local time-of-day for netductor-backup.timer (host TZ or UTC offset).
// Default: 01:00 UTC ≈ 04:00 MSK.
type BackupSchedule struct {
	Hour   int  `json:"hour"`   // 0-23
	Minute int  `json:"minute"` // 0-59
	UTC    bool `json:"utc"`    // if true OnCalendar uses UTC
}

func backupSchedulePath() string {
	return filepath.Join(paths.StateDir(), "backup_schedule.json")
}

func DefaultBackupSchedule() BackupSchedule {
	return BackupSchedule{Hour: 1, Minute: 0, UTC: true} // 04:00 MSK
}

func LoadBackupSchedule() BackupSchedule {
	b, err := os.ReadFile(backupSchedulePath())
	if err != nil {
		return DefaultBackupSchedule()
	}
	var s BackupSchedule
	if json.Unmarshal(b, &s) != nil {
		return DefaultBackupSchedule()
	}
	if s.Hour < 0 || s.Hour > 23 {
		s.Hour = 1
	}
	if s.Minute < 0 || s.Minute > 59 {
		s.Minute = 0
	}
	return s
}

func SaveBackupSchedule(s BackupSchedule) error {
	_ = os.MkdirAll(filepath.Dir(backupSchedulePath()), 0o700)
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(backupSchedulePath(), append(raw, '\n'), 0o600); err != nil {
		return err
	}
	return ApplyBackupSchedule(s)
}

// ApplyBackupSchedule writes systemd drop-in and reloads timer.
func ApplyBackupSchedule(s BackupSchedule) error {
	dir := "/etc/systemd/system/netductor-backup.timer.d"
	_ = os.MkdirAll(dir, 0o755)
	cal := fmt.Sprintf("*-*-* %02d:%02d:00", s.Hour, s.Minute)
	content := "[Timer]\nOnCalendar=\nOnCalendar=" + cal + "\nPersistent=true\n"
	if err := os.WriteFile(filepath.Join(dir, "schedule.conf"), []byte(content), 0o644); err != nil {
		return err
	}
	_ = run("systemctl", "daemon-reload")
	_ = run("systemctl", "restart", "netductor-backup.timer")
	return nil
}

func FormatBackupSchedule(s BackupSchedule) string {
	note := "host local"
	if s.UTC {
		note = "UTC (~MSK = UTC+3)"
	}
	return fmt.Sprintf("%02d:%02d (%s)", s.Hour, s.Minute, note)
}

func ParseHHMM(s string) (hour, minute int, err error) {
	s = strings.TrimSpace(s)
	var h, m int
	if _, err = fmt.Sscanf(s, "%d:%d", &h, &m); err != nil {
		return 0, 0, fmt.Errorf("use HH:MM")
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("invalid time")
	}
	return h, m, nil
}
