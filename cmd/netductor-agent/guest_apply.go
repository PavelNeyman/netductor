package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/guest"
)

// applyGuestNetwork writes OpenWrt UCI for guest AP + isolation + captive redirect.
// Internet for guests is NOT open by zone forwarding — only via nft MAC allow-list.
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
	ip := "192.168.50.1"
	if i := strings.Index(gc.SubnetCIDR, "/"); i > 0 {
		ip = gc.SubnetCIDR[:i]
	}
	capPort := gc.CaptivePort
	if capPort == 0 {
		capPort = 7881
	}
	// Zone: forward REJECT — WAN only through nft allow set.
	// No guest→wan forwarding section (would bypass MAC gate).
	batch := fmt.Sprintf(`
set network.guest=interface
set network.guest.proto=static
set network.guest.ipaddr=%s
set network.guest.netmask=255.255.255.0
set network.guest.device=br-guest
set network.guest.type=bridge
set network.guest6=interface
set network.guest6.proto=none
set network.guest6.device=@guest
set dhcp.guest=dhcp
set dhcp.guest.interface=guest
set dhcp.guest.start=100
set dhcp.guest.limit=100
set dhcp.guest.leasetime=1h
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
set firewall.guest_http_redir=redirect
set firewall.guest_http_redir.name=Guest-Captive-HTTP
set firewall.guest_http_redir.src=guest
set firewall.guest_http_redir.src_dport=80
set firewall.guest_http_redir.dest_port=%d
set firewall.guest_http_redir.proto=tcp
set firewall.guest_http_redir.target=DNAT
delete firewall.guest_wan
commit network
commit dhcp
commit wireless
commit firewall
`, ip, uciQuote(ssid), uciQuote(psk), hidden, capPort, capPort)
	res := uciBatch(batch)
	_ = installGuestNFTHooks()
	_ = applyGuestVPNBypass(ip)
	_ = exec.Command("/etc/init.d/network", "reload").Run()
	_ = exec.Command("/etc/init.d/dnsmasq", "restart").Run()
	_ = exec.Command("wifi", "reload").Run()
	_ = exec.Command("/etc/init.d/firewall", "reload").Run()
	time.Sleep(500 * time.Millisecond)
	if st, err := guest.OpenStore(guestStoreDir()); err == nil {
		_ = applyGuestFirewallAllow(st)
	}
	return res
}

func uciQuote(s string) string {
	return strings.ReplaceAll(s, "'", "")
}

func installGuestNFTHooks() error {
	dir := "/etc/nftables.d"
	_ = os.MkdirAll(dir, 0o755)
	// fw4 include: drop guest→wan unless ether saddr in set (populated by sync).
	nft := `# netductor guest MAC gate — managed by netductor-agent
table inet netductor_guest {
	set allowed {
		type ether_addr
		flags timeout
		timeout 1d
	}
	chain forward {
		type filter hook forward priority -5; policy accept;
		iifname "br-guest" oifname "br-guest" accept
		iifname "br-guest" ether saddr @allowed accept
		iifname "br-guest" counter drop comment "netductor-guest-deny"
	}
}
`
	path := filepath.Join(dir, "90-netductor-guest.nft")
	if err := os.WriteFile(path, []byte(nft), 0o644); err != nil {
		// fallback agent dir (read by sync script applying nft -f)
		alt := filepath.Join(agentDir(), "guest", "90-netductor-guest.nft")
		_ = os.MkdirAll(filepath.Dir(alt), 0o700)
		_ = os.WriteFile(alt, []byte(nft), 0o644)
	}
	script := `#!/bin/sh
# Sync guest allow-list into nft set netductor_guest.allowed
# Arg1: path to allowed.mac (one MAC per line)
set -e
FILE="${1:-/etc/netductor-agent/guest/allowed.mac}"
TABLE=inet
# Ensure table/set exist
if ! nft list table inet netductor_guest >/dev/null 2>&1; then
  if [ -f /etc/nftables.d/90-netductor-guest.nft ]; then
    nft -f /etc/nftables.d/90-netductor-guest.nft 2>/dev/null || true
  elif [ -f /etc/netductor-agent/guest/90-netductor-guest.nft ]; then
    nft -f /etc/netductor-agent/guest/90-netductor-guest.nft 2>/dev/null || true
  fi
fi
nft flush set inet netductor_guest allowed 2>/dev/null || true
[ -f "$FILE" ] || exit 0
# Format: mac OR mac|timeout_seconds
while IFS= read -r line || [ -n "$line" ]; do
  line=$(echo "$line" | tr -d '\r' | sed 's/#.*//' | tr -s ' ')
  [ -z "$line" ] && continue
  mac=$(echo "$line" | cut -d'|' -f1 | tr 'A-Z' 'a-z')
  to=$(echo "$line" | cut -d'|' -f2)
  case "$mac" in
    *:*) ;;
    *) continue ;;
  esac
  if [ -n "$to" ] && [ "$to" != "$mac" ]; then
    nft add element inet netductor_guest allowed "{ $mac timeout ${to}s }" 2>/dev/null || \
      nft add element inet netductor_guest allowed "{ $mac }" 2>/dev/null || true
  else
    nft add element inet netductor_guest allowed "{ $mac }" 2>/dev/null || true
  fi
done < "$FILE"
`
	libexec := "/usr/libexec/netductor-guest-sync.sh"
	_ = os.MkdirAll("/usr/libexec", 0o755)
	if err := os.WriteFile(libexec, []byte(script), 0o755); err != nil {
		alt := filepath.Join(agentDir(), "guest", "sync.sh")
		_ = os.WriteFile(alt, []byte(script), 0o755)
	}
	// try load table now
	_ = exec.Command("nft", "-f", path).Run()
	return nil
}

// applyGuestVPNBypass keeps guest subnet on main/wan table (not VPN/tun).
func applyGuestVPNBypass(gatewayIP string) error {
	// Derive /24 from gateway 192.168.50.1 → 192.168.50.0/24
	parts := strings.Split(gatewayIP, ".")
	subnet := "192.168.50.0/24"
	if len(parts) == 4 {
		subnet = fmt.Sprintf("%s.%s.%s.0/24", parts[0], parts[1], parts[2])
	}
	// idempotent: delete then add
	_ = exec.Command("ip", "rule", "del", "from", subnet, "lookup", "main", "priority", "48").Run()
	_ = exec.Command("ip", "rule", "add", "from", subnet, "lookup", "main", "priority", "48").Run()
	// persist via hotplug-ish script
	body := fmt.Sprintf("#!/bin/sh\nip rule del from %s lookup main priority 48 2>/dev/null\nip rule add from %s lookup main priority 48\n", subnet, subnet)
	_ = os.MkdirAll("/etc/netductor-agent/guest", 0o700)
	path := "/etc/netductor-agent/guest/vpn-bypass.sh"
	_ = os.WriteFile(path, []byte(body), 0o755)
	// OpenWrt hotplug iface
	hp := fmt.Sprintf("#!/bin/sh\n[ \"$ACTION\" = ifup ] || exit 0\n[ \"$INTERFACE\" = guest ] || [ \"$INTERFACE\" = wan ] || exit 0\n%s\n", body)
	_ = os.MkdirAll("/etc/hotplug.d/iface", 0o755)
	_ = os.WriteFile("/etc/hotplug.d/iface/48-netductor-guest-vpn", []byte(hp), 0o755)
	return nil
}

// applyGuestFirewallAllow writes MAC list and syncs nft set (with timeouts).
func applyGuestFirewallAllow(store *guest.Store) error {
	allow, _ := store.Snapshot()
	dir := filepath.Join(agentDir(), "guest")
	_ = os.MkdirAll(dir, 0o700)
	path := filepath.Join(dir, "allowed.mac")
	var b strings.Builder
	now := time.Now()
	for _, e := range allow {
		sec := int(e.ExpiresAt.Sub(now).Seconds())
		if sec < 1 {
			continue
		}
		b.WriteString(e.MAC)
		b.WriteByte('|')
		b.WriteString(fmt.Sprintf("%d", sec))
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return err
	}
	_ = installGuestNFTHooks()
	for _, syn := range []string{
		"/usr/libexec/netductor-guest-sync.sh",
		filepath.Join(agentDir(), "guest", "sync.sh"),
	} {
		if st, err := os.Stat(syn); err == nil && st.Mode()&0o111 != 0 {
			out, err := exec.Command(syn, path).CombinedOutput()
			if err != nil {
				fmt.Fprintf(os.Stderr, "guest-sync: %v %s\n", err, truncate(string(out), 200))
				continue
			}
			return nil
		}
	}
	// pure nft fallback without script
	return nftSyncAllowDirect(allow, now)
}

func nftSyncAllowDirect(allow []guest.AllowEntry, now time.Time) error {
	_ = exec.Command("nft", "list", "table", "inet", "netductor_guest").Run()
	_ = exec.Command("nft", "flush", "set", "inet", "netductor_guest", "allowed").Run()
	for _, e := range allow {
		sec := int(e.ExpiresAt.Sub(now).Seconds())
		if sec < 1 {
			continue
		}
		elem := fmt.Sprintf("{ %s timeout %ds }", e.MAC, sec)
		_ = exec.Command("nft", "add", "element", "inet", "netductor_guest", "allowed", elem).Run()
	}
	return nil
}

// startGuestExpireLoop periodically prunes store + resyncs nft.
func startGuestExpireLoop(store *guest.Store) {
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			_, _ = store.Snapshot() // prunes
			_ = applyGuestFirewallAllow(store)
		}
	}()
}
