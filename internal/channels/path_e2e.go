package channels

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

// PathE2E is end-to-end data-plane health for one secondary (not just face :443).
type PathE2E struct {
	SecondaryID   string  `json:"secondary_id"`
	Name          string  `json:"name"`
	Online        bool    `json:"online"`
	Face443       bool    `json:"face_443"` // public :443
	Face443ms     float64 `json:"face_443_ms,omitempty"`
	UplinkOK      bool    `json:"uplink_ok"` // agent-reported primary path
	SingBoxOK     bool    `json:"singbox_ok"`
	SSH52222      bool    `json:"ssh_52222"` // out-of-band management
	SSHms         float64 `json:"ssh_ms,omitempty"`
	ICMP          bool    `json:"icmp_ok"`
	PathOK        bool    `json:"path_ok"` // online && uplink && sing-box (data plane)
	MgmtOK        bool    `json:"mgmt_ok"` // ssh or icmp — host reachable
	Note          string  `json:"note,omitempty"`
}

// CollectPathE2E probes management plane when secondary looks dead and summarises path.
func CollectPathE2E() []PathE2E {
	var out []PathE2E
	for _, d := range secondary.List() {
		sp := PathE2E{
			SecondaryID: d.ID, Name: d.Name,
			SingBoxOK: d.SingBoxOK, UplinkOK: d.UplinkOK,
		}
		if sp.Name == "" {
			sp.Name = d.ID
		}
		if !d.LastSeen.IsZero() {
			sp.Online = secondary.Online(d, 3*time.Minute)
		}
		ip := strings.TrimSpace(d.PublicIP)
		if ip != "" {
			ok, ms := tcpProbe(ip, 443, 5*time.Second)
			sp.Face443, sp.Face443ms = ok, ms
			sok, sms := tcpProbe(ip, 52222, 5*time.Second)
			sp.SSH52222, sp.SSHms = sok, sms
			sp.ICMP = icmpProbe(ip, 3*time.Second)
		}
		sp.PathOK = sp.Online && sp.UplinkOK && sp.SingBoxOK
		sp.MgmtOK = sp.SSH52222 || sp.ICMP
		switch {
		case sp.PathOK && sp.Face443:
			sp.Note = "ok"
		case !sp.Online && !sp.MgmtOK:
			sp.Note = "host unreachable (no hb, no ssh, no icmp) — use hoster console"
		case !sp.Online && sp.MgmtOK:
			sp.Note = "host up but agent heartbeat stale — check agent/sing-box on secondary"
		case sp.Online && !sp.UplinkOK:
			sp.Note = "agent online but uplink to primary broken"
		case sp.Online && !sp.Face443:
			sp.Note = "uplink ok but public :443 face down (firewall/sing-box listen?)"
		default:
			sp.Note = "degraded"
		}
		out = append(out, sp)
	}
	return out
}

func icmpProbe(host string, timeout time.Duration) bool {
	// best-effort; may require cap_net_raw
	sec := int(timeout.Seconds())
	if sec < 1 {
		sec = 1
	}
	cmd := exec.Command("ping", "-c", "1", "-W", strconv.Itoa(sec), host)
	return cmd.Run() == nil
}

// FormatPathE2E for CLI.
func FormatPathE2E(list []PathE2E) string {
	var b strings.Builder
	b.WriteString("path e2e\n")
	if len(list) == 0 {
		b.WriteString("  (no secondaries)\n")
		return b.String()
	}
	for _, p := range list {
		b.WriteString(fmt.Sprintf("  %s path=%v mgmt=%v online=%v uplink=%v sb=%v face443=%v ssh=%v icmp=%v — %s\n",
			p.Name, p.PathOK, p.MgmtOK, p.Online, p.UplinkOK, p.SingBoxOK, p.Face443, p.SSH52222, p.ICMP, p.Note))
	}
	return b.String()
}

// DialRef used by tests.
var _ = net.DialTimeout
