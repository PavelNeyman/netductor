package fleet

import (
	"os"
	"os/exec"
)

// InstallSyncTimer runs fleet sync periodically from primary (data → secondary).
func InstallSyncTimer() error {
	svc := `[Unit]
Description=Netductor fleet sync once

[Service]
Type=oneshot
ExecStart=/usr/local/bin/netductor fleet sync
`
	timer := `[Unit]
Description=Netductor fleet sync hourly

[Timer]
OnCalendar=hourly
Persistent=true
RandomizedDelaySec=10m

[Install]
WantedBy=timers.target
`
	_ = os.MkdirAll("/etc/systemd/system", 0o755)
	if err := os.WriteFile("/etc/systemd/system/netductor-fleet-sync.service", []byte(svc), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("/etc/systemd/system/netductor-fleet-sync.timer", []byte(timer), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", "--now", "netductor-fleet-sync.timer").Run()
	return nil
}
