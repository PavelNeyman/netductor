package main

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// collectDeviceFacts builds a JSON-friendly map matching edge.DeviceFacts for heartbeat.
// Best-effort; never fails the agent tick.
func collectDeviceFacts() map[string]any {
	f := map[string]any{
		"arch": runtime.GOARCH,
		"os":   "openwrt",
	}
	if b, err := os.ReadFile("/tmp/sysinfo/board_name"); err == nil {
		f["board"] = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile("/tmp/sysinfo/model"); err == nil {
		f["model"] = strings.TrimSpace(string(b))
	}
	ser := ""
	if out, err := exec.Command("sh", "-c", "awk -F': ' '/^Serial/{print $2; exit}' /proc/cpuinfo").Output(); err == nil {
		ser = strings.TrimSpace(string(out))
	}
	if ser == "" || ser == "0000000000000000" {
		if b, err := os.ReadFile("/sys/class/net/eth0/address"); err == nil {
			ser = strings.TrimSpace(string(b))
		}
	}
	if ser != "" {
		f["serial"] = ser
	}
	uci := map[string]any{}
	if s := uciGet("network.lan.device"); s != "" {
		uci["lan_device"] = s
	}
	if s := uciGet("network.lan.ipaddr"); s != "" {
		uci["lan_ip"] = s
	}
	if s := uciGet("network.lan.proto"); s != "" {
		uci["lan_proto"] = s
	}
	if s := uciGet("network.wan.proto"); s != "" {
		uci["wan_present"] = true
		uci["wan_proto"] = s
		if ip := uciGet("network.wan.ipaddr"); ip != "" {
			uci["wan_ip"] = ip
		}
	} else {
		uci["wan_present"] = false
	}
	f["uci"] = uci

	if b, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				var v int64
				fmtSscanfMem(line, &v)
				f["mem_total_kb"] = v
			}
			if strings.HasPrefix(line, "MemAvailable:") {
				var v int64
				fmtSscanfMem(line, &v)
				f["mem_avail_kb"] = v
			}
		}
	}

	storage := map[string]any{}
	if _, err := os.Stat("/dev/mmcblk0"); err == nil {
		storage["has_mmc"] = true
	}
	if matches, _ := exec.Command("sh", "-c", "ls /dev/sd[a-z] 2>/dev/null | head -1").Output(); len(strings.TrimSpace(string(matches))) > 0 {
		storage["has_usb_disk"] = true
	}
	f["storage"] = storage

	var ifaces []map[string]any
	if out, err := exec.Command("ip", "-o", "link").Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			// "2: eth0: <...>"
			parts := strings.SplitN(line, ":", 3)
			if len(parts) < 2 {
				continue
			}
			name := strings.TrimSpace(parts[1])
			switch {
			case name == "lo", strings.HasPrefix(name, "br-"), strings.HasPrefix(name, "wlan"):
				// keep wlan as wireless below
			}
			if name == "lo" || strings.HasPrefix(name, "br-") || strings.HasPrefix(name, "docker") ||
				strings.HasPrefix(name, "veth") || strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "wg") {
				continue
			}
			typ := "other"
			switch {
			case strings.HasPrefix(name, "eth"), strings.HasPrefix(name, "lan"), strings.HasPrefix(name, "wan"), strings.HasPrefix(name, "usb"):
				typ = "ethernet"
			case strings.HasPrefix(name, "wlan"), strings.HasPrefix(name, "wl"), strings.HasPrefix(name, "ra"):
				typ = "wireless"
			}
			up := strings.Contains(line, "state UP")
			ifaces = append(ifaces, map[string]any{"name": name, "type": typ, "up": up})
		}
	}
	if len(ifaces) > 0 {
		f["ifaces"] = ifaces
	}

	// radios from wifi_radios helpers if present
	var radios []map[string]any
	for _, d := range detectWifiDevices() {
		radios = append(radios, map[string]any{"name": d.Name, "band": d.Band})
	}
	if len(radios) > 0 {
		f["radios"] = radios
	}

	var ssids []map[string]any
	if out, err := exec.Command("sh", "-c", `uci -q show wireless 2>/dev/null | grep '=wifi-iface'`).Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			sec := strings.TrimPrefix(line, "wireless.")
			if i := strings.Index(sec, "="); i > 0 {
				sec = sec[:i]
			}
			ssid := uciGet("wireless." + sec + ".ssid")
			if ssid == "" {
				continue
			}
			ssids = append(ssids, map[string]any{
				"section":  sec,
				"device":   uciGet("wireless." + sec + ".device"),
				"mode":     uciGet("wireless." + sec + ".mode"),
				"disabled": uciGet("wireless."+sec+".disabled") == "1",
				"ssid":     ssid,
			})
		}
	}
	if len(ssids) > 0 {
		f["ssids"] = ssids
	}
	return f
}

func uciGet(path string) string {
	out, err := exec.Command("uci", "-q", "get", path).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func fmtSscanfMem(line string, v *int64) {
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		n, _ := strconv.ParseInt(fields[1], 10, 64)
		*v = n
	}
}
