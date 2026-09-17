package fleet

import "fmt"

// Bot failover / standby on secondary removed (secondary = VPN entry only).

func InstallBotStandbyUnits(primarySSH string) error {
	return fmt.Errorf("bot standby on secondary removed — bot runs on primary only")
}

func InstallBotFailoverTimer() error {
	DisableLegacyFleetUnits()
	return fmt.Errorf("bot-failover removed — bot runs on primary only")
}

func CheckBotFailover() error {
	return fmt.Errorf("bot-failover removed")
}

func PromoteStandbyBot() error {
	return fmt.Errorf("bot-failover removed")
}

func DemoteStandbyBot() error {
	return fmt.Errorf("bot-failover removed")
}
