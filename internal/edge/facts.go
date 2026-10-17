package edge

import (
	"strings"
)

// NetIface is a probed host network interface (deploy-time or agent).
type NetIface struct {
	Name string `json:"name"`
	Type string `json:"type"` // ethernet | wireless | other
	Up   bool   `json:"up"`
}

// RadioFact is a wifi-device style radio.
type RadioFact struct {
	Name string `json:"name"`
	Band string `json:"band,omitempty"` // 2g | 5g | 6g | unknown
}

// UCINetFacts summarizes OpenWrt network UCI without requiring uplink.
type UCINetFacts struct {
	LANDevice  string `json:"lan_device,omitempty"`
	LANIP      string `json:"lan_ip,omitempty"`
	LANProto   string `json:"lan_proto,omitempty"`
	WANPresent bool   `json:"wan_present"`
	WANProto   string `json:"wan_proto,omitempty"`
	WANIP      string `json:"wan_ip,omitempty"`
}

// WiFiSSIDFact is a configured AP iface (current state).
type WiFiSSIDFact struct {
	Section  string `json:"section,omitempty"` // e.g. default_radio0
	Device   string `json:"device,omitempty"`  // radio0
	SSID     string `json:"ssid,omitempty"`
	Mode     string `json:"mode,omitempty"` // ap | sta
	Disabled bool   `json:"disabled,omitempty"`
}

// USBDiskFact is one USB/block disk seen at probe (PLAN-EDGE-USB-STORAGE).
type USBDiskFact struct {
	Name       string `json:"name"` // sda
	SizeKB     int64  `json:"size_kb,omitempty"`
	Partitions int    `json:"partitions,omitempty"`
	FS         string `json:"fs,omitempty"` // first partition fs if any
	Mount      string `json:"mount,omitempty"`
	FreeKB     int64  `json:"free_kb,omitempty"`
	UsedKB     int64  `json:"used_kb,omitempty"`
	UUID       string `json:"uuid,omitempty"`
	Label      string `json:"label,omitempty"`
}

// StorageFacts best-effort disk / overlay signals for UI offers.
type StorageFacts struct {
	OverlayTotalKB int64         `json:"overlay_total_kb,omitempty"`
	OverlayFreeKB  int64         `json:"overlay_free_kb,omitempty"`
	RootTotalKB    int64         `json:"root_total_kb,omitempty"`
	RootFreeKB     int64         `json:"root_free_kb,omitempty"`
	FlashKB        int64         `json:"flash_kb,omitempty"` // from /proc/mtd when present
	HasMMC         bool          `json:"has_mmc,omitempty"`
	HasUSBDisk     bool          `json:"has_usb_disk,omitempty"`
	USBDisks       []USBDiskFact `json:"usb_disks,omitempty"`
	ExtrootActive  bool          `json:"extroot_active,omitempty"` // /overlay already on external
	ExpectUSB      bool          `json:"expect_usb,omitempty"`
	ExpandHint     bool          `json:"expand_hint,omitempty"`
	OverlayNote    string        `json:"overlay_note,omitempty"`
}

// HasUsableUSB is true when a USB mass-storage disk is present (plan gate for vpn/extroot).
func (s StorageFacts) HasUsableUSB() bool {
	return s.HasUSBDisk || len(s.USBDisks) > 0
}

// DeviceFacts is the capability snapshot used to build a deploy plan and UI card.
type DeviceFacts struct {
	Arch       string         `json:"arch,omitempty"`
	Board      string         `json:"board,omitempty"`
	Model      string         `json:"model,omitempty"`
	Serial     string         `json:"serial,omitempty"`
	OS         string         `json:"os,omitempty"` // openwrt | other
	Ifaces     []NetIface     `json:"ifaces,omitempty"`
	Radios     []RadioFact    `json:"radios,omitempty"`
	SSIDs      []WiFiSSIDFact `json:"ssids,omitempty"`
	UCI        UCINetFacts    `json:"uci"`
	MemTotalKB int64          `json:"mem_total_kb,omitempty"`
	MemAvailKB int64          `json:"mem_avail_kb,omitempty"`
	Storage    StorageFacts   `json:"storage"`
	FlashMB    int            `json:"flash_mb,omitempty"`
}

func (f DeviceFacts) EthernetCount() int {
	n := 0
	for _, i := range f.Ifaces {
		name := strings.ToLower(strings.TrimSpace(i.Name))
		if name == "" || strings.Contains(name, ".") || strings.Contains(name, "@") {
			continue
		}
		// usb tether is type=usb; ignore non-ethernet
		if i.Type != "ethernet" {
			continue
		}
		switch {
		case strings.HasPrefix(name, "eth"):
			n++
		case name == "wan" || strings.HasPrefix(name, "wan"):
			n++
		case strings.HasPrefix(name, "lan"): // lan, lan1, lan2 (DSA)
			n++
		}
	}
	return n
}

func (f DeviceFacts) HasRadios() bool {
	return len(f.Radios) > 0
}

func (f DeviceFacts) WANCapable() bool {
	if f.UCI.WANPresent {
		return true
	}
	return f.EthernetCount() >= 2
}

func (f DeviceFacts) Bands() []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range f.Radios {
		b := strings.ToLower(r.Band)
		if b == "" {
			b = "unknown"
		}
		if !seen[b] {
			seen[b] = true
			out = append(out, b)
		}
	}
	return out
}

// NormalizeArch maps uname -m style to netductor agent arch labels.
func NormalizeArch(unameM string) string {
	u := strings.ToLower(strings.TrimSpace(unameM))
	switch u {
	case "x86_64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	case "armv7l", "armv6l", "arm":
		return "arm"
	case "mips", "mipsel":
		return "mipsle"
	case "riscv64":
		return "riscv64"
	default:
		return u
	}
}

// ModuleSelection is the UI/operator choice of which modules to run.
type ModuleSelection struct {
	Advanced bool            `json:"advanced"` // unlock all modules despite facts
	Enabled  map[string]bool `json:"enabled"`  // module id → on/off
}

// ApplySelection filters a plan: disabled modules become skip; Advanced can force-apply wan/wifi/etc.
func ApplySelection(plan DeployPlan, sel ModuleSelection, facts DeviceFacts) DeployPlan {
	if sel.Enabled == nil {
		return plan
	}
	var steps []PlanStep
	for _, s := range plan.Steps {
		en, ok := sel.Enabled[s.Module]
		if ok && !en {
			s.Action = "skip"
			s.Reason = "disabled by operator"
			steps = append(steps, s)
			continue
		}
		if ok && en && s.Action == "skip" {
			// VPN without USB: only Advanced may force (operator accepts likely ENOSPC).
			if s.Module == ModVPNClient && !facts.Storage.HasUsableUSB() && !facts.Storage.ExtrootActive && !sel.Advanced {
				s.Action = "skip"
				s.Reason = "no USB — vpn_client disabled (PLAN-EDGE-USB-STORAGE)"
				steps = append(steps, s)
				continue
			}
			if sel.Advanced || s.Module == ModFSExpand || s.Module == ModOverlay {
				s.Action = "apply"
				if sel.Advanced {
					s.Reason = "forced (advanced)"
				} else {
					s.Reason = "enabled by operator"
				}
				steps = append(steps, s)
				continue
			}
		}
		steps = append(steps, s)
	}
	plan.Steps = steps
	return plan
}
