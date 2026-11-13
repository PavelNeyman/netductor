package channels

import (
	"fmt"
	"time"
)

// Alert is a channel degradation signal for notify.AlertOnce.
type Alert struct {
	Key      string
	Msg      string
	Incident bool // attach diagnostic archive (only serious / sustained)
}

// Face TCP dial (C3) is NOT network RTT — includes accept queue / Reality.
// Uplink agent RTT (C2) is closer to path quality but still noisy.
const (
	uplinkRTTWarnMs = 500.0
	uplinkRTTCritMs = 1000.0
	faceDialWarnMs  = 2500.0
	faceDialCritMs  = 5000.0
	// Need this many recent samples (including current) above threshold before alert.
	rttConsecutive = 3
)

// EvaluateSampleAlerts checks RTT and rate collapse against plan defaults.
// History window: last hour of samples for p50-ish baseline (median of recent RTT/rates).
func EvaluateSampleAlerts(pt SamplePoint, history []SamplePoint) []Alert {
	var out []Alert
	rateSamples := []float64{}
	for _, h := range history {
		for _, c := range h.Channels {
			if c.ID == "C7" && (c.RxBps+c.TxBps) > 0 {
				rateSamples = append(rateSamples, c.RxBps+c.TxBps)
			}
		}
	}
	for _, c := range pt.Channels {
		switch c.ID {
		case "C2":
			// secondary→primary uplink RTT from agent
			if c.RTTMs <= 0 {
				if !c.OK {
					out = append(out, Alert{
						Key: fmt.Sprintf("channel:uplink:%s", c.Peer),
						Msg: fmt.Sprintf("🔴 Channel C2 peer %s uplink down", c.Peer),
						Incident: true,
					})
				}
				continue
			}
			level := rttLevel(c.RTTMs, uplinkRTTWarnMs, uplinkRTTCritMs)
			if level == 0 {
				continue
			}
			if !sustainedRTT(history, "C2", c.Peer, level, c.RTTMs) {
				continue
			}
			key := fmt.Sprintf("channel:rtt:C2:%s", c.Peer)
			if level >= 2 {
				out = append(out, Alert{
					Key: key,
					Msg: fmt.Sprintf("🔴 Channel C2 peer %s uplink RTT %.0fms sustained (crit ≥%.0f)", c.Peer, c.RTTMs, uplinkRTTCritMs),
					Incident: true,
				})
			} else {
				out = append(out, Alert{
					Key: key,
					Msg: fmt.Sprintf("⚠️ Channel C2 peer %s uplink RTT %.0fms sustained", c.Peer, c.RTTMs),
				})
			}
		case "C3":
			// primary→secondary public :443 dial — only extreme / fail
			if !c.OK {
				out = append(out, Alert{
					Key: fmt.Sprintf("channel:face:%s", c.Peer),
					Msg: fmt.Sprintf("🔴 Channel C3 peer %s face :443 unreachable", c.Peer),
					Incident: true,
				})
				continue
			}
			if c.RTTMs <= 0 {
				continue
			}
			level := rttLevel(c.RTTMs, faceDialWarnMs, faceDialCritMs)
			if level == 0 {
				continue
			}
			if !sustainedRTT(history, "C3", c.Peer, level, c.RTTMs) {
				continue
			}
			key := fmt.Sprintf("channel:rtt:C3:%s", c.Peer)
			if level >= 2 {
				out = append(out, Alert{
					Key: key,
					Msg: fmt.Sprintf("🔴 Channel C3 peer %s face dial %.0fms sustained (crit ≥%.0f)", c.Peer, c.RTTMs, faceDialCritMs),
					Incident: true,
				})
			} else {
				out = append(out, Alert{
					Key: key,
					Msg: fmt.Sprintf("⚠️ Channel C3 peer %s face dial %.0fms sustained", c.Peer, c.RTTMs),
				})
			}
		case "C4", "C5", "C6":
			// service plane — reuse uplink thresholds, sustained only
			if c.RTTMs <= 0 {
				continue
			}
			level := rttLevel(c.RTTMs, uplinkRTTWarnMs, uplinkRTTCritMs)
			if level == 0 || !sustainedRTT(history, c.ID, c.Peer, level, c.RTTMs) {
				continue
			}
			key := fmt.Sprintf("channel:rtt:%s:%s", c.ID, c.Peer)
			if level >= 2 {
				out = append(out, Alert{Key: key, Msg: fmt.Sprintf("🔴 Channel %s peer %s RTT %.0fms sustained (crit)", c.ID, c.Peer, c.RTTMs), Incident: true})
			} else {
				out = append(out, Alert{Key: key, Msg: fmt.Sprintf("⚠️ Channel %s peer %s RTT %.0fms sustained", c.ID, c.Peer, c.RTTMs)})
			}
		case "C7":
			agg := c.RxBps + c.TxBps
			if agg <= 0 {
				continue // idle — not collapse
			}
			p50 := percentile(rateSamples, 0.5)
			if p50 > 50e6 { // only absolute floors if baseline was healthy
				mbps := agg / 1e6
				if mbps < 8 {
					out = append(out, Alert{Key: "channel:rate:c7", Msg: fmt.Sprintf("🔴 Host egress rate collapse: %.1f Mbps (crit)", mbps), Incident: true})
				} else if mbps < 15 {
					out = append(out, Alert{Key: "channel:rate:c7", Msg: fmt.Sprintf("⚠️ Host egress rate low: %.1f Mbps", mbps)})
				}
			}
			if p50 > 0 && agg < p50*DefaultRateRatioCrit {
				out = append(out, Alert{Key: "channel:rate:c7", Msg: fmt.Sprintf("🔴 Host rate <10%% of 1h baseline (%.0f bps vs p50 %.0f)", agg, p50), Incident: true})
			} else if p50 > 0 && agg < p50*DefaultRateRatioWarn {
				out = append(out, Alert{Key: "channel:rate:c7", Msg: fmt.Sprintf("⚠️ Host rate <20%% of 1h baseline (%.0f bps vs p50 %.0f)", agg, p50)})
			}
		}
	}
	return out
}

func rttLevel(ms, warn, crit float64) int {
	if ms >= crit {
		return 2
	}
	if ms >= warn {
		return 1
	}
	return 0
}

// sustainedRTT requires rttConsecutive samples (current + recent history) at the same level.
func sustainedRTT(history []SamplePoint, id, peer string, level int, current float64) bool {
	warn, crit := uplinkRTTWarnMs, uplinkRTTCritMs
	if id == "C3" {
		warn, crit = faceDialWarnMs, faceDialCritMs
	}
	need := rttConsecutive
	n := 0
	if rttLevel(current, warn, crit) >= level {
		n++
	}
	// walk history newest-last: ReadHistory is chronological append order
	for i := len(history) - 1; i >= 0 && n < need; i-- {
		for _, c := range history[i].Channels {
			if c.ID != id || c.Peer != peer || c.RTTMs <= 0 {
				continue
			}
			if rttLevel(c.RTTMs, warn, crit) >= level {
				n++
			} else {
				return false // break streak
			}
			break
		}
	}
	return n >= need
}

func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	cp := append([]float64(nil), vals...)
	for i := 1; i < len(cp); i++ {
		for j := i; j > 0 && cp[j] < cp[j-1]; j-- {
			cp[j], cp[j-1] = cp[j-1], cp[j]
		}
	}
	idx := int(float64(len(cp)-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

// SampleAndEvaluate is used by collect/metrics tick when channel sample is enabled.
func SampleAndEvaluate() []Alert {
	pt, err := SampleOnce()
	if err != nil {
		return nil
	}
	hist, _ := ReadHistory(time.Now().UTC().Add(-time.Hour), "")
	// exclude the sample we just appended from consecutive count (use prior only)
	if len(hist) > 0 {
		last := hist[len(hist)-1]
		if !last.TS.IsZero() && pt.TS.Sub(last.TS) < 2*time.Second {
			hist = hist[:len(hist)-1]
		}
	}
	return EvaluateSampleAlerts(pt, hist)
}
