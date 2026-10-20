package edge

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// NetworkZone is one logical segment on a site (LAN / guest / IoT).
type NetworkZone struct {
	ID      string `json:"id"` // lan | guest | iot
	Title   string `json:"title"`
	CIDR    string `json:"cidr,omitempty"`
	Enabled bool   `json:"enabled"`
	SSID    string `json:"ssid,omitempty"`
}

// SiteSummary is the UI card model (topology + metrics).
type SiteSummary struct {
	DeviceID  string        `json:"device_id"`
	Name      string        `json:"name"`
	Status    string        `json:"status"` // online | offline | draft | pending
	Kind      string        `json:"kind"`   // openwrt | mikrotik | …
	Board     string        `json:"board,omitempty"`
	WANIP     string        `json:"wan_ip,omitempty"`
	RouterIP  string        `json:"router_ip,omitempty"`
	Hostname  string        `json:"hostname,omitempty"`
	Zones     []NetworkZone `json:"zones"`
	Devices   int           `json:"devices"`
	Guests    int           `json:"guests"`
	UptimeSec float64       `json:"uptime_sec,omitempty"`
	UptimeFmt string        `json:"uptime_fmt,omitempty"`
	LatencyMs float64       `json:"latency_ms,omitempty"`
	Online    bool          `json:"online"`
	LastSeen  int64         `json:"last_seen,omitempty"`
	Agent     string        `json:"agent,omitempty"`
}

// ListSiteSummaries builds topology cards for all edge devices.
func ListSiteSummaries() []SiteSummary {
	devs := ListDevices()
	out := make([]SiteSummary, 0, len(devs))
	for _, d := range devs {
		out = append(out, BuildSiteSummary(d))
	}
	return out
}

// BuildSiteSummary derives card fields from device + overlay.
func BuildSiteSummary(d Device) SiteSummary {
	name := d.Hostname
	if name == "" {
		name = d.DeviceID
	}
	online := d.LastSeen > 0 && time.Now().Unix()-d.LastSeen < 180
	status := "draft"
	switch d.Status {
	case StatusApproved:
		if online {
			status = "online"
		} else {
			status = "offline"
		}
	case StatusPending:
		status = "pending"
	case StatusDenied, StatusRevoked:
		status = d.Status
	default:
		if d.DeviceID != "" && d.Status == "" {
			status = "draft"
		}
	}
	kind := "openwrt"
	if d.Facts != nil && d.Facts.OS != "" {
		kind = strings.ToLower(d.Facts.OS)
	}
	routerIP := overlayStr(d.Overlay, "lan_ip")
	if routerIP == "" && d.Facts != nil {
		routerIP = d.Facts.UCI.LANIP
	}
	s := SiteSummary{
		DeviceID:  d.DeviceID,
		Name:      name,
		Status:    status,
		Kind:      kind,
		Board:     d.Board,
		WANIP:     d.WANIP,
		RouterIP:  routerIP,
		Hostname:  d.Hostname,
		Zones:     zonesFromDevice(d),
		Devices:   d.DeviceCount,
		Guests:    d.GuestActive,
		UptimeSec: d.UptimeSec,
		UptimeFmt: formatUptime(d.UptimeSec),
		LatencyMs: d.LatencyMs,
		Online:    online,
		LastSeen:  d.LastSeen,
		Agent:     d.Agent,
	}
	return s
}

func zonesFromDevice(d Device) []NetworkZone {
	lanIP := overlayStr(d.Overlay, "lan_ip")
	if lanIP == "" && d.Facts != nil {
		lanIP = d.Facts.UCI.LANIP
	}
	lanCIDR := overlayStr(d.Overlay, "lan_cidr")
	if lanCIDR == "" {
		lanCIDR = cidrFromIP(lanIP, 24)
	}
	guestCIDR := overlayStr(d.Overlay, "guest_cidr")
	iotCIDR := overlayStr(d.Overlay, "iot_cidr")
	if guestCIDR == "" || iotCIDR == "" {
		g, i := deriveSiblingCIDRs(lanCIDR)
		if guestCIDR == "" {
			guestCIDR = g
		}
		if iotCIDR == "" {
			iotCIDR = i
		}
	}
	guestOn := overlayBool(d.Overlay, "guest_enabled", true)
	if gmap, ok := d.Overlay["guest"].(map[string]any); ok {
		if en, ok := gmap["enabled"].(bool); ok {
			guestOn = en
		}
	}
	iotOn := overlayBool(d.Overlay, "iot_enabled", false)
	if imap, ok := d.Overlay["iot"].(map[string]any); ok {
		if en, ok := imap["enabled"].(bool); ok {
			iotOn = en
		}
		if c, ok := imap["cidr"].(string); ok && c != "" {
			iotCIDR = c
		}
	}
	ssidGuest := overlayNestedStr(d.Overlay, "guest", "ssid")
	ssidIoT := overlayNestedStr(d.Overlay, "iot", "ssid")
	return []NetworkZone{
		{ID: "lan", Title: "LAN", CIDR: lanCIDR, Enabled: lanCIDR != ""},
		{ID: "guest", Title: "Guest Wi-Fi", CIDR: guestCIDR, Enabled: guestOn, SSID: ssidGuest},
		{ID: "iot", Title: "IoT", CIDR: iotCIDR, Enabled: iotOn, SSID: ssidIoT},
	}
}

func overlayStr(ov map[string]any, key string) string {
	if ov == nil {
		return ""
	}
	if v, ok := ov[key].(string); ok {
		return v
	}
	if net, ok := ov["network"].(map[string]any); ok {
		if v, ok := net[key].(string); ok {
			return v
		}
	}
	return ""
}

func overlayNestedStr(ov map[string]any, a, b string) string {
	if ov == nil {
		return ""
	}
	if m, ok := ov[a].(map[string]any); ok {
		if v, ok := m[b].(string); ok {
			return v
		}
	}
	return ""
}

func overlayBool(ov map[string]any, key string, def bool) bool {
	if ov == nil {
		return def
	}
	if v, ok := ov[key].(bool); ok {
		return v
	}
	return def
}

func cidrFromIP(ip string, bits int) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ""
	}
	if strings.Contains(ip, "/") {
		return ip
	}
	parsed := net.ParseIP(ip)
	if parsed == nil || parsed.To4() == nil {
		return ""
	}
	mask := net.CIDRMask(bits, 32)
	network := parsed.Mask(mask)
	return fmt.Sprintf("%s/%d", network.String(), bits)
}

// deriveSiblingCIDRs: 10.120.0.0/24 → guest 10.120.1.0/24, iot 10.120.2.0/24
func deriveSiblingCIDRs(lanCIDR string) (guest, iot string) {
	if lanCIDR == "" {
		return "10.120.1.0/24", "10.120.2.0/24"
	}
	ip, ipnet, err := net.ParseCIDR(lanCIDR)
	if err != nil {
		return "", ""
	}
	v4 := ip.To4()
	if v4 == nil {
		return "", ""
	}
	ones, _ := ipnet.Mask.Size()
	if ones != 24 {
		// keep same mask; only bump third octet for /24 shops
		return lanCIDR, lanCIDR
	}
	g := append(net.IP(nil), v4...)
	i := append(net.IP(nil), v4...)
	g[2] = byte((int(g[2]) + 1) % 256)
	i[2] = byte((int(i[2]) + 2) % 256)
	g[3], i[3] = 0, 0
	return g.String() + "/24", i.String() + "/24"
}

func formatUptime(sec float64) string {
	if sec <= 0 {
		return ""
	}
	s := int64(sec)
	d := s / 86400
	h := (s % 86400) / 3600
	m := (s % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%dd %dh %dm", d, h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
