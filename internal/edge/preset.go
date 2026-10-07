package edge

import "strings"

// Deploy presets: default module sets before facts filter.
const (
	PresetTravelRouter = "travel-router" // Cudy-like lan+wan
	PresetSBCLab       = "sbc-lab"       // RPi single-NIC lab
	PresetSBCDualNIC   = "sbc-dual-nic"  // RPi + USB ethernet as wan
)

// Module IDs for compose plan.
const (
	ModAgentInstall = "agent_install"
	ModLANBaseline  = "lan_baseline"
	ModWANBaseline  = "wan_baseline"
	ModWiFiAP       = "wifi_ap"
	ModGuest        = "guest"
	ModVPNClient    = "vpn_client"
	ModSSHHarden    = "ssh_harden"
)

// AllModuleOrder is the canonical execution order.
var AllModuleOrder = []string{
	ModAgentInstall,
	ModLANBaseline,
	ModWANBaseline,
	ModWiFiAP,
	ModGuest,
	ModVPNClient,
	ModSSHHarden,
}

// PresetModules returns the modules a preset wants (before facts).
func PresetModules(preset string) []string {
	switch strings.TrimSpace(preset) {
	case PresetSBCLab:
		return []string{
			ModAgentInstall, ModLANBaseline, ModWiFiAP, ModGuest, ModSSHHarden,
		}
	case PresetSBCDualNIC:
		return []string{
			ModAgentInstall, ModLANBaseline, ModWANBaseline, ModWiFiAP, ModGuest, ModVPNClient, ModSSHHarden,
		}
	case PresetTravelRouter, "":
		fallthrough
	default:
		return []string{
			ModAgentInstall, ModLANBaseline, ModWANBaseline, ModWiFiAP, ModGuest, ModVPNClient, ModSSHHarden,
		}
	}
}

// ValidPreset reports known preset IDs.
func ValidPreset(id string) bool {
	switch strings.TrimSpace(id) {
	case PresetTravelRouter, PresetSBCLab, PresetSBCDualNIC, "":
		return true
	default:
		return false
	}
}

// SuggestPreset picks a preset from facts (hint only).
func SuggestPreset(f DeviceFacts) string {
	if f.WANCapable() {
		return PresetTravelRouter
	}
	if f.EthernetCount() <= 1 {
		return PresetSBCLab
	}
	return PresetTravelRouter
}
