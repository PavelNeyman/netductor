// Package fleet aggregates day-2 health for operator digests (TG/API/CLI).
package fleet

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/secondary"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	"github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

// Digest is a single snapshot of dual-node + VPN soft signals.
type Digest struct {
	TS              time.Time         `json:"ts"`
	PrimaryVersion  string            `json:"primary_version"`
	UpdateAvailable bool              `json:"update_available"`
	LatestRelease   string            `json:"latest_release,omitempty"`
	Services        map[string]string `json:"services"`
	Secondaries     []SecRow          `json:"secondaries"`
	SPOK            *bool             `json:"sp_ok,omitempty"`
	PSOK            *bool             `json:"ps_ok,omitempty"`
	BackupAgeHours  *float64          `json:"backup_age_hours,omitempty"`
	LEHosts         []string          `json:"le_hosts,omitempty"`
	VPNUsers        int               `json:"vpn_users"`
	VPNQuiet        []string          `json:"vpn_quiet,omitempty"` // no activity hint
	VPNNearQuota    []string          `json:"vpn_near_quota,omitempty"`
	Notes           []string          `json:"notes,omitempty"`
	FirewallOK      *bool             `json:"firewall_ok,omitempty"`
	FirewallBackend string            `json:"firewall_backend,omitempty"`
}

// SecRow is one secondary agent line.
type SecRow struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Online  bool   `json:"online"`
	Version string `json:"version,omitempty"`
	Uplink  bool   `json:"uplink_ok"`
	SingBox bool   `json:"singbox_ok"`
	IP      string `json:"ip,omitempty"`
}

// Build collects live signals without failing hard on missing optional state.
func Build() Digest {
	d := Digest{
		TS:             time.Now().UTC(),
		PrimaryVersion: version.Running(),
		Services:       map[string]string{},
	}
	for _, u := range []string{"sing-box", "netductor-api", "netductor-telegram-bot", "netductor-redirect"} {
		out, err := exec.Command("systemctl", "is-active", u).CombinedOutput()
		st := strings.TrimSpace(string(out))
		if err != nil && st == "" {
			st = "unknown"
		}
		d.Services[u] = st
	}
	{
		fw := firewall.AlertIfUnhealthy()
		ok := fw.OK
		d.FirewallOK = &ok
		d.FirewallBackend = string(fw.Backend)
		if !fw.OK {
			d.Notes = append(d.Notes, "firewall not OK: "+string(fw.Backend))
		}
	}
	st := ndupdate.CheckStatus(version.Running())
	if st.Error == "" {
		d.UpdateAvailable = st.Update
		d.LatestRelease = st.Latest
	}
	for _, sec := range secondary.List() {
		on := secondary.Online(sec, 2*time.Minute)
		d.Secondaries = append(d.Secondaries, SecRow{
			ID: sec.ID, Name: sec.Name, Online: on, Version: sec.Version,
			Uplink: sec.UplinkOK, SingBox: sec.SingBoxOK, IP: sec.PublicIP,
		})
	}
	// svc paths health file (written by timer)
	if b, err := os.ReadFile(filepath.Join(paths.StateDir(), "svc-paths-health.json")); err == nil {
		s := string(b)
		if strings.Contains(s, `"sp"`) || strings.Contains(s, "sp") {
			// best-effort parse without full JSON dependency on shape
			sp := strings.Contains(s, `"sp_ok":true`) || strings.Contains(s, `"sp":true`)
			ps := strings.Contains(s, `"ps_ok":true`) || strings.Contains(s, `"ps":true`)
			d.SPOK = &sp
			d.PSOK = &ps
		}
	}
	// latest backup age
	bakDir := filepath.Join(paths.StateDir(), "backups")
	entries, _ := os.ReadDir(bakDir)
	var newest time.Time
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		if !strings.HasSuffix(n, ".ndenc") && !strings.HasSuffix(n, ".tar.gz") {
			continue
		}
		if info, err := e.Info(); err == nil {
			if info.ModTime().After(newest) {
				newest = info.ModTime()
			}
		}
	}
	if !newest.IsZero() {
		h := time.Since(newest).Hours()
		d.BackupAgeHours = &h
	}
	// LE live certs
	live := "/etc/letsencrypt/live"
	if ents, err := os.ReadDir(live); err == nil {
		for _, e := range ents {
			if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
				d.LEHosts = append(d.LEHosts, e.Name())
			}
		}
	}
	// VPN soft signals
	users, _ := vpn.List()
	d.VPNUsers = len(users)
	for _, u := range users {
		name := u.Name
		if name == "" {
			continue
		}
		// quiet: no last_seen in client state for 14d — soft only via quota file presence
		lim := vpn.SoftLimitGB(name)
		if lim > 0 {
			// near quota is informational; real bytes need metrics — flag configured limits
			d.VPNNearQuota = append(d.VPNNearQuota, name)
		}
	}
	if len(d.VPNNearQuota) > 5 {
		d.VPNNearQuota = d.VPNNearQuota[:5]
		d.Notes = append(d.Notes, "vpn soft limits configured (sample)")
	}
	return d
}
