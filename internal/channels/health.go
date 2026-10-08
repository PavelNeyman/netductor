// Package channels reports dual-node path health for collect / doctor / CLI.
package channels

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

// SecondaryPath is one enrolled secondary's control + data path view from primary.
type SecondaryPath struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	PublicIP        string  `json:"public_ip"`
	Online          bool    `json:"online"`
	HeartbeatAgeSec int     `json:"heartbeat_age_sec"`
	SingBoxOK       bool    `json:"singbox_ok"`
	UplinkOK        bool    `json:"uplink_ok"` // agent: primary:443 reachable from secondary
	TCP443OK        bool    `json:"tcp443_ok"` // primary → secondary public:443 (face only; uplink is data-plane)
	TCP443ms        float64 `json:"tcp443_ms,omitempty"`
	Mismatch30m     int     `json:"mismatch_30m"` // as reported by agent heartbeat
}

// Report is primary-side channel snapshot.
type Report struct {
	TS                         time.Time       `json:"ts"`
	Secondaries                []SecondaryPath `json:"secondaries"`
	MismatchLocal30m           int             `json:"mismatch_local_30m"` // primary journal flow mismatch
	RealityInvalidFromSec15m   int             `json:"reality_invalid_from_secondary_15m"`
	RealityInvalidTotal15m     int             `json:"reality_invalid_total_15m"`
	Notes                      []string        `json:"notes,omitempty"`
}

// Collect builds a report (primary). Safe on secondary (empty list).
func Collect() Report {
	rep := Report{TS: time.Now().UTC()}
	mm := vpn.CollectMismatch(30)
	rep.MismatchLocal30m = mm.Total

	invTotal, invFromSec := countRealityInvalid(15)
	rep.RealityInvalidTotal15m = invTotal
	rep.RealityInvalidFromSec15m = invFromSec

	secIPs := map[string]bool{}
	for _, d := range secondary.List() {
		if d.PublicIP != "" {
			secIPs[d.PublicIP] = true
		}
		sp := SecondaryPath{
			ID: d.ID, Name: d.Name, PublicIP: d.PublicIP,
			SingBoxOK: d.SingBoxOK, UplinkOK: d.UplinkOK,
			Mismatch30m: d.MismatchTotal,
		}
		if sp.Name == "" {
			sp.Name = d.ID
		}
		if !d.LastSeen.IsZero() {
			sp.HeartbeatAgeSec = int(time.Since(d.LastSeen).Seconds())
			sp.Online = secondary.Online(d, 3*time.Minute)
		}
		if d.PublicIP != "" {
			ok, ms := tcpProbe(d.PublicIP, 443, 5*time.Second)
			sp.TCP443OK, sp.TCP443ms = ok, ms
		}
		rep.Secondaries = append(rep.Secondaries, sp)
	}
	_ = secIPs
	return rep
}

func tcpProbe(host string, port int, timeout time.Duration) (bool, float64) {
	t0 := time.Now()
	c, err := net.DialTimeout("tcp", net.JoinHostPort(host, strconv.Itoa(port)), timeout)
	ms := float64(time.Since(t0).Milliseconds())
	if err != nil {
		return false, ms
	}
	_ = c.Close()
	return true, ms
}

// countRealityInvalid scans sing-box journal; returns total invalid and those from secondary public IPs.
func countRealityInvalid(windowMin int) (total, fromSecondary int) {
	if windowMin < 1 {
		windowMin = 15
	}
	out, err := exec.Command("journalctl", "-u", "sing-box",
		"--since", fmt.Sprintf("%d min ago", windowMin),
		"-o", "cat", "--no-pager").CombinedOutput()
	if err != nil {
		return 0, 0
	}
	secIPs := map[string]bool{}
	for _, d := range secondary.List() {
		if d.PublicIP != "" {
			secIPs[d.PublicIP] = true
		}
	}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "REALITY: processed invalid") {
			continue
		}
		total++
		// "from 92.255.77.253:55304"
		if i := strings.Index(line, "from "); i >= 0 {
			rest := line[i+5:]
			ip := rest
			if j := strings.IndexAny(rest, ": "); j > 0 {
				ip = rest[:j]
			}
			if secIPs[ip] {
				fromSecondary++
			}
		}
	}
	return total, fromSecondary
}

// FormatText human-readable for CLI/doctor.
func FormatText(r Report) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("channel health ts=%s\n", r.TS.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("mismatch_local_30m=%d reality_invalid_15m=%d (from_secondary=%d)\n",
		r.MismatchLocal30m, r.RealityInvalidTotal15m, r.RealityInvalidFromSec15m))
	if len(r.Secondaries) == 0 {
		b.WriteString("secondaries: (none)\n")
		return b.String()
	}
	for _, s := range r.Secondaries {
		b.WriteString(fmt.Sprintf("  %s ip=%s online=%v hb_age=%ds sb=%v uplink=%v tcp443=%v (%.0fms) mismatch30m=%d\n",
			s.Name, s.PublicIP, s.Online, s.HeartbeatAgeSec, s.SingBoxOK, s.UplinkOK, s.TCP443OK, s.TCP443ms, s.Mismatch30m))
	}
	return b.String()
}
