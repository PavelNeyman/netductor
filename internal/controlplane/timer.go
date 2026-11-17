package controlplane

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	syncService = "netductor-control-sync.service"
	syncTimer   = "netductor-control-sync.timer"
)

// InstallSyncTimer installs a systemd timer that runs sync-pull on standby only.
// Interval: 30 minutes. Does not auto-restore or promote.
func InstallSyncTimer() error {
	if Get() != RoleStandby {
		return fmt.Errorf("control sync timer: install only when control_role=standby (current=%s)", Get())
	}
	if SyncURL() == "" {
		return fmt.Errorf("control sync timer: set NETDUCTOR_CONTROL_SYNC_URL or CONTROL_SYNC_URL")
	}
	if SyncToken() == "" {
		return fmt.Errorf("control sync timer: missing control_standby_token (or NETDUCTOR_CONTROL_SYNC_TOKEN)")
	}
	svc := `[Unit]
Description=Netductor control standby SoT pull (inbox only)
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
EnvironmentFile=-/etc/netductor/netductor.conf
ExecStart=/usr/local/bin/netductor control sync-pull
Nice=10

[Install]
WantedBy=multi-user.target
`
	timer := `[Unit]
Description=Netductor control standby SoT pull timer (30min)

[Timer]
OnBootSec=5min
OnUnitActiveSec=30min
Persistent=true
Unit=netductor-control-sync.service

[Install]
WantedBy=timers.target
`
	if err := os.WriteFile("/etc/systemd/system/"+syncService, []byte(svc), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("/etc/systemd/system/"+syncTimer, []byte(timer), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	if err := exec.Command("systemctl", "enable", "--now", syncTimer).Run(); err != nil {
		return fmt.Errorf("enable %s: %w", syncTimer, err)
	}
	fmt.Fprintln(os.Stderr, "control sync timer enabled (30min, inbox only — no auto restore/promote)")
	return nil
}

// DisableSyncTimer stops and disables the timer (units left on disk).
func DisableSyncTimer() error {
	_ = exec.Command("systemctl", "disable", "--now", syncTimer).Run()
	_ = exec.Command("systemctl", "stop", syncService).Run()
	fmt.Fprintln(os.Stderr, "control sync timer disabled")
	return nil
}

// SyncTimerStatus is a short human line for CLI/doctor.
func SyncTimerStatus() string {
	out, err := exec.Command("systemctl", "is-active", syncTimer).Output()
	state := strings.TrimSpace(string(out))
	if err != nil && state == "" {
		state = "inactive"
	}
	enabled, _ := exec.Command("systemctl", "is-enabled", syncTimer).Output()
	en := strings.TrimSpace(string(enabled))
	if en == "" {
		en = "disabled"
	}
	return fmt.Sprintf("timer=%s enabled=%s", state, en)
}
