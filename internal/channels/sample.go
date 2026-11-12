package channels

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

// ChannelSample is one point-in-time observation for a segment.
type ChannelSample struct {
	ID    string  `json:"id"` // C2, C4, …
	Peer  string  `json:"peer,omitempty"`
	OK    bool    `json:"ok"`
	RTTMs float64 `json:"rtt_ms,omitempty"`
	RxBps float64 `json:"rx_bps,omitempty"`
	TxBps float64 `json:"tx_bps,omitempty"`
	Note  string  `json:"note,omitempty"`
}

// SamplePoint is a history record.
type SamplePoint struct {
	TS       time.Time       `json:"ts"`
	Node     string          `json:"node"`
	Channels []ChannelSample `json:"channels"`
}

// AlertDefaults from PLAN-CHANNEL-TELEMETRY (starting values).
const (
	DefaultSampleInterval = 20 * time.Second
	DefaultDownSamples    = 2
	DefaultRTTWarnMs      = 150.0
	DefaultRTTCritMs      = 300.0
	DefaultRateRatioWarn  = 0.2
	DefaultRateRatioCrit  = 0.1
	DefaultHistoryRetain  = 24 * time.Hour
)

func historyPath() string {
	return filepath.Join(paths.StateDir(), "metrics", "channel-history.jsonl")
}

// SamplePrimary collects C2–C6-ish signals available on primary without agent changes.
// C2: secondary uplink_ok + face TCP RTT from primary→S:443
// C4/C5: best-effort ICMP/TCP to service-plane VIP if present in peers material
func SamplePrimary() SamplePoint {
	pt := SamplePoint{TS: time.Now().UTC(), Node: "primary"}
	for _, d := range secondary.List() {
		peer := d.ID
		if peer == "" {
			peer = d.Name
		}
		// C2: uplink boolean from agent + face RTT
		c2 := ChannelSample{ID: "C2", Peer: peer, OK: d.UplinkOK && secondary.Online(d, 3*time.Minute)}
		if ip := strings.TrimSpace(d.PublicIP); ip != "" {
			ok, ms := tcpProbe(ip, 443, 3*time.Second)
			c2.RTTMs = ms
			if !ok {
				c2.Note = "face443_fail"
			}
			// face alone is C3-ish from primary view of S
			c3 := ChannelSample{ID: "C3", Peer: peer, OK: ok, RTTMs: ms}
			if !ok {
				c3.Note = "tcp443_fail"
			}
			pt.Channels = append(pt.Channels, c3)
		}
		if !d.UplinkOK {
			c2.Note = "uplink_down"
		}
		pt.Channels = append(pt.Channels, c2)
	}
	// Host default-route iface rates (C7 egress baseline)
	if rx, tx, iface, ok := defaultRouteRates(); ok {
		pt.Channels = append(pt.Channels, ChannelSample{
			ID: "C7", OK: true, RxBps: rx, TxBps: tx, Note: "iface=" + iface,
		})
	}
	return pt
}

// AppendHistory writes one sample to jsonl and prunes old lines.
func AppendHistory(pt SamplePoint) error {
	dir := filepath.Dir(historyPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(historyPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	if err := enc.Encode(pt); err != nil {
		return err
	}
	_ = pruneHistory(DefaultHistoryRetain)
	return nil
}

func pruneHistory(retain time.Duration) error {
	path := historyPath()
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	cutoff := time.Now().UTC().Add(-retain)
	var keep []string
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := sc.Text()
		var pt SamplePoint
		if json.Unmarshal([]byte(line), &pt) != nil {
			continue
		}
		if pt.TS.After(cutoff) {
			keep = append(keep, line)
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(strings.Join(keep, "\n")+"\n"), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// ReadHistory returns samples newer than since (or all if since is zero).
func ReadHistory(since time.Time, idFilter string) ([]SamplePoint, error) {
	b, err := os.ReadFile(historyPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []SamplePoint
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	// larger lines possible
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for sc.Scan() {
		var pt SamplePoint
		if json.Unmarshal(sc.Bytes(), &pt) != nil {
			continue
		}
		if !since.IsZero() && pt.TS.Before(since) {
			continue
		}
		if idFilter != "" {
			var filtered []ChannelSample
			for _, c := range pt.Channels {
				if c.ID == idFilter {
					filtered = append(filtered, c)
				}
			}
			pt.Channels = filtered
			if len(pt.Channels) == 0 {
				continue
			}
		}
		out = append(out, pt)
	}
	return out, nil
}

// FormatHistoryText for CLI.
func FormatHistoryText(pts []SamplePoint) string {
	var b strings.Builder
	if len(pts) == 0 {
		b.WriteString("channel history: (empty)\n")
		return b.String()
	}
	for _, pt := range pts {
		b.WriteString(pt.TS.Format(time.RFC3339))
		b.WriteString(" node=")
		b.WriteString(pt.Node)
		b.WriteByte('\n')
		for _, c := range pt.Channels {
			b.WriteString(fmt.Sprintf("  %s peer=%s ok=%v rtt=%.1fms rx=%.0f tx=%.0f %s\n",
				c.ID, c.Peer, c.OK, c.RTTMs, c.RxBps, c.TxBps, c.Note))
		}
	}
	return b.String()
}

// --- iface rate via /proc/net/dev deltas ---

var (
	ifaceMu   sync.Mutex
	ifaceLast = map[string]struct {
		rx, tx uint64
		at     time.Time
	}{}
)

func readIfaceBytes(name string) (rx, tx uint64, ok bool) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, name+":") && !strings.Contains(line, name+":") {
			continue
		}
		// name: rx ... tx ...
		parts := strings.Fields(line)
		if len(parts) < 10 {
			continue
		}
		// parts[0] is "eth0:" or similar
		rx, _ = strconv.ParseUint(parts[1], 10, 64)
		tx, _ = strconv.ParseUint(parts[9], 10, 64)
		return rx, tx, true
	}
	return 0, 0, false
}

func ifaceRateBps(name string) (rxBps, txBps float64, ok bool) {
	rx, tx, ok := readIfaceBytes(name)
	if !ok {
		return 0, 0, false
	}
	now := time.Now()
	ifaceMu.Lock()
	defer ifaceMu.Unlock()
	prev, had := ifaceLast[name]
	ifaceLast[name] = struct {
		rx, tx uint64
		at     time.Time
	}{rx, tx, now}
	if !had || now.Sub(prev.at) < time.Second {
		return 0, 0, true // first sample: rates unknown
	}
	dt := now.Sub(prev.at).Seconds()
	if dt <= 0 {
		return 0, 0, true
	}
	var drx, dtx float64
	if rx >= prev.rx {
		drx = float64(rx - prev.rx)
	}
	if tx >= prev.tx {
		dtx = float64(tx - prev.tx)
	}
	return drx / dt * 8, dtx / dt * 8, true
}

func defaultRouteIface() string {
	b, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == "00000000" {
			return fields[0]
		}
	}
	return ""
}

func defaultRouteRates() (rx, tx float64, iface string, ok bool) {
	iface = defaultRouteIface()
	if iface == "" {
		return 0, 0, "", false
	}
	rx, tx, ok = ifaceRateBps(iface)
	return rx, tx, iface, ok
}

// SampleOnce is a convenience for CLI/timer: sample + append.
func SampleOnce() (SamplePoint, error) {
	pt := SamplePrimary()
	return pt, AppendHistory(pt)
}

// Ensure tcpProbe is available (defined in health.go).
var _ = net.DialTimeout
