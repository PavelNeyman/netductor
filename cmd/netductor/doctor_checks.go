package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/channels"
	"github.com/PavelNeyman/netductor/internal/ci"
	"github.com/PavelNeyman/netductor/internal/cli18n"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/git"
	"github.com/PavelNeyman/netductor/internal/hardening"
	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/integrity"
	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/nvr"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/tlsle"
	"github.com/PavelNeyman/netductor/internal/policy"
	"github.com/PavelNeyman/netductor/internal/registry"
	"github.com/PavelNeyman/netductor/internal/servicenet"
	"github.com/PavelNeyman/netductor/internal/sites"
	"github.com/PavelNeyman/netductor/internal/svcpaths"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	"github.com/PavelNeyman/netductor/internal/upstream"
	"github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func doctorRunChecks(h doctorHooks) {
	h.check("debian", h.exists("/etc/debian_version"))
	h.check("secrets", h.exists(filepath.Join(h.etc, "secrets")))
	h.check("netductor", lookPath("netductor") != "")
	h.check("sing-box binary", h.exists("/usr/local/bin/sing-box"))
	h.check("sing-box unit", activeUnit("sing-box"))
	if h.exists("/usr/local/bin/sing-box") && h.exists("/usr/local/etc/sing-box/config.json") {
		h.check("sing-box config", exec.Command("/usr/local/bin/sing-box", "check", "-c", "/usr/local/etc/sing-box/config.json").Run() == nil)
		fi, err := os.Stat("/usr/local/etc/sing-box/config.json")
		h.warnCheck("sing-box config perms 600", err == nil && fi.Mode().Perm()&0o077 == 0)
	} else {
		h.check("sing-box config", false)
	}

	pa := sshdT("passwordauthentication")
	h.check("ssh passwordauth no", pa == "no")
	x11 := sshdT("x11forwarding")
	h.warnCheck("ssh x11forwarding no", x11 == "no" || x11 == "")
	wantPort := hardening.SSHPort()
	gotPort := sshdT("port")
	h.warnCheck(fmt.Sprintf("ssh port %d (got %s)", wantPort, gotPort), gotPort == fmt.Sprintf("%d", wantPort) || gotPort == "")
	if os.Getenv("NETDUCTOR_FAIL2BAN") != "0" {
		h.warnCheck("fail2ban active", activeUnit("fail2ban"))
	}
	// Hoster monitoring / CM agents (units, ports, residual packages/paths)
	hu, hp := install.HostAgentFindings()
	hpkgs, hpaths := install.HostAgentResidual()
	hprocs := install.HostAgentProcs()
	if len(hu) == 0 && len(hp) == 0 && len(hpkgs) == 0 && len(hpaths) == 0 && len(hprocs) == 0 {
		doctorPrintln("OK   no hoster monitoring/CM agents")
		(*h.ok)++
	} else {
		if len(hu) > 0 {
			doctorPrintf("FAIL hoster units active: %s\n", strings.Join(hu, ", "))
			(*h.fail)++
		}
		if len(hp) > 0 {
			doctorPrintf("WARN hoster-like ports open: %s\n", strings.Join(hp, ", "))
			(*h.warn)++
		}
		if len(hpkgs) > 0 {
			doctorPrintf("FAIL hoster packages installed: %s\n", strings.Join(hpkgs, ", "))
			(*h.fail)++
		}
		if len(hpaths) > 0 {
			doctorPrintf("FAIL hoster residual paths: %s\n", strings.Join(hpaths, ", "))
			(*h.fail)++
		}
		if len(hprocs) > 0 {
			doctorPrintf("FAIL hoster procs: %s\n", strings.Join(hprocs, ", "))
			(*h.fail)++
		}
	}

	fw := firewall.Collect(h.role)
	if fw.OK {
		doctorPrintf("OK   firewall backend=%s active=%v\n", fw.Backend, fw.Active)
		(*h.ok)++
	} else {
		doctorPrintf("FAIL firewall backend=%s active=%v detail=%s\n", fw.Backend, fw.Active, fw.Detail)
		if len(fw.Warnings) > 0 {
			for _, w := range fw.Warnings {
				doctorPrintf("WARN firewall: %s\n", w)
				(*h.warn)++
			}
		}
		(*h.fail)++
	}

	bl := install.CheckHostBaseline()
	if bl.OK {
		doctorPrintln("OK   host baseline (role+apt-pin+firewall+watchdog+hoster)")
		(*h.ok)++
	} else {
		for _, is := range bl.Issues {
			doctorPrintf("WARN host baseline: %s\n", is)
			(*h.warn)++
		}
	}

	drift := install.DiffHostBaseline()
	if !drift.HaveSnapshot {
		doctorPrintln("INFO no host snapshot yet (written at install / host-baseline apply)")
	} else if len(drift.NewPorts) == 0 && len(drift.NewUnits) == 0 {
		doctorPrintf("OK   host snapshot matches (%s)\n", drift.Created)
		(*h.ok)++
	} else {
		if len(drift.NewPorts) > 0 {
			doctorPrintf("WARN snapshot new listen: %s\n", strings.Join(drift.NewPorts, ", "))
			(*h.warn)++
		}
		if len(drift.NewUnits) > 0 {
			doctorPrintf("WARN snapshot new units: %s\n", strings.Join(drift.NewUnits, ", "))
			(*h.warn)++
		}
	}

	idrift := integrity.Check()
	if !idrift.HaveManifest {
		doctorPrintln("INFO no integrity manifest yet (host-baseline apply / stack apply)")
	} else if idrift.OK {
		doctorPrintln("OK   integrity manifest matches (files + SSH keys)")
		(*h.ok)++
	} else {
		doctorPrintf("WARN integrity drift files=%d new_ssh=%d\n", len(idrift.ChangedFiles), len(idrift.NewSSH))
		(*h.warn)++
	}
	if rok, _ := install.RebootRequired(); rok {
		doctorPrintln("WARN reboot required (/var/run/reboot-required)")
		(*h.warn)++
	}

	// Service-net (primary VIP plane for internal services)
	if h.role == "primary" && servicenet.Enabled() {
		st := servicenet.Collect()
		if st.OK {
			doctorPrintln("OK   service-net " + st.Iface + " " + st.Gateway)
			(*h.ok)++
		} else {
			doctorPrintf("WARN service-net: %s (netductor servicenet apply)\n", st.Detail)
			(*h.warn)++
		}
	}
	// Policy / catalog integrity (primary)
	if h.role == "primary" {
		if cat, err := policy.EnsureCatalog(); err != nil {
			doctorPrintf("WARN policy catalog: %v\n", err)
			(*h.warn)++
		} else {
			for _, is := range policy.CheckCatalogIntegrity(cat) {
				doctorPrintf("WARN catalog %s: %s\n", is.Subject, is.Detail)
				(*h.warn)++
			}
			if users, err := vpn.ListNative(); err == nil {
				for _, u := range users {
					if vpn.IsEdgeUser(u.Name) {
						continue
					}
					p, _ := vpn.GetUserPolicy(u.Name)
					for _, is := range policy.CheckPolicyAgainstCatalog(u.Name, "user", p, cat) {
						doctorPrintf("WARN user policy %s: %s\n", is.Subject, is.Detail)
						(*h.warn)++
					}
				}
			}
			// edge devices
			for _, d := range edge.ListDevices() {
				p, err := edge.GetDevicePolicy(d.DeviceID)
				if err != nil {
					continue
				}
				for _, is := range policy.CheckPolicyAgainstCatalog(d.DeviceID, "edge", p, cat) {
					doctorPrintf("WARN edge policy %s: %s\n", is.Subject, is.Detail)
					(*h.warn)++
				}
			}
		}
	}

	switch h.role {
	case "primary":
		h.check("READY.txt", h.exists(filepath.Join(h.etc, "READY.txt")))
		h.check("vpn-users.json", h.exists(filepath.Join(h.etc, "vpn-users.json")))
		if users, err := vpn.List(); err == nil {
			live := make([]string, 0, len(users))
			for _, u := range users {
				live = append(live, u.Name)
			}
			if n := len(vpn.ListOrphanSubUsers(live)); n > 0 {
				doctorPrintf("WARN orphan sub tokens: %d (netductor vpn sub-prune)\n", n)
				(*h.warn)++
			}
		}
		h.check("blocky unit", activeUnit("blocky"))
		if activeUnit("blocky") {
			if listeningOnAll("53") {
				doctorPrintln(cli18n.T("doctor.blocky_all"))
				(*h.warn)++
			} else if listeningLocalhost("53") {
				doctorPrintln(cli18n.T("doctor.blocky_local"))
				(*h.ok)++
			} else {
				doctorPrintln(cli18n.T("doctor.blocky_unknown"))
				(*h.warn)++
			}
		}
		h.check("netductor-api", activeUnit("netductor-api"))
		// Telegram is an optional add-on — only require unit if installed
		if h.exists("/etc/systemd/system/netductor-telegram-bot.service") || h.exists("/lib/systemd/system/netductor-telegram-bot.service") {
			h.check("netductor-telegram-bot", activeUnit("netductor-telegram-bot"))
		} else {
			doctorPrintln(cli18n.T("doctor.tg_skip"))
		}
		// import redirect for TG deep-link buttons (port 80, already allowed for ACME)
		h.warnCheck("netductor-redirect unit", activeUnit("netductor-redirect"))
		if strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_BASE")) == "" {
			doctorPrintln(cli18n.T("doctor.redirect_unset"))
		} else {
			doctorPrintf(cli18n.T("doctor.redirect_ok")+"\n", os.Getenv("NETDUCTOR_REDIRECT_BASE"))
		}
		if activeUnit("netductor-redirect") {
			// Product: redirect on HTTPS :8443 (no public :80). Accept either bind for healthz.
			okH := h.curlOKInsecure("https://127.0.0.1:8443/healthz") || h.curlOK("http://127.0.0.1/healthz") || h.curlOK("http://127.0.0.1:8443/healthz")
			if okH {
				doctorPrintln(cli18n.T("doctor.redirect_health_ok"))
				(*h.ok)++
			} else {
				doctorPrintln(cli18n.T("doctor.redirect_health_bad"))
				(*h.warn)++
			}
		}
		
		// LE / :80 surface
		if tlsle.HooksInstalled() {
			doctorPrintln("OK   le renew hooks (pre/post/deploy free :80)")
			(*h.ok)++
		} else if _, err := os.Stat("/etc/letsencrypt"); err == nil {
			doctorPrintln("WARN le renew hooks missing (netductor host-baseline apply installs)")
			(*h.warn)++
		}
		if listeners := tlsle.Port80Listeners(); listeners != "" {
			doctorPrintln("WARN :80 has listener(s) — expect free except during certbot standalone:")
			for _, line := range strings.Split(listeners, "\n") {
				if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "State") {
					doctorPrintln("      " + strings.TrimSpace(line))
				}
			}
			(*h.warn)++
		} else if _, err := os.Stat("/etc/letsencrypt"); err == nil {
			doctorPrintln("OK   :80 no local listener (ACME standby)")
			(*h.ok)++
		}
		for _, c := range tlsle.ListLiveCerts() {
			if c.DaysLeft <= 21 {
				doctorPrintf("WARN le cert %s expires in %d days (%s)\n", c.Name, c.DaysLeft, c.NotAfter.Format("2006-01-02"))
				(*h.warn)++
			} else {
				doctorPrintf("OK   le cert %s valid ~%d days\n", c.Name, c.DaysLeft)
				(*h.ok)++
			}
		}

		if activeUnit("netductor-api") {
			h.check("api health :8787", h.curlOK("http://127.0.0.1:8787/health") || h.curlOKInsecure("https://127.0.0.1:8787/health"))
			h.warnCheck("api :8787 localhost only", listeningLocalhost("8787") && !listeningOnAll("8787"))
		}
		if h.exists("/etc/systemd/system/netductor-metrics.timer") {
			h.warnCheck("metrics timer", activeUnit("netductor-metrics.timer"))
			h.warnCheck("metrics latest.json", h.exists(filepath.Join(h.state, "metrics", "latest.json")))
		}
		// SCP peer removed — offsite is agent backup_pull
		h.warnCheck("tg admin id", h.exists(filepath.Join(h.etc, "secrets", "telegram_admin_id")) || os.Getenv("NETDUCTOR_TG_ADMIN") != "")
		if os.Getenv("CLAIM_FIRST") == "1" || os.Getenv("NETDUCTOR_TG_CLAIM_FIRST") == "1" {
			doctorPrintln(cli18n.T("doctor.claim_first"))
		}
		h.warnCheck("tg bot token", h.exists(filepath.Join(h.etc, "secrets", "telegram_bot_token")) || os.Getenv("NETDUCTOR_TG_TOKEN") != "")
		secReg := h.exists(paths.SecondaryDevicesFile())
		if secReg {
			doctorPrintln(cli18n.T("doctor.secondary_reg"))
		} else {
			h.warnCheck("secondary registry", false)
		}
		if listeningOnAll("8788") {
			doctorPrintln(cli18n.T("doctor.plain_8788_fail"))
			(*h.warn)++
		} else {
			doctorPrintln(cli18n.T("doctor.plain_8788_ok"))
			(*h.ok)++
		}
		if listeningOnAll("8790") {
			// Expected during short recovery arm (DR from new primary over WAN).
			doctorPrintln(cli18n.T("doctor.recovery_wan_armed"))
			(*h.warn)++
		} else if listeningLocalhost("8790") {
			doctorPrintln(cli18n.T("doctor.recovery_loopback"))
			(*h.ok)++
		}

		if mtls.ServerReady() {
			doctorPrintln(cli18n.T("doctor.mtls_server_ok"))
			(*h.ok)++
			if listeningOnAll(mtls.AgentTLSPort) {
				doctorPrintf(cli18n.T("doctor.mtls_listen_ok")+"\n", mtls.AgentTLSPort)
				(*h.ok)++
			} else {
				doctorPrintf(cli18n.T("doctor.mtls_listen_fail")+"\n", mtls.AgentTLSPort)
				(*h.warn)++
			}
		} else {
			doctorPrintln(cli18n.T("doctor.mtls_missing"))
			(*h.warn)++
		}
		// A5: removed env footguns must not linger
		footgunHit := false
		for _, e := range []string{"NETDUCTOR_API_ALLOW_PUBLIC", "NETDUCTOR_PLAIN_AGENT", "NETDUCTOR_API_PUBLIC", "NETDUCTOR_TRUST_PROXY"} {
			if strings.TrimSpace(os.Getenv(e)) != "" {
				doctorPrintf(cli18n.T("doctor.footgun_env")+"\n", e)
				(*h.warn)++
				footgunHit = true
			}
		}
		if !footgunHit {
			doctorPrintln(cli18n.T("doctor.footgun_clean"))
			(*h.ok)++
		}
		// cert expiry (CA/server/default client + per-node clients)
		for _, pc := range mtls.ListPlaneCerts() {
			if pc.DaysLeft < 0 {
				doctorPrintf(cli18n.T("doctor.mtls_expired")+"\n", pc.Name, pc.NotAfter.Format("2006-01-02"))
				(*h.warn)++
			} else if pc.DaysLeft <= 30 {
				doctorPrintf(cli18n.T("doctor.mtls_expires_soon")+"\n", pc.Name, pc.DaysLeft, pc.NotAfter.Format("2006-01-02"))
				(*h.warn)++
			} else {
				doctorPrintf(cli18n.T("doctor.mtls_valid")+"\n", pc.Name, pc.DaysLeft)
				(*h.ok)++
			}
		}
		if pend, err := mtls.PendingRotates(); err == nil && len(pend) > 0 {
			for _, p := range pend {
				doctorPrintf(cli18n.T("doctor.mtls_pending")+"\n", p.NodeID, p.OldSerial, p.NewSerial)
				(*h.warn)++
			}
		}
		if clients, err := mtls.ListClientCerts(); err == nil {
			for _, c := range clients {
				tag := "OK  "
				if c.Revoked {
					doctorPrintf(cli18n.T("doctor.mtls_client_revoked")+"\n", c.NodeID, c.Serial)
					(*h.warn)++
					continue
				}
				if c.DaysLeft <= 30 {
					tag = "WARN"
					(*h.warn)++
				} else {
					(*h.ok)++
				}
				doctorPrintf(cli18n.T("doctor.mtls_client_line")+"\n", tag, c.NodeID, c.DaysLeft, c.Serial)
			}
		}

		// self-host git + registry + isolated CI (optional components)
		if st, err := os.Stat(gitstore.Root()); err == nil && st.IsDir() {
			doctorPrintf("OK   git root %s\n", gitstore.Root())
			if tags, err := ndupdate.ListLocalTags(); err == nil {
				if len(tags) == 0 {
					doctorPrintf("INFO local releases empty (%s)\n", ndupdate.LocalReleasesDir())
				} else {
					doctorPrintf("OK   local releases %d (latest %s)\n", len(tags), tags[0])
					for _, line := range upstream.Check() {
						if strings.HasPrefix(line, "OK ") {
							doctorPrintln(line)
							(*h.ok)++
						} else if strings.HasPrefix(line, "WARN ") {
							doctorPrintln(line)
							(*h.warn)++
						} else {
							doctorPrintln(line)
						}
					}
					if st := ndupdate.ReadReleaseBuildStatus(); st.State != "" {
						doctorPrintf("INFO release build status=%s tag=%s at=%s\n", st.State, st.Tag, st.At)
					}
				}
			}
			(*h.ok)++
		} else {
			doctorPrintf("INFO git root absent (%s) — netductor git init <name>\n", gitstore.Root())
		}
		// ship jobs (PLAN-PROJECT-SHIP)
		if sc := gitstore.CountShipJobsByState(); len(sc) > 0 {
			doctorPrintf("INFO ship jobs")
			for k, v := range sc {
				doctorPrintf(" %s=%d", k, v)
			}
			doctorPrintf(" vps_slot=%v\n", gitstore.VpsSlotTaken())
		}
		doctorPrintf("INFO ci %s\n", ci.StatusLine())
		doctorPrintln("INFO " + gitstore.FormatCacheInfo())
		doctorPrintln("INFO " + gitstore.BuildHostStatusLine())
		doctorPrintln("INFO " + svcpaths.BuildPathsStatus())
		for _, s := range gitstore.ListBuildHostStates() {
			on := "offline"
			if s.Online {
				on = "online"
			}
			doctorPrintf("INFO build-host %s %s %s disk=%.1fG\n", s.ID, s.Label, on, s.DiskFreeGB)
		}
		if ci.IsolationEnabled() && ci.Engine() == "" {
			doctorPrintln("WARN ci isolation on but no docker/podman")
			(*h.warn)++
		} else if ci.IsolationEnabled() {
			doctorPrintln("OK   ci builds isolated in containers")
			(*h.ok)++
		} else {
			doctorPrintln("WARN ci NETDUCTOR_CI_HOST=1 (host builds)")
			(*h.warn)++
		}
		rst := registry.StatusInfo()
		if rst.OK {
			doctorPrintf("OK   registry %s (auth=%v)\n", rst.Addr, rst.Auth)
			(*h.ok)++
		} else if rst.Engine == "" {
			doctorPrintf("INFO registry skipped (no docker/podman)\n")
		} else {
			doctorPrintf("WARN registry not ready: %s\n", rst.Error)
			if rst.Error == "" {
				doctorPrintf("WARN registry not running — netductor registry ensure\n")
			}
			(*h.warn)++
		}

	case "secondary":
		h.check("vpn-users.json", h.exists(filepath.Join(h.etc, "vpn-users.json")))
		agentOK := activeUnit("netductor-secondary-agent")
		h.check("secondary agent", agentOK)
		if h.exists("/opt/netductor/lampac") || dirHasDockerLampac() {
			h.warnCheck("lampac container", dockerLampacHealthy())
			h.warnCheck("lampac localhost only", listeningLocalhost("9118") && !listeningOnAll("9118"))
		}
		if strings.HasPrefix(h.host, "nd-secondary") || h.host == "nd-secondary" {
			doctorPrintf(cli18n.T("doctor.hostname_ok")+"\n", h.host)
			(*h.ok)++
		} else {
			doctorPrintf(cli18n.T("doctor.hostname_warn")+"\n", h.host)
			(*h.warn)++
		}
		h.warnCheck("mtls client certs", mtls.ClientReady())
		if activeUnit("blocky") {
			doctorPrintln(cli18n.T("doctor.blocky_on_sec"))
			(*h.warn)++
		}

	default:
		h.warnCheck("READY.txt", h.exists(filepath.Join(h.etc, "READY.txt")))
		h.warnCheck("vpn-users.json", h.exists(filepath.Join(h.etc, "vpn-users.json")))
		h.warnCheck("blocky unit", activeUnit("blocky"))
		h.warnCheck("netductor-api", activeUnit("netductor-api"))
	}

	sniFile := filepath.Join(h.etc, "secrets", "singbox_reality_sni")
	sniVal := "api.vk.me"
	if b, err := os.ReadFile(sniFile); err == nil && strings.TrimSpace(string(b)) != "" {
		sniVal = strings.TrimSpace(string(b))
	}
	doctorPrintf(cli18n.T("doctor.sni")+"\n", sniVal)

	// NVR
	cfgN := nvr.LoadConfig()
	stN := nvr.GetStorageStatus()
	doctorPrintln()
	doctorPrintln(cli18n.T("doctor.nvr"))
	doctorPrintf(cli18n.T("doctor.nvr_path")+"\n", cfgN.Path)
	doctorPrintf(cli18n.T("doctor.nvr_storage")+"\n", stN.Exists, stN.Writable, stN.MountPoint, stN.FreeGB, stN.SegmentCnt, stN.SegmentGB)
	if stN.EncryptedHint != "" {
		doctorPrintf(cli18n.T("doctor.nvr_hint")+"\n", stN.EncryptedHint)
	}
	doctorPrintf(cli18n.T("doctor.nvr_record")+"\n",
		cfgN.RecordEnabled, cfgN.RetentionDays, cfgN.MaxGB, cfgN.MinFreeGB)
	if st, err := os.Stat(cfgN.Path); err != nil {
		doctorPrintf(cli18n.T("doctor.nvr_missing")+"\n", err)
	} else if !st.IsDir() {
		doctorPrintln(cli18n.T("doctor.nvr_notdir"))
	} else {
		files, _ := nvr.ListSegmentFiles(cfgN.Path)
		var sum int64
		for _, f := range files {
			sum += f.Size
		}
		doctorPrintf(cli18n.T("doctor.nvr_segments")+"\n", len(files), float64(sum)/(1024*1024*1024))
	}
	if r, ok := nvr.LastRetentionReport(); ok {
		doctorPrintf(cli18n.T("doctor.nvr_retention")+"\n", r.Deleted, r.Kept, r.At)
	}
	cams := nvr.ListCameras()
	doctorPrintf("INFO nvr cameras=%d\n", len(cams))
	{
		files, _ := nvr.ListSegmentFiles(cfgN.Path)
		if len(files) > 0 {
			newest := files[0].ModTime
			for _, f := range files {
				if f.ModTime.After(newest) {
					newest = f.ModTime
				}
			}
			age := time.Since(newest)
			doctorPrintf("INFO nvr last_segment_age=%s\n", age.Truncate(time.Second))
			if cfgN.RecordEnabled && age > 2*time.Hour {
				h.warnCheck("nvr segments stale (>2h)", false)
			}
		} else if cfgN.RecordEnabled && len(cams) > 0 {
			h.warnCheck("nvr has cameras but no segments", false)
		}
	}
	go2path := filepath.Join(paths.StateDir(), "nvr", "go2rtc.yaml")
	if st, err := os.Stat(go2path); err == nil {
		mode := st.Mode().Perm()
		if mode&0o077 != 0 {
			h.warnCheck("nvr go2rtc.yaml perms (want 0600)", false)
			doctorPrintf("WARN nvr go2rtc.yaml mode=%04o (may contain stream secrets)\n", mode)
		} else {
			doctorPrintf("OK   nvr go2rtc.yaml present mode=%04o\n", mode)
		}
	} else {
		doctorPrintf("INFO nvr go2rtc.yaml not written (netductor nvr go2rtc)\n")
	}
	secPath := filepath.Join(paths.StateDir(), "nvr", "secrets.json")
	if st, err := os.Stat(secPath); err == nil {
		mode := st.Mode().Perm()
		if mode&0o077 != 0 {
			h.warnCheck("nvr secrets.json perms (want 0600)", false)
		} else {
			doctorPrintf("OK   nvr secrets.json mode=%04o\n", mode)
		}
	}
	if nvr.Go2RTCListening() {
		h.check("nvr go2rtc :1984", true)
	} else {
		doctorPrintf("INFO nvr go2rtc not listening on 127.0.0.1:1984\n")
	}
	if stN.Exists && cfgN.MaxGB > 0 && stN.SegmentGB >= cfgN.MaxGB*0.9 {
		h.warnCheck("nvr segment usage ≥90% MaxGB", false)
	}
	if stN.Exists && cfgN.MinFreeGB > 0 && stN.FreeGB > 0 && stN.FreeGB < cfgN.MinFreeGB {
		h.warnCheck("nvr free < MinFreeGB", false)
	}

	// Sites / rooms inventory
	if list, err := sites.List(); err != nil {
		h.warnCheck("sites list", false)
	} else {
		doctorPrintf("INFO sites count=%d\n", len(list))
		roomsN := 0
		for _, s := range list {
			rs, _ := sites.ListRooms(s.ID)
			roomsN += len(rs)
		}
		doctorPrintf("INFO sites rooms total=%d\n", roomsN)
		sdir := filepath.Join(paths.StateDir(), "sites")
		if st, err := os.Stat(sdir); err != nil {
			doctorPrintf("INFO sites dir not yet created\n")
		} else if st.IsDir() {
			h.check("sites state dir", true)
		}
	}

	if h.role == "primary" {
		ch := channels.Collect()
		for _, s := range ch.Secondaries {
			if !s.Online {
				doctorPrintf("WARN channel %s offline (hb_age=%ds)\n", s.Name, s.HeartbeatAgeSec)
			} else if !s.TCP443OK {
				doctorPrintf("WARN channel %s tcp443 fail ip=%s\n", s.Name, s.PublicIP)
			} else if !s.UplinkOK {
				doctorPrintf("WARN channel %s uplink_ok=false\n", s.Name)
			} else {
				doctorPrintf("OK   channel %s online tcp443=%.0fms uplink=ok\n", s.Name, s.TCP443ms)
			}
		}
		if ch.RealityInvalidFromSec15m >= 200 {
			doctorPrintf("WARN reality invalid from secondary: %d/15m\n", ch.RealityInvalidFromSec15m)
		}
		if ch.MismatchLocal30m >= 25 {
			doctorPrintf("WARN flow mismatch local: %d/30m\n", ch.MismatchLocal30m)
		}
	}

	doctorPrintf(cli18n.T("doctor.summary")+"\n", *h.ok, *h.fail, *h.warn)
	// Service plane SP/PS (when secondary is expected)
	{
		st := svcpaths.Status()
		if st.KeysDirOK || st.SP.Up || st.PS.Up {
			spOK := st.SP.Up && st.SP.PeerPingOK != nil && *st.SP.PeerPingOK
			psOK := st.PS.Up && st.PS.PeerPingOK != nil && *st.PS.PeerPingOK
			if spOK {
				h.check("svc-paths SP peer", true)
			} else {
				h.warnCheck("svc-paths SP peer", false)
			}
			if psOK {
				h.check("svc-paths PS peer", true)
			} else {
				h.warnCheck("svc-paths PS peer", false)
			}
		}
	}
	// stack watchdog timer (optional orchestrator)
	if st, err := exec.Command("systemctl", "is-active", "netductor-stack-watchdog.timer").CombinedOutput(); err == nil && strings.TrimSpace(string(st)) == "active" {
		(*h.ok)++
		fmt.Println("OK   netductor-stack-watchdog.timer")
	} else {
		(*h.warn)++
		fmt.Println("WARN netductor-stack-watchdog.timer inactive (netductor stack watchdog-install)")
	}
	// DNS path note (client → VPN → blocky vs direct)
	lastDoctorReport.Checks = append(lastDoctorReport.Checks, DoctorCheck{
		ID:     "dns-path",
		Status: "info",
		Detail: "client → VPN → blocky (ads/YouTube filtering) or direct system DNS. PROXY path uses node blocky; DIRECT uses client DNS.",
	})
	doctorPrintln("INFO dns-path: client→VPN→blocky vs direct (see detail)")
	// VERSION vs latest GitHub tag (best-effort)
	ver := strings.TrimSpace(string(mustReadFile("VERSION")))
	if ver == "" {
		ver = version.Running()
	}
	latest := latestGitHubTag()
	if latest != "" && latest != ver && latest != "v"+ver {
		h.warnCheck("VERSION matches latest GitHub tag", false)
		lastDoctorReport.Checks = append(lastDoctorReport.Checks, DoctorCheck{
			ID: "version-tag", Status: "warn", Detail: "local=" + ver + " latest=" + latest,
		})
	} else if latest != "" {
		h.check("VERSION matches latest GitHub tag", true)
	}
}
