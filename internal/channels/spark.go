package channels

import (
	"math"
	"strings"
	"time"
)

// Sparkline renders up to n points as unicode blocks (TG/CLI safe).
func Sparkline(vals []float64) string {
	if len(vals) == 0 {
		return ""
	}
	bars := []rune("▁▂▃▄▅▆▇█")
	minV, maxV := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	span := maxV - minV
	if span < 1e-9 {
		span = 1
	}
	var b strings.Builder
	b.Grow(len(vals) * 3)
	for _, v := range vals {
		idx := int(math.Round((v - minV) / span * float64(len(bars)-1)))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		b.WriteRune(bars[idx])
	}
	return b.String()
}

// HistorySeries extracts one numeric series from sample history for sparks.
// field: "rtt" | "rx" | "tx"
func HistorySeries(pts []SamplePoint, id, peer, field string, maxPts int) []float64 {
	if maxPts <= 0 {
		maxPts = 24
	}
	var out []float64
	for _, pt := range pts {
		for _, c := range pt.Channels {
			if id != "" && c.ID != id {
				continue
			}
			if peer != "" && c.Peer != peer {
				continue
			}
			var v float64
			switch field {
			case "rx":
				v = c.RxBps
			case "tx":
				v = c.TxBps
			default:
				v = c.RTTMs
			}
			if v > 0 || field == "rtt" && c.RTTMs >= 0 && (c.OK || c.RTTMs > 0) {
				if field == "rtt" && c.RTTMs <= 0 && !c.OK {
					continue
				}
				out = append(out, v)
			}
		}
	}
	if len(out) > maxPts {
		out = out[len(out)-maxPts:]
	}
	return out
}

// SparkRTT is a short helper for UI: last hour C2 RTT spark or empty.
func SparkRTT(maxPts int) string {
	since := time.Now().UTC().Add(-time.Hour)
	pts, err := ReadHistory(since, "C2")
	if err != nil || len(pts) == 0 {
		return ""
	}
	vals := HistorySeries(pts, "C2", "", "rtt", maxPts)
	return Sparkline(vals)
}
