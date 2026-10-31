package install

import (
	"fmt"
	"os"
)

// InstallAptGuardTimer runs read-only APT source scan every 6 hours.
func InstallAptGuardTimer() error {
	bin := "/usr/local/bin/netductor"
	unit := fmt.Sprintf(`[Unit]
Description=Netductor APT Repository Guard scan
After=network-online.target

[Service]
Type=oneshot
ExecStart=%s aptguard scan
`, bin)
	timer := `[Unit]
Description=Netductor APT Guard every 6 hours

[Timer]
OnBootSec=5min
OnUnitActiveSec=6h
AccuracySec=5min
Persistent=true

[Install]
WantedBy=timers.target
`
	if err := writeUnit("netductor-aptguard.service", unit); err != nil {
		return err
	}
	if err := writeUnit("netductor-aptguard.timer", timer); err != nil {
		return err
	}
	if err := run("systemctl", "enable", "netductor-aptguard.timer"); err != nil {
		return fmt.Errorf("enable aptguard timer: %w", err)
	}
	if err := run("systemctl", "start", "netductor-aptguard.timer"); err != nil {
		return fmt.Errorf("start aptguard timer: %w", err)
	}
	fmt.Fprintln(os.Stderr, "aptguard timer enabled (6h)")
	return nil
}
