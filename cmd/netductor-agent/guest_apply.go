package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/guest"
)

// applyGuestNetwork writes OpenWrt UCI batch for guest bridge + wifi + firewall skeleton.
func applyGuestNetwork(gc guest.Config) string {
	if !gc.Enabled {
		return "guest disabled"
	}
	psk := gc.PSK
	if psk == "" {
		return "guest: missing psk"
	}
	ssid := gc.SSID
	if ssid == "" {
		ssid = "Guest"
	}
	hidden := "0"
	if gc.Hidden {
		hidden = "1"
	}
	// IP from CIDR 192.168.50.1/24
	ip := "192.168.50.1"
	if i := strings.Index(gc.SubnetCIDR, "/"); i > 0 {
		ip = gc.SubnetCIDR[:i]
	}
	batch := fmt.Sprintf(`
set network.guest=interface
set network.guest.proto=static
set network.guest.ipaddr=%s
set network.guest.netmask=255.255.255.0
set network.guest.type=bridge
set network.guest.device=br-guest
set wireless.guest24=wifi-iface
set wireless.guest24.device=radio0
set wireless.guest24.mode=ap
set wireless.guest24.network=guest
set wireless.guest24.ssid=%s
set wireless.guest24.encryption=psk2
set wireless.guest24.key=%s
set wireless.guest24.hidden=%s
set wireless.guest24.isolate=1
set firewall.guest=zone
set firewall.guest.name=guest
set firewall.guest.network=guest
set firewall.guest.input=REJECT
set firewall.guest.output=ACCEPT
set firewall.guest.forward=REJECT
set firewall.guest_wan=forwarding
set firewall.guest_wan.src=guest
set firewall.guest_wan.dest=wan
set firewall.guest_dhcp=rule
set firewall.guest_dhcp.name=Guest-DHCP
set firewall.guest_dhcp.src=guest
set firewall.guest_dhcp.proto=udp
set firewall.guest_dhcp.dest_port=67:68
set firewall.guest_dhcp.target=ACCEPT
set firewall.guest_dns=rule
set firewall.guest_dns.name=Guest-DNS
set firewall.guest_dns.src=guest
set firewall.guest_dns.proto=tcpudp
set firewall.guest_dns.dest_port=53
set firewall.guest_dns.target=ACCEPT
set firewall.guest_captive=rule
set firewall.guest_captive.name=Guest-Captive
set firewall.guest_captive.src=guest
set firewall.guest_captive.proto=tcp
set firewall.guest_captive.dest_port=%d
set firewall.guest_captive.target=ACCEPT
commit network
commit wireless
commit firewall
`, ip, uciQuote(ssid), uciQuote(psk), hidden, gc.CaptivePort)
	if gc.CaptivePort == 0 {
		batch = strings.Replace(batch, "dest_port=0", "dest_port=7881", 1)
	}
	res := uciBatch(batch)
	_ = exec.Command("/etc/init.d/network", "reload").Run()
	_ = exec.Command("wifi", "reload").Run()
	_ = exec.Command("/etc/init.d/firewall", "reload").Run()
	return res
}

func uciQuote(s string) string {
	return strings.ReplaceAll(s, "'", "")
}

// applyGuestFirewallAllow syncs nft/macset best-effort from allow-list.
// Full isolation with per-MAC forward requires custom nft; here we document + try ipset.
func applyGuestFirewallAllow(store *guest.Store) error {
	allow, _ := store.Snapshot()
	path := agentDir() + "/guest/allowed.mac"
	var b strings.Builder
	for _, e := range allow {
		b.WriteString(e.MAC)
		b.WriteByte('\n')
	}
	_ = os.WriteFile(path, []byte(b.String()), 0o600)
	// Optional helper script installed by provision
	if _, err := os.Stat("/usr/libexec/netductor-guest-sync.sh"); err == nil {
		return exec.Command("/usr/libexec/netductor-guest-sync.sh", path).Run()
	}
	return nil
}
