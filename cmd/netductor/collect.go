package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/hardening"
	"github.com/PavelNeyman/netductor/internal/metrics"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/probes"
	"github.com/PavelNeyman/netductor/internal/secondary"
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
	if enabled("secondary_offline") {
		for _, d := range secondary.List() {
			key := "secondary:" + d.ID
			if !secondary.Online(d, 3*time.Minute) {
				notify.AlertOnce(key, fmt.Sprintf("🔴 Secondary offline: <b>%s</b> (%s)", d.Name, d.PublicIP))
			} else {
				notify.ClearAlert(key)
				if !d.SingBoxOK {
					notify.AlertOnce(key+":sb", fmt.Sprintf("⚠️ Secondary <b>%s</b> sing-box not active", d.Name))
				} else {
					notify.ClearAlert(key + ":sb")
				}
			}
		}
	}
	if enabled("sni_health") {
		ok, ms, det := vpn.SNIHealth()
		vpn.WriteSNIHealthMetric(ok, ms, det)
		// Reality often fails plain TLS probe — only alert on dial failure
		if strings.Contains(det, "connection refused") || strings.Contains(det, "i/o timeout") {
			notify.AlertOnce("sni:down", fmt.Sprintf("⚠️ VLESS port/SNI probe failed: %s", det))
		} else {
			notify.ClearAlert("sni:down")
		}
		_ = ok
		_ = ms
	}
	if enabled("mismatch_spike") {
		st := vpn.CollectMismatch(30)
		if st.Total >= 20 {
			msg := fmt.Sprintf("⚠️ Flow mismatch spike: <b>%d</b> in 30m\n<code>%s</code>\n<i>Usually clients without vision flow (old link / phone without config)</i>", st.Total, vpn.FormatMismatchText(st))
			notify.AlertOnce("mismatch:core", msg)
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
	_ = m
}
