package nvr

import (
	"bufio"
	"strconv"
	"strings"
	"time"
)

// Lease is one DHCP lease entry (OpenWrt /tmp/dhcp.leases style).
type Lease struct {
	Expiry   int64  `json:"expiry"`
	MAC      string `json:"mac"`
	IP       string `json:"ip"`
	Hostname string `json:"hostname,omitempty"`
	ClientID string `json:"client_id,omitempty"`
}

// ParseDHCPLeases parses OpenWrt dhcp.leases content.
// Format: <expiry> <mac> <ip> <hostname> <clientid>
func ParseDHCPLeases(content string) []Lease {
	var out []Lease
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		exp, _ := strconv.ParseInt(fields[0], 10, 64)
		l := Lease{
			Expiry: exp,
			MAC:    normalizeMAC(fields[1]),
			IP:     fields[2],
		}
		if len(fields) >= 4 && fields[3] != "*" {
			l.Hostname = fields[3]
		}
		if len(fields) >= 5 && fields[4] != "*" {
			l.ClientID = fields[4]
		}
		out = append(out, l)
	}
	return out
}

// LeaseRemainingSec returns seconds until expiry (0 if unknown/expired).
func LeaseRemainingSec(l Lease, now time.Time) int64 {
	if l.Expiry <= 0 {
		return 0
	}
	d := l.Expiry - now.Unix()
	if d < 0 {
		return 0
	}
	return d
}
