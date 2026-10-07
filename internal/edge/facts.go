package edge

import (
	"encoding/json"
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
	WANPresent bool   `json:"wan_present"`
	WANProto   string `json:"wan_proto,omitempty"`
}

// DeviceFacts is the capability snapshot used to build a deploy plan.
type DeviceFacts struct {
	Arch    string      `json:"arch,omitempty"`
	Board   string      `json:"board,omitempty"`
	OS      string      `json:"os,omitempty"` // openwrt | other
	Ifaces  []NetIface  `json:"ifaces,omitempty"`
	Radios  []RadioFact `json:"radios,omitempty"`
	UCI     UCINetFacts `json:"uci"`
	FlashMB int         `json:"flash_mb,omitempty"`
}

func (f DeviceFacts) EthernetCount() int {
	n := 0
	for _, i := range f.Ifaces {
		if i.Type == "ethernet" {
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
	// Heuristic: two or more ethernet ports often means lan+wan class hardware.
	return f.EthernetCount() >= 2
}

func FactsToJSON(f DeviceFacts) string {
	b, err := json.Marshal(f)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func FactsFromJSON(s string) (DeviceFacts, error) {
	var f DeviceFacts
	if err := json.Unmarshal([]byte(s), &f); err != nil {
		return f, err
	}
	return f, nil
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
