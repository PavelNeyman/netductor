package fleet

import (
	"os/exec"
)

// DisableLegacyFleetUnits stops mirror/failover units if present on this host.
func DisableLegacyFleetUnits() {
	for _, u := range []string{
		"netductor-fleet-sync.timer",
		"netductor-fleet-sync.service",
		"netductor-bot-failover.timer",
		"netductor-bot-failover.service",
		"netductor-telegram-bot-standby.service",
		"netductor-tg-socks-tunnel.service",
	} {
		_ = exec.Command("systemctl", "disable", "--now", u).Run()
	}
}
