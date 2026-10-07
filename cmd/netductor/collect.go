package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/channels"
	"github.com/PavelNeyman/netductor/internal/logs"
	"github.com/PavelNeyman/netductor/internal/hardening"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/metrics"
	"github.com/PavelNeyman/netductor/internal/notify"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	ndver "github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/probes"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/svcpaths"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func runCollect() int {
	dir := paths.MetricsDir()
	_ = os.MkdirAll(dir, 0o755)

	lockPath := filepath.Join(dir, "collector.lock")
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	defer lf.Close()
	// best-effort exclusive lock via O_EXCL style: write pid if empty
	// (full flock needs syscall; skip concurrent races for MVP)

	cfg := probes.Load()
	// ensure probes.cfg exists
	if _, err := os.Stat(probes.CfgPath()); err != nil {
		_ = probes.Save(cfg)
	}

	m := metrics.Collect()
	mem, _ := m["mem"].(map[string]int64)
	if mem != nil {
		total := mem["total"]
		if total <= 0 {
			total = 1
		}
		m["mem_pct"] = float64(mem["used"]) / float64(total) * 100
	}
	disk, _ := m["disk"].(map[string]any)
	if disk != nil {
		var total, used float64
		switch v := disk["total"].(type) {
		case int64:
			total = float64(v)
		case int:
			total = float64(v)
		case float64:
			total = v
		}
		switch v := disk["used"].(type) {
		case int64:
			used = float64(v)
		case int:
			used = float64(v)
		case float64:
			used = v
		}
		if total <= 0 {
			total = 1
		}
		m["disk_pct"] = used / total * 100
	}

	live := probes.Run(cfg)
	// dynamic: TCP 443 to each enrolled secondary
	for _, d := range secondary.List() {
		if d.PublicIP == "" {
			continue
		}
		p := map[string]any{"name": "secondary-" + d.PublicIP, "type": "tcp", "host": d.PublicIP, "port": 443, "timeout": 5}
		res := probes.Run(map[string]any{"probes": []any{p}})
		live = append(live, res...)
	}
	m["probes"] = live

	ch := channels.Collect()
	m["channels"] = ch

	// history
	hist := filepath.Join(dir, "history.jsonl")
	line, _ := json.Marshal(m)
	f, err := os.OpenFile(hist, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		_, _ = f.Write(append(line, '\n'))
		_ = f.Close()
		// trim to last 10000 lines
		if b, err := os.ReadFile(hist); err == nil {
			lines := splitKeep(b, 10000)
			_ = os.WriteFile(hist, lines, 0o644)
		}
	}
	latest := filepath.Join(dir, "latest.json")
	_ = os.WriteFile(latest, append(line, '\n'), 0o644)

	install.EnsureRedirectRunning()
	evaluateSimpleAlerts(m, live, cfg)
	fmt.Printf("collected ts=%v probes=%d → %s\n", m["ts"], len(live), latest)
	return 0
}

func splitKeep(b []byte, max int) []byte {
	// keep last max lines
	n := 0
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] == '\n' {
			n++
			if n > max {
				return b[i+1:]
			}
		}
	}
	return b
}

func evaluateSimpleAlerts(m map[string]any, live []map[string]any, cfg map[string]any) {
	al, _ := cfg["alerts"].(map[string]any)
	if al == nil {
		al = map[string]any{"probe_fail": true, "service_down": true, "secondary_offline": true}
	}
	enabled := func(k string) bool {
		v, ok := al[k].(bool)
		return !ok || v // default on
	}
	if enabled("probe_fail") {
		for _, p := range live {
			if ok, _ := p["ok"].(bool); !ok {
				name, _ := p["name"].(string)
				err, _ := p["error"].(string)
				notify.AlertOnce("probe:"+name, fmt.Sprintf("⚠️ Probe <b>%s</b> failed: %s", name, err))
			} else {
				name, _ := p["name"].(string)
				notify.ClearAlert("probe:" + name)
			}
		}
	}
	if enabled("service_down") {
		units := []string{"sing-box", "netductor-api", "netductor-telegram-bot"}
		// primary-only optional unit
		if _, err := os.Stat("/etc/systemd/system/netductor-redirect.service"); err == nil {
			units = append(units, "netductor-redirect")
		}
		for _, u := range units {
			out, err := exec.Command("systemctl", "is-active", u).CombinedOutput()
			st := strings.TrimSpace(string(out))
			if err != nil || st != "active" {
				notify.AlertOnce("svc:"+u, fmt.Sprintf("🔴 Service <b>%s</b> is %s", u, st))
			} else {
				notify.ClearAlert("svc:" + u)
			}
		}
	}
	if enabled("svc_path_down") {
		b, err := os.ReadFile("/var/lib/netductor/svc-paths-health.json")
		if err == nil {
			var h map[string]any
			if json.Unmarshal(b, &h) == nil {
				sp, _ := h["svc_sp_up"].(float64)
				ps, _ := h["svc_ps_up"].(float64)
				if int(sp) != 1 {
					notify.AlertOnce("svcpath:sp", "🔴 Service path <b>SP</b> (nd-svc-sp) down")
				} else {
					notify.ClearAlert("svcpath:sp")
				}
				if int(ps) != 1 {
					notify.AlertOnce("svcpath:ps", "🔴 Service path <b>PS</b> (nd-svc-ps) down")
				} else {
					notify.ClearAlert("svcpath:ps")
				}
			}
		}
		// advance failover state machine (anti-flap + desired JSON)
		if _, _, err := svcpaths.Tick(); err != nil {
			_ = err
		}
	}
	if enabled("secondary_offline") {
		// Drop prepare-pack ghosts (never heartbeated) and long-stale empties
		if n := secondary.PruneStale(24 * time.Hour); n > 0 {
			fmt.Fprintf(os.Stderr, "secondary prune: removed %d stale/ghost device(s)\n", n)
		}
		for _, d := range secondary.List() {
			key := "secondary:" + d.ID
			// Never alert for tokens that never joined (empty last_seen) — prepare-pack residue
			if d.LastSeen.IsZero() {
				notify.ClearAlert(key)
				notify.ClearAlert(key + ":sb")
				notify.ClearAlert(key + ":uplink")
				continue
			}
			label := d.Name
			if label == "" {
				label = d.ID
			}
			ip := d.PublicIP
			if ip == "" {
				ip = "no-ip"
			}
			if !secondary.Online(d, 3*time.Minute) {
				notify.AlertOnce(key, fmt.Sprintf("🔴 Secondary offline: <b>%s</b> (%s)", label, ip))
			} else {
				notify.ClearAlert(key)
				if !d.SingBoxOK {
					notify.AlertOnce(key+":sb", fmt.Sprintf("⚠️ Secondary <b>%s</b> sing-box not active", label))
				} else {
					notify.ClearAlert(key + ":sb")
				}
				if !d.UplinkOK {
					notify.AlertOnce(key+":uplink", fmt.Sprintf("⚠️ Secondary <b>%s</b> uplink to primary:443 failed (mux may be stuck)", label))
				} else {
					notify.ClearAlert(key + ":uplink")
				}
			}
		}
	}
	if enabled("sni_health") {
		ok, ms, det := vpn.SNIHealth()
		vpn.WriteSNIHealthMetric(ok, ms, det)
		// Only real dial failures (port closed). Reality rejects plain TLS — not an outage.
		if vpn.SNIDialDown(det) || !ok {
			notify.AlertOnce("sni:down", fmt.Sprintf("⚠️ VLESS port closed/unreachable: %s", det))
		} else {
			notify.ClearAlert("sni:down")
		}
		_ = ms
	}
	if enabled("channel_health") {
		ch := channels.Collect()
		for _, s := range ch.Secondaries {
			key := "channel:tcp443:" + s.ID
			if s.Online && s.PublicIP != "" && !s.TCP443OK {
				notify.AlertOnce(key, fmt.Sprintf("🔴 Channel <b>%s</b>: primary→secondary:443 TCP fail (ip %s)", s.Name, s.PublicIP))
			} else {
				notify.ClearAlert(key)
			}
			keyU := "channel:uplink:" + s.ID
			if s.Online && !s.UplinkOK {
				notify.AlertOnce(keyU, fmt.Sprintf("⚠️ Channel <b>%s</b>: secondary→primary:443 uplink probe failed", s.Name))
			} else {
				notify.ClearAlert(keyU)
			}
		}
		// Reality invalid from secondary IP = uplink path noise / flap
		if ch.RealityInvalidFromSec15m >= 30 {
			notify.AlertOnce("channel:reality-sec",
				fmt.Sprintf("⚠️ Reality invalid from secondary IPs: <b>%d</b> in 15m (uplink path noise)", ch.RealityInvalidFromSec15m))
		} else {
			notify.ClearAlert("channel:reality-sec")
		}
		if ch.RealityInvalidTotal15m >= 80 {
			notify.AlertOnce("channel:reality-total",
				fmt.Sprintf("⚠️ Reality invalid total: <b>%d</b> in 15m (scanners + clients)", ch.RealityInvalidTotal15m))
		} else {
			notify.ClearAlert("channel:reality-total")
		}
		// Attach last-1h journals when channel degraded (best-effort, once per alert key cooldown)
		needAttach := false
		for _, s := range ch.Secondaries {
			if s.Online && ((s.PublicIP != "" && !s.TCP443OK) || !s.UplinkOK) {
				needAttach = true
			}
		}
		if ch.RealityInvalidFromSec15m >= 30 || ch.RealityInvalidTotal15m >= 80 {
			needAttach = true
		}
		if needAttach {
			if path, err := logs.ExportHours(1); err == nil {
				_ = notify.SendDocument(path, "📎 channel incident logs (last 1h)")
			}
		}
	}

	if enabled("mismatch_spike") {
		st := vpn.CollectMismatch(30)
		st10 := vpn.CollectMismatch(10)
		if st10.Total >= 15 || st.Total >= 25 {
			msg := fmt.Sprintf("⚠️ Flow mismatch spike: <b>%d</b>/10m, <b>%d</b>/30m\n<code>%s</code>\n<i>Server expects vision; client sent empty flow</i>", st10.Total, st.Total, vpn.FormatMismatchText(st10))
			notify.AlertOnce("mismatch:core", msg)
			if path, err := logs.ExportHours(1); err == nil {
				_ = notify.SendDocument(path, "📎 flow mismatch logs (last 1h)")
			}
		} else {
			notify.ClearAlert("mismatch:core")
		}
	}
	if n, err := vpn.ExpireGuests(); err == nil && n > 0 {
		notify.AlertOnce("guest:expire", fmt.Sprintf("🧹 Expired <b>%d</b> guest VPN user(s)", n))
	}
	for _, m := range hardening.UnusualSSHAlerts() {
		key := "ssh:unusual"
		if len(m) > 24 {
			key = "ssh:unusual:" + m[len(m)-24:]
		}
		notify.AlertOnce(key, "🔐 "+m)
	}
	if enabled("backup_offsite") {
		// marker written by backup on scp failure
		p := filepath.Join(paths.StateDir(), "backup_offsite_fail")
		if b, err := os.ReadFile(p); err == nil && len(b) > 0 {
			notify.AlertOnce("backup:offsite", "⚠️ Backup offsite failed:\n<pre>"+string(b)+"</pre>")
		} else {
			notify.ClearAlert("backup:offsite")
		}
	}

	// GitHub release newer than local node — notify once per remote tag
	if enabled("release_update") {
		st := ndupdate.CheckStatus(ndver.Release)
		if st.Error == "" && st.Update && st.Latest != "" {
			key := "update:available:" + st.Latest
			notify.AlertOnce(key, fmt.Sprintf(
				"🆕 <b>Netductor update</b>\nlocal <code>%s</code> → latest <code>%s</code>\n<i>TG Tools → Updates · or: netductor update apply %s</i>",
				st.Local, st.Latest, strings.TrimPrefix(st.Latest, "v")))
		} else if st.Error == "" && !st.Update {
			// clear any prior version keys is hard; clear generic
			notify.ClearAlert("update:available")
		}
	}
	_ = m
}
