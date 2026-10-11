package gitstore

import (
	"fmt"
	"os"
	"os/exec"
)

const (
	syncAllService = "netductor-git-sync-all.service"
	syncAllTimer   = "netductor-git-sync-all.timer"
	cleanupService = "netductor-cleanup-weekly.service"
	cleanupTimer   = "netductor-cleanup-weekly.timer"
)

// InstallSyncAllTimer fetches all project mirrors periodically (GH → primary).
// Default: every 6 hours. Disable: systemctl disable --now netductor-git-sync-all.timer
func InstallSyncAllTimer() error {
	bin := "/usr/local/bin/netductor"
	svc := fmt.Sprintf(`[Unit]
Description=Netductor git project sync-all (GH → primary)
After=network-online.target

[Service]
Type=oneshot
ExecStart=%s git project sync-all
Nice=10
`, bin)
	tmr := `[Unit]
Description=Netductor git sync-all every 6h

[Timer]
OnBootSec=10min
OnUnitActiveSec=6h
Persistent=true

[Install]
WantedBy=timers.target
`
	if err := os.WriteFile("/etc/systemd/system/"+syncAllService, []byte(svc), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("/etc/systemd/system/"+syncAllTimer, []byte(tmr), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", "--now", syncAllTimer).Run()
	return nil
}

// InstallCleanupWeeklyTimer runs netductor cleanup --apply weekly (docker prune + leftovers).
func InstallCleanupWeeklyTimer() error {
	bin := "/usr/local/bin/netductor"
	svc := fmt.Sprintf(`[Unit]
Description=Netductor weekly cleanup (docker prune + leftovers)
After=network-online.target docker.service

[Service]
Type=oneshot
ExecStart=%s cleanup --apply
Nice=15
`, bin)
	tmr := `[Unit]
Description=Netductor weekly cleanup

[Timer]
OnCalendar=Sun 04:30
Persistent=true
RandomizedDelaySec=30min

[Install]
WantedBy=timers.target
`
	if err := os.WriteFile("/etc/systemd/system/"+cleanupService, []byte(svc), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("/etc/systemd/system/"+cleanupTimer, []byte(tmr), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", "--now", cleanupTimer).Run()
	return nil
}

// EnsureGitOpsTimers installs sync-all + weekly cleanup timers (idempotent).
func EnsureGitOpsTimers() error {
	if err := InstallSyncAllTimer(); err != nil {
		return fmt.Errorf("sync-all timer: %w", err)
	}
	if err := InstallCleanupWeeklyTimer(); err != nil {
		return fmt.Errorf("cleanup timer: %w", err)
	}
	return nil
}
