package main

import (
	"os"
	"os/exec"
	"sort"
	"strings"
)

// wifiDevice is an OpenWrt UCI wifi-device (radio*).
type wifiDevice struct {
	Name string // e.g. radio0
	Band string // 2g | 5g | 6g | unknown
}

// detectWifiDevices lists wifi-device sections from `uci show wireless`.
// Board-agnostic (not Cudy-specific): any OpenWrt UCI wireless layout.
func detectWifiDevices() []wifiDevice {
	out, err := exec.Command("uci", "show", "wireless").Output()
	if err != nil {
		return nil
	}
	type meta struct {
		isDev bool
		band  string
	}
	m := map[string]*meta{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "wireless.") {
			continue
		}
		rest := strings.TrimPrefix(line, "wireless.")
		eq := strings.IndexByte(rest, '=')
		if eq <= 0 {
			continue
		}
		left, right := rest[:eq], strings.Trim(rest[eq+1:], "'\"")
		if !strings.Contains(left, ".") {
			if right == "wifi-device" {
				if m[left] == nil {
					m[left] = &meta{}
				}
				m[left].isDev = true
			}
			continue
		}
		sec, key, ok := strings.Cut(left, ".")
		if !ok {
			continue
		}
		if m[sec] == nil {
			m[sec] = &meta{}
		}
		switch key {
		case "band":
			m[sec].band = normalizeBand(right)
		case "hwmode":
			if m[sec].band == "" {
				m[sec].band = bandFromHwmode(right)
			}
		}
	}
	var list []wifiDevice
	for name, meta := range m {
		if !meta.isDev {
			continue
		}
		b := meta.band
		if b == "" {
			b = "unknown"
		}
		list = append(list, wifiDevice{Name: name, Band: b})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list
}

func normalizeBand(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "2g", "2.4g", "2.4", "24g":
		return "2g"
	case "5g", "5":
		return "5g"
	case "6g", "6":
		return "6g"
	default:
		return s
	}
}

func bandFromHwmode(hw string) string {
	hw = strings.ToLower(hw)
	switch {
	case strings.Contains(hw, "ac"):
		return "5g"
	case strings.Contains(hw, "a") && !strings.Contains(hw, "g"):
		return "5g"
	case strings.Contains(hw, "g"), strings.Contains(hw, "b"):
		return "2g"
	default:
		return "unknown"
	}
}

// guestRadioTargets picks radios for guest AP sections.
// Prefer one 2.4 + one 5 GHz when band is known; else all wifi-device.
// Override: NETDUCTOR_GUEST_RADIOS=radio0,radio1
func guestRadioTargets() []wifiDevice {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_GUEST_RADIOS")); v != "" {
		var out []wifiDevice
		for _, p := range strings.Split(v, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			out = append(out, wifiDevice{Name: p, Band: "unknown"})
		}
		if len(out) > 0 {
			return out
		}
	}
	devs := detectWifiDevices()
	if len(devs) == 0 {
		return []wifiDevice{{Name: "radio0", Band: "unknown"}}
	}
	var g2, g5 []wifiDevice
	for _, d := range devs {
		switch d.Band {
		case "2g":
			g2 = append(g2, d)
		case "5g":
			g5 = append(g5, d)
		}
	}
	var out []wifiDevice
	if len(g2) > 0 {
		out = append(out, g2[0])
	}
	if len(g5) > 0 {
		out = append(out, g5[0])
	}
	if len(out) > 0 {
		return out
	}
	return devs
}
