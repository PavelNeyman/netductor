package channels

import (
	"fmt"
	"time"
)

// Alert is a channel degradation signal for notify.AlertOnce.
type Alert struct {
	Key string
	Msg string
}

// EvaluateSampleAlerts checks RTT and rate collapse against plan defaults.
// History window: last hour of samples for p50-ish baseline (median of recent RTT/rates).
func EvaluateSampleAlerts(pt SamplePoint, history []SamplePoint) []Alert {
	var out []Alert
	// baseline from history
	rttSamples := map[string][]float64{} // peer -> rtts
	rateSamples := []float64{}
	for _, h := range history {
		for _, c := range h.Channels {
			if c.RTTMs > 0 {
				rttSamples[c.Peer] = append(rttSamples[c.Peer], c.RTTMs)
			}
			if c.ID == "C7" && (c.RxBps+c.TxBps) > 0 {
				rateSamples = append(rateSamples, c.RxBps+c.TxBps)
			}
		}
	}
	for _, c := range pt.Channels {
		switch c.ID {
		case "C2", "C3", "C4", "C5", "C6":
			if c.RTTMs <= 0 {
				continue
			}
			p50 := percentile(rttSamples[c.Peer], 0.5)
			warn := c.RTTMs > DefaultRTTWarnMs
			if p50 > 0 && c.RTTMs > p50*3 {
				warn = true
			}
			crit := c.RTTMs > DefaultRTTCritMs
			if p50 > 0 && c.RTTMs > p50*5 {
				crit = true
			}
			if crit {
				out = append(out, Alert{
					Key: fmt.Sprintf("channel:rtt:%s:%s", c.ID, c.Peer),
					Msg: fmt.Sprintf("🔴 Channel %s peer %s RTT %.0fms (crit)", c.ID, c.Peer, c.RTTMs),
				})
			} else if warn {
				out = append(out, Alert{
					Key: fmt.Sprintf("channel:rtt:%s:%s", c.ID, c.Peer),
					Msg: fmt.Sprintf("⚠️ Channel %s peer %s RTT %.0fms", c.ID, c.Peer, c.RTTMs),
				})
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
					out = append(out, Alert{
						Key: "channel:rate:c7",
						Msg: fmt.Sprintf("🔴 Host egress rate collapse: %.1f Mbps (crit)", mbps),
					})
				} else if mbps < 15 {
					out = append(out, Alert{
						Key: "channel:rate:c7",
						Msg: fmt.Sprintf("⚠️ Host egress rate low: %.1f Mbps", mbps),
					})
				}
			}
			if p50 > 0 && agg < p50*DefaultRateRatioCrit {
				out = append(out, Alert{
					Key: "channel:rate:c7",
					Msg: fmt.Sprintf("🔴 Host rate <10%% of 1h baseline (%.0f bps vs p50 %.0f)", agg, p50),
				})
			} else if p50 > 0 && agg < p50*DefaultRateRatioWarn {
				out = append(out, Alert{
					Key: "channel:rate:c7",
					Msg: fmt.Sprintf("⚠️ Host rate <20%% of 1h baseline (%.0f bps vs p50 %.0f)", agg, p50),
				})
			}
		}
	}
	return out
}

func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	cp := append([]float64(nil), vals...)
	// simple insertion sort
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

// SampleAndEvaluate is used by collect/metrics tick when NETDUCTOR_CHANNEL_SAMPLE=1.
func SampleAndEvaluate() []Alert {
	pt, err := SampleOnce()
	if err != nil {
		return nil
	}
	hist, _ := ReadHistory(time.Now().UTC().Add(-time.Hour), "")
	return EvaluateSampleAlerts(pt, hist)
}
