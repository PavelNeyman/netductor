package mikrotik

import (
	"fmt"
	"strings"
)

// ClientRSC generates a RouterOS script for identity + heartbeat scheduler.
// Full VLESS+Reality client on ROS is limited; VPN path is documented separately.
func ClientRSC(name, relayIP, coreURL, note string) string {
	var b strings.Builder
	b.WriteString("# netductor MikroTik bootstrap (generated; not hardware-tested)\n")
	b.WriteString("# RouterOS 7.12+ recommended; containers not required\n")
	b.WriteString(fmt.Sprintf("# device=%s relay=%s\n", name, relayIP))
	b.WriteString(fmt.Sprintf("/system identity set name=\"%s\"\n", name))
	n := strings.ReplaceAll(note, "\"", "'")
	b.WriteString(fmt.Sprintf("/system note set show-at-login=yes note=\"%s\"\n", n))
	b.WriteString("/system script remove [find name=nd-heartbeat]\n")
	b.WriteString("/system script add name=nd-heartbeat policy=read,write,test,api source={\n")
	b.WriteString(fmt.Sprintf("  :local url \"%s/api/edge/heartbeat\"\n", strings.TrimRight(coreURL, "/")))
	b.WriteString("  /tool fetch url=$url http-method=post keep-result=no\n")
	b.WriteString("}\n")
	b.WriteString("/system scheduler remove [find name=nd-hb]\n")
	b.WriteString("/system scheduler add name=nd-hb interval=1m on-event=nd-heartbeat\n")
	b.WriteString(fmt.Sprintf("# Probe relay %s:443; on failure disable interface-list nd-vpn (WAN fallback)\n", relayIP))
	return b.String()
}

// Policy documents the management model without requiring live ROS.
func Policy() string {
	return "enroll via RSC + edge heartbeat; commands via polled JSON; VPN interim WG/external until native Reality"
}
