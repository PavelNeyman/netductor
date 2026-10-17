package deploy

import (
	"fmt"
	"os"
	"strconv"
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
if [ -f /tmp/sysinfo/model ]; then
  echo MODEL=$(cat /tmp/sysinfo/model 2>/dev/null)
fi
# Serial: RPi cpuinfo or first eth MAC as stable id fallback
SER=$(awk -F': ' '/^Serial/{print $2; exit}' /proc/cpuinfo 2>/dev/null | tr -d ' ')
if [ -z "$SER" ] || [ "$SER" = "0000000000000000" ]; then
  SER=$(cat /sys/class/net/eth0/address 2>/dev/null)
fi
echo SERIAL=${SER:-}
echo LAN_DEVICE=$(uci -q get network.lan.device 2>/dev/null)
echo LAN_IP=$(uci -q get network.lan.ipaddr 2>/dev/null)
echo LAN_PROTO=$(uci -q get network.lan.proto 2>/dev/null)
WP=$(uci -q get network.wan.proto 2>/dev/null)
if [ -n "$WP" ]; then
  echo WAN_PRESENT=1
  echo WAN_PROTO=$WP
  echo WAN_IP=$(uci -q get network.wan.ipaddr 2>/dev/null)
else
  echo WAN_PRESENT=0
fi
# memory
if [ -f /proc/meminfo ]; then
  echo MEM_TOTAL_KB=$(awk '/MemTotal/{print $2}' /proc/meminfo)
  echo MEM_AVAIL_KB=$(awk '/MemAvailable/{print $2}' /proc/meminfo)
fi
# storage / overlay
if command -v df >/dev/null; then
  # overlay or root
  line=$(df -k /overlay 2>/dev/null | tail -1)
  if [ -n "$line" ]; then
    set -- $line
    echo OVERLAY_TOTAL_KB=$2
    echo OVERLAY_FREE_KB=$4
  fi
  line=$(df -k / 2>/dev/null | tail -1)
  if [ -n "$line" ]; then
    set -- $line
    echo ROOT_TOTAL_KB=$2
    echo ROOT_FREE_KB=$4
  fi
fi
# flash size from /proc/mtd if present (NOR/NAND sum)
if [ -f /proc/mtd ]; then
  awk 'NR>1 {gsub(/["]/,$4); sum+=$2} END{if(sum>0) printf "FLASH_KB=%d\n", sum/2}' /proc/mtd 2>/dev/null
fi
ls /dev/mmcblk* >/dev/null 2>&1 && echo HAS_MMC=1 || echo HAS_MMC=0
ls /dev/sd[a-z] >/dev/null 2>&1 && echo HAS_USB_DISK=1 || echo HAS_USB_DISK=0
# USB disk details (PLAN-EDGE-USB-STORAGE)
for d in /sys/block/sd*; do
  [ -d "$d" ] || continue
  name=$(basename "$d")
  sectors=$(cat "$d/size" 2>/dev/null || echo 0)
  size_kb=$((sectors / 2))
  parts=0
  for p in /dev/${name}[0-9]* /dev/${name}p[0-9]*; do
    [ -b "$p" ] || continue
    parts=$((parts+1))
  done
  echo USB_DISK name=$name size_kb=$size_kb partitions=$parts
done
if command -v block >/dev/null 2>&1; then
  block info 2>/dev/null | while IFS= read -r bi; do
    echo "BLOCK $bi"
  done
fi
ov_src=$(mount 2>/dev/null | awk '/ on \/overlay /{print $1; exit}')
case "$ov_src" in
  /dev/sd*|/dev/mmcblk*) echo EXTROOT_ACTIVE=1 ;;
  *) echo EXTROOT_ACTIVE=0 ;;
esac
EXPAND_HINT=0
if ls /dev/mmcblk0 >/dev/null 2>&1; then
  root_k=$(df -k / 2>/dev/null | tail -1 | awk '{print $2}')
  if [ -n "$root_k" ] && [ "$root_k" -lt 1600000 ] 2>/dev/null; then
    EXPAND_HINT=1
  fi
fi
echo EXPAND_HINT=$EXPAND_HINT
# interfaces (physical-ish only; never lo/bridges/tunnels/usb-tether as "ethernet ports")
ip -o link 2>/dev/null | while read -r idx rest; do
  name=$(echo "$rest" | cut -d: -f1 | tr -d ' ')
  case "$name" in
    lo|br-*|docker*|veth*|tun*|wg*|nd-*|ifb*|dummy*|sit*|gre*|teql*|bond*) continue ;;
  esac
  # skip VLAN/alias eth0.1 eth0@eth1
  case "$name" in
    *.*|*@*) continue ;;
  esac
  [ -z "$name" ] && continue
  up=0
  echo "$rest" | grep -q 'state UP' && up=1
  type=other
  case "$name" in
    eth*|wan|wan[0-9]*|lan[0-9]*) type=ethernet ;;
    lan) type=ethernet ;;
    usb*|wwan*) type=usb ;;
    wlan*|wl*|ra*|rax*|phy*) type=wireless ;;
  esac
  echo IFACE $name $type $up
done
# radios
uci -q show wireless 2>/dev/null | grep '=wifi-device' | while read -r line; do
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
# current wifi ifaces (ssid)
uci -q show wireless 2>/dev/null | grep '=wifi-iface' | while read -r line; do
  s=$(echo "$line" | sed -n 's/^wireless\.\([^=]*\)=.*/\1/p')
  [ -z "$s" ] && continue
  ssid=$(uci -q get wireless.$s.ssid 2>/dev/null)
  mode=$(uci -q get wireless.$s.mode 2>/dev/null)
  dev=$(uci -q get wireless.$s.device 2>/dev/null)
  dis=$(uci -q get wireless.$s.disabled 2>/dev/null)
  [ -z "$ssid" ] && continue
  echo SSID $s ${dev:--} ${mode:--} ${dis:-0} $ssid
done
`

// ProbeDeviceFacts SSHs to the router and fills edge.DeviceFacts (no uplink required).
func ProbeDeviceFacts(password, keyPath, user, host, keyPassphrase string) (edge.DeviceFacts, string, error) {
	var f edge.DeviceFacts
	// Probe must reflect real USB capability: install mass-storage kmods from Mac cache first
	// so card can show disk / Format / overlay (router has no WAN).
	if err := EnsureUSBStorageOnRouter(password, keyPath, user, host, keyPassphrase); err != nil {
		fmt.Fprintln(os.Stderr, "warn: USB storage packages during probe:", err)
	} else {
		_, _ = runSSHOnPort(factorySSHPort(), password, keyPath, user, host, "sleep 2; ls /sys/block/sd* 2>/dev/null || true", keyPassphrase)
	}
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
		case strings.HasPrefix(line, "MODEL="):
			f.Model = strings.TrimPrefix(line, "MODEL=")
		case strings.HasPrefix(line, "SERIAL="):
			f.Serial = strings.TrimPrefix(line, "SERIAL=")
		case strings.HasPrefix(line, "LAN_DEVICE="):
			f.UCI.LANDevice = strings.TrimPrefix(line, "LAN_DEVICE=")
		case strings.HasPrefix(line, "LAN_IP="):
			f.UCI.LANIP = strings.TrimPrefix(line, "LAN_IP=")
		case strings.HasPrefix(line, "LAN_PROTO="):
			f.UCI.LANProto = strings.TrimPrefix(line, "LAN_PROTO=")
		case strings.HasPrefix(line, "WAN_PRESENT="):
			f.UCI.WANPresent = strings.TrimPrefix(line, "WAN_PRESENT=") == "1"
		case strings.HasPrefix(line, "WAN_PROTO="):
			f.UCI.WANProto = strings.TrimPrefix(line, "WAN_PROTO=")
		case strings.HasPrefix(line, "WAN_IP="):
			f.UCI.WANIP = strings.TrimPrefix(line, "WAN_IP=")
		case strings.HasPrefix(line, "MEM_TOTAL_KB="):
			f.MemTotalKB = atoi64(strings.TrimPrefix(line, "MEM_TOTAL_KB="))
		case strings.HasPrefix(line, "MEM_AVAIL_KB="):
			f.MemAvailKB = atoi64(strings.TrimPrefix(line, "MEM_AVAIL_KB="))
		case strings.HasPrefix(line, "OVERLAY_TOTAL_KB="):
			f.Storage.OverlayTotalKB = atoi64(strings.TrimPrefix(line, "OVERLAY_TOTAL_KB="))
		case strings.HasPrefix(line, "OVERLAY_FREE_KB="):
			f.Storage.OverlayFreeKB = atoi64(strings.TrimPrefix(line, "OVERLAY_FREE_KB="))
		case strings.HasPrefix(line, "ROOT_TOTAL_KB="):
			f.Storage.RootTotalKB = atoi64(strings.TrimPrefix(line, "ROOT_TOTAL_KB="))
		case strings.HasPrefix(line, "ROOT_FREE_KB="):
			f.Storage.RootFreeKB = atoi64(strings.TrimPrefix(line, "ROOT_FREE_KB="))
		case strings.HasPrefix(line, "FLASH_KB="):
			f.Storage.FlashKB = atoi64(strings.TrimPrefix(line, "FLASH_KB="))
			if f.FlashMB == 0 && f.Storage.FlashKB > 0 {
				f.FlashMB = int(f.Storage.FlashKB / 1024)
			}
		case strings.HasPrefix(line, "HAS_MMC="):
			f.Storage.HasMMC = strings.TrimPrefix(line, "HAS_MMC=") == "1"
		case strings.HasPrefix(line, "HAS_USB_DISK="):
			f.Storage.HasUSBDisk = strings.TrimPrefix(line, "HAS_USB_DISK=") == "1"
		case strings.HasPrefix(line, "EXTROOT_ACTIVE="):
			f.Storage.ExtrootActive = strings.TrimPrefix(line, "EXTROOT_ACTIVE=") == "1"
		case strings.HasPrefix(line, "USB_DISK "):
			// USB_DISK name=sda size_kb=N partitions=N
			ud := edge.USBDiskFact{}
			for _, kv := range strings.Fields(strings.TrimPrefix(line, "USB_DISK ")) {
				k, v, ok := strings.Cut(kv, "=")
				if !ok {
					continue
				}
				switch k {
				case "name":
					ud.Name = v
				case "size_kb":
					ud.SizeKB = atoi64(v)
				case "partitions":
					n, _ := strconv.Atoi(v)
					ud.Partitions = n
				}
			}
			if ud.Name != "" {
				f.Storage.USBDisks = append(f.Storage.USBDisks, ud)
				f.Storage.HasUSBDisk = true
			}
		case strings.HasPrefix(line, "BLOCK "):
			// Enrich last matching USB disk from block info (UUID TYPE MOUNT)
			bi := strings.TrimPrefix(line, "BLOCK ")
			enrichUSBFromBlockInfo(&f, bi)
		case strings.HasPrefix(line, "EXPAND_HINT="):
			f.Storage.ExpandHint = strings.TrimPrefix(line, "EXPAND_HINT=") == "1"
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
		case strings.HasPrefix(line, "SSID "):
			// SSID section device mode disabled ssid...
			parts := strings.Fields(line)
			if len(parts) >= 6 {
				f.SSIDs = append(f.SSIDs, edge.WiFiSSIDFact{
					Section:  parts[1],
					Device:   parts[2],
					Mode:     parts[3],
					Disabled: parts[4] == "1",
					SSID:     strings.Join(parts[5:], " "),
				})
			}
		}
	}
	return f
}

// enrichUSBFromBlockInfo parses "block info" lines into USBDiskFact fields.
func enrichUSBFromBlockInfo(f *edge.DeviceFacts, bi string) {
	dev := strings.Split(bi, ":")[0]
	dev = strings.TrimSpace(dev)
	base := strings.TrimPrefix(dev, "/dev/")
	if base == "" || base == dev {
		return
	}
	// disk name: sda1 -> sda, mmcblk0p1 -> mmcblk0
	disk := base
	if i := strings.LastIndexAny(base, "0123456789"); i > 0 {
		// strip trailing partition number; keep mmcblk0 from mmcblk0p1
		if strings.Contains(base, "mmcblk") && strings.Contains(base, "p") {
			disk = base[:strings.LastIndex(base, "p")]
		} else {
			j := i
			for j >= 0 && base[j] >= '0' && base[j] <= '9' {
				j--
			}
			disk = base[:j+1]
		}
	}
	var uuid, fs, mount, label string
	for _, part := range strings.Fields(bi) {
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		val = strings.Trim(val, "\"")
		switch key {
		case "UUID":
			uuid = val
		case "TYPE":
			fs = val
		case "MOUNT":
			mount = val
		case "LABEL":
			label = val
		}
	}
	for i := range f.Storage.USBDisks {
		n := f.Storage.USBDisks[i].Name
		if n != disk && !strings.HasPrefix(base, n) {
			continue
		}
		if uuid != "" {
			f.Storage.USBDisks[i].UUID = uuid
		}
		if fs != "" {
			f.Storage.USBDisks[i].FS = fs
		}
		if mount != "" {
			f.Storage.USBDisks[i].Mount = mount
		}
		if label != "" {
			f.Storage.USBDisks[i].Label = label
		}
		return
	}
}

func atoi64(s string) int64 {
	n, _ := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return n
}

// FormatPlanHuman prints deploy plan for CLI dry-run.
func FormatPlanHuman(preset string, facts edge.DeviceFacts, plan edge.DeployPlan) string {
	var b strings.Builder
	fmt.Fprintf(&b, "preset: %s (suggested: %s)\n", plan.Preset, edge.SuggestPreset(facts))
	model := facts.Model
	if model == "" {
		model = facts.Board
	}
	fmt.Fprintf(&b, "device: model=%s serial=%s arch=%s os=%s\n", model, facts.Serial, facts.Arch, facts.OS)
	fmt.Fprintf(&b, "facts: wan_capable=%v eth=%d radios=%v lan=%s/%s mem_avail_kb=%d mmc=%v\n",
		facts.WANCapable(), facts.EthernetCount(), facts.Bands(), facts.UCI.LANIP, facts.UCI.LANProto,
		facts.MemAvailKB, facts.Storage.HasMMC)
	if len(facts.SSIDs) > 0 {
		fmt.Fprintf(&b, "wifi_now:")
		for _, s := range facts.SSIDs {
			fmt.Fprintf(&b, " %s", s.SSID)
		}
		fmt.Fprintln(&b)
	}
	for _, s := range plan.Steps {
		fmt.Fprintf(&b, "  %-16s %-5s %s\n", s.Module, s.Action, s.Reason)
	}
	return b.String()
}
