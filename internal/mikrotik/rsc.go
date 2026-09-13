package mikrotik

import (
	"fmt"
	"strings"
)

// ClientRSC is optional identity/note bootstrap only.
// VPN must NOT run on RouterOS — use RPi OpenWrt on the same LAN (see docs/MIKROTIK.md).
func ClientRSC(name, relayIP, coreURL, note string) string {
	var b strings.Builder
	nl := "\n"
	b.WriteString("# netductor MikroTik — routing host only, no VLESS on ROS" + nl)
	b.WriteString("# Put OpenWrt (RPi) on same LAN as VPN edge; point policy routes to RPi" + nl)
	b.WriteString(fmt.Sprintf("# name=%s  (relay %s is for RPi, not this ROS)%s", name, relayIP, nl))
	b.WriteString(fmt.Sprintf("/system identity set name=%q%s", name, nl))
	n := strings.ReplaceAll(note, "\"", "'")
	b.WriteString(fmt.Sprintf("/system note set show-at-login=yes note=%q%s", n+" — VPN on RPi OpenWrt", nl))
	b.WriteString("# Example: /ip route add dst-address=0.0.0.0/0 gateway=RPI_LAN_IP routing-table=via-rpi" + nl)
	b.WriteString("# Example: /ip firewall mangle ... action=mark-routing new-routing-mark=via-rpi" + nl)
	_ = coreURL
	return b.String()
}

func Policy() string {
	return "ROS routing only; VPN on RPi OpenWrt same LAN; no Reality client on RouterOS"
}
