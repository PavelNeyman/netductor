package channels

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

// ProbeResult is on-demand segment check (T6).
type ProbeResult struct {
	TS       time.Time `json:"ts"`
	Target   string    `json:"target"`
	Port     int       `json:"port"`
	OK       bool      `json:"ok"`
	RTTMs    float64   `json:"rtt_ms"`
	BWRxMbps float64   `json:"bw_rx_mbps,omitempty"` // passive iface sample if --bw
	BWTxMbps float64   `json:"bw_tx_mbps,omitempty"`
	Note     string    `json:"note,omitempty"`
}

// ProbeTCP dials host:port once (face check).
func ProbeTCP(host string, port int, timeout time.Duration) ProbeResult {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	if port <= 0 {
		port = 443
	}
	r := ProbeResult{TS: time.Now().UTC(), Target: host, Port: port}
	ok, ms := tcpProbe(host, port, timeout)
	r.OK, r.RTTMs = ok, ms
	if !ok {
		r.Note = "tcp_fail"
	}
	return r
}

// ProbeBW samples default-route iface rates over window (passive; no sustained load).
// Optional target still gets a TCP face probe.
func ProbeBW(host string, port int, window time.Duration) ProbeResult {
	if window < time.Second {
		window = 2 * time.Second
	}
	r := ProbeResult{TS: time.Now().UTC(), Target: host, Port: port, Note: "passive_iface"}
	if host != "" {
		ok, ms := tcpProbe(host, port, 3*time.Second)
		r.OK, r.RTTMs = ok, ms
		if !ok {
			r.Note = "tcp_fail+passive_iface"
		}
	} else {
		r.OK = true
	}
	// Prime counters
	_, _, _, _ = defaultRouteRates()
	time.Sleep(window)
	rx, tx, iface, ok := defaultRouteRates()
	if ok && iface != "" {
		r.BWRxMbps = rx / 1e6
		r.BWTxMbps = tx / 1e6
		if r.Note == "passive_iface" || strings.Contains(r.Note, "passive") {
			r.Note = "passive_iface=" + iface
		} else {
			r.Note += ";iface=" + iface
		}
	}
	return r
}

// ProbeSecondaries runs face TCP (+ optional BW) for each online secondary public IP.
func ProbeSecondaries(withBW bool) []ProbeResult {
	var out []ProbeResult
	for _, d := range secondary.List() {
		ip := strings.TrimSpace(d.PublicIP)
		if ip == "" {
			continue
		}
		var r ProbeResult
		if withBW {
			r = ProbeBW(ip, 443, 2*time.Second)
		} else {
			r = ProbeTCP(ip, 443, 3*time.Second)
		}
		name := d.Name
		if name == "" {
			name = d.ID
		}
		r.Note = strings.TrimSpace(name + " " + r.Note)
		out = append(out, r)
	}
	return out
}

// FormatProbeText for CLI.
func FormatProbeText(list []ProbeResult) string {
	var b strings.Builder
	if len(list) == 0 {
		return "channel probe: (no targets)\n"
	}
	for _, r := range list {
		b.WriteString(fmt.Sprintf("%s:%d ok=%v rtt=%.0fms", r.Target, r.Port, r.OK, r.RTTMs))
		if r.BWRxMbps > 0 || r.BWTxMbps > 0 {
			b.WriteString(fmt.Sprintf(" rx=%.2fMbps tx=%.2fMbps", r.BWRxMbps, r.BWTxMbps))
		}
		if r.Note != "" {
			b.WriteString(" " + r.Note)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// ParseProbeTarget accepts host or host:port.
func ParseProbeTarget(s string) (host string, port int) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 443
	}
	if h, p, err := net.SplitHostPort(s); err == nil {
		port, _ = strconv.Atoi(p)
		return h, port
	}
	return s, 443
}
