package deploy

import (
	"fmt"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
)

// factsProbeScript runs on OpenWrt (ash-compatible). Emits KEY=value and list lines.
const factsProbeScript = `set +e
echo ARCH=$(uname -m 2>/dev/null)
if [ -f /etc/openwrt_release ]; then
  echo OS=openwrt
  . /etc/openwrt_release 2>/dev/null
  echo BOARD=${DISTRIB_ID:-openwrt}/${DISTRIB_RELEASE:-}
else
  echo OS=other
fi
if [ -f /tmp/sysinfo/board_name ]; then
  echo BOARD_NAME=$(cat /tmp/sysinfo/board_name 2>/dev/null)
elif [ -f /etc/board.json ]; then
  echo BOARD_NAME=$(sed -n 's/.*"id":[[:space:]]*"\([^"]*\)".*/\1/p' /etc/board.json 2>/dev/null | head -1)
fi
echo LAN_DEVICE=$(uci -q get network.lan.device 2>/dev/null)
echo LAN_IP=$(uci -q get network.lan.ipaddr 2>/dev/null)
WP=$(uci -q get network.wan.proto 2>/dev/null)
if [ -n "$WP" ]; then
  echo WAN_PRESENT=1
  echo WAN_PROTO=$WP
else
  echo WAN_PRESENT=0
fi
# interfaces: IFACE name type up(0|1)
ip -o link 2>/dev/null | while read -r idx rest; do
  name=$(echo "$rest" | cut -d: -f1 | tr -d ' ')
  case "$name" in
    lo|br-*|docker*|veth*|tun*|wg*|nd-*) continue ;;
  esac
  [ -z "$name" ] && continue
  up=0
  echo "$rest" | grep -q 'state UP' && up=1
  type=other
  case "$name" in
    eth*|lan*|wan*|usb*) type=ethernet ;;
    wlan*|wl*|ra*|rax*|phy*) type=wireless ;;
  esac
  echo IFACE $name $type $up
done
# radios from UCI wifi-device
uci -q show wireless 2>/dev/null | grep '=wifi-device' | while read -r line; do
  # wireless.radio0=wifi-device
  r=$(echo "$line" | sed -n 's/^wireless\.\([^=]*\)=.*/\1/p')
  [ -z "$r" ] && continue
  band=$(uci -q get wireless.$r.band 2>/dev/null)
  if [ -z "$band" ]; then
    hw=$(uci -q get wireless.$r.hwmode 2>/dev/null)
    case "$hw" in
      *a*|*ac*|*ax*) band=5g ;;
      *g*|*b*|*n*) band=2g ;;
      *) band=unknown ;;
    esac
  fi
  echo RADIO $r $band
done
`

// ProbeDeviceFacts SSHs to the router and fills edge.DeviceFacts (no uplink required).
func ProbeDeviceFacts(password, keyPath, user, host, keyPassphrase string) (edge.DeviceFacts, string, error) {
	var f edge.DeviceFacts
	out, err := runSSHOnPort(factorySSHPort(), password, keyPath, user, host, factsProbeScript, keyPassphrase)
	raw := out
	if err != nil {
		return f, raw, fmt.Errorf("probe device facts: %w\n%s", err, out)
	}
	f = parseFactsProbe(out)
	return f, raw, nil
}

func parseFactsProbe(out string) edge.DeviceFacts {
	var f edge.DeviceFacts
	f.OS = "openwrt"
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Warning:") {
			continue
		}
		if strings.Contains(line, "post-quantum") {
			continue
		}
		switch {
		case strings.HasPrefix(line, "ARCH="):
			f.Arch = edge.NormalizeArch(strings.TrimPrefix(line, "ARCH="))
		case strings.HasPrefix(line, "OS="):
			f.OS = strings.TrimPrefix(line, "OS=")
		case strings.HasPrefix(line, "BOARD_NAME="):
			f.Board = strings.TrimPrefix(line, "BOARD_NAME=")
		case strings.HasPrefix(line, "BOARD=") && f.Board == "":
			f.Board = strings.TrimPrefix(line, "BOARD=")
		case strings.HasPrefix(line, "LAN_DEVICE="):
			f.UCI.LANDevice = strings.TrimPrefix(line, "LAN_DEVICE=")
		case strings.HasPrefix(line, "LAN_IP="):
			f.UCI.LANIP = strings.TrimPrefix(line, "LAN_IP=")
		case strings.HasPrefix(line, "WAN_PRESENT="):
			f.UCI.WANPresent = strings.TrimPrefix(line, "WAN_PRESENT=") == "1"
		case strings.HasPrefix(line, "WAN_PROTO="):
			f.UCI.WANProto = strings.TrimPrefix(line, "WAN_PROTO=")
		case strings.HasPrefix(line, "IFACE "):
			parts := strings.Fields(line)
			if len(parts) >= 4 {
				f.Ifaces = append(f.Ifaces, edge.NetIface{
					Name: parts[1],
					Type: parts[2],
					Up:   parts[3] == "1",
				})
			}
		case strings.HasPrefix(line, "RADIO "):
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				band := "unknown"
				if len(parts) >= 3 {
					band = parts[2]
				}
				f.Radios = append(f.Radios, edge.RadioFact{Name: parts[1], Band: band})
			}
		}
	}
	return f
}

// FormatPlanHuman prints deploy plan for CLI dry-run.
func FormatPlanHuman(preset string, facts edge.DeviceFacts, plan edge.DeployPlan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "preset: %s (suggested: %s)\n", plan.Preset, edge.SuggestPreset(facts))
	fmt.Fprintf(&b, "facts: arch=%s board=%s os=%s wan_capable=%v eth=%d radios=%d lan_ip=%s\n",
		facts.Arch, facts.Board, facts.OS, facts.WANCapable(), facts.EthernetCount(), len(facts.Radios), facts.UCI.LANIP)
	for _, s := range plan.Steps {
		fmt.Fprintf(&b, "  %-16s %-5s %s\n", s.Module, s.Action, s.Reason)
	}
	return b.String()
}
