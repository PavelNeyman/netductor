package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/cli18n"
	"github.com/PavelNeyman/netductor/internal/hardening"
	"github.com/PavelNeyman/netductor/internal/ci"
	"github.com/PavelNeyman/netductor/internal/git"
	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/registry"
	"github.com/PavelNeyman/netductor/internal/nvr"
	"github.com/PavelNeyman/netductor/internal/paths"
)

func detectRole() string {
	etc := paths.EtcDir()
	if b, err := os.ReadFile(filepath.Join(etc, "role")); err == nil {
		s := strings.TrimSpace(strings.ToLower(string(b)))
		if s == "primary" || s == "core" {
			return "primary"
		}
		if s == "secondary" {
			return "secondary"
		}
	}
	if b, err := os.ReadFile(filepath.Join(etc, "READY.txt")); err == nil {
		s := strings.ToLower(string(b))
		if strings.Contains(s, "primary") || strings.Contains(s, "core") {
			return "primary"
		}
		if strings.Contains(s, "secondary") {
			return "secondary"
		}
	}
	host, _ := os.Hostname()
	hl := strings.ToLower(host)
	if strings.Contains(hl, "primary") || strings.HasPrefix(hl, "nd-core") {
		return "primary"
	}
	if strings.Contains(hl, "secondary") {
		return "secondary"
	}
	if activeUnit("netductor-secondary-agent") {
		return "secondary"
	}
	if activeUnit("netductor-api") || activeUnit("netductor-telegram-bot") {
		return "primary"
	}
	return "unknown"
}

func activeUnit(unit string) bool {
	out, _ := exec.Command("systemctl", "is-active", unit).Output()
	return strings.TrimSpace(string(out)) == "active"
}

func sshdT(key string) string {
	out, err := exec.Command("sshd", "-T").Output()
	if err != nil {
		return ""
	}
	key = strings.ToLower(key)
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(strings.ToLower(line))
		if len(f) >= 2 && f[0] == key {
			return f[1]
		}
	}
	return ""
}

func listeningOnAll(port string) bool {
	for _, flag := range []string{"-tlnp", "-ulnp"} {
		out, err := exec.Command("ss", flag).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.Contains(line, ":"+port) {
				continue
			}
			if strings.Contains(line, "*:"+port) || strings.Contains(line, "0.0.0.0:"+port) || strings.Contains(line, "[::]:"+port) {
				return true
			}
		}
	}
	return false
}

func listeningLocalhost(port string) bool {
	for _, flag := range []string{"-tlnp", "-ulnp"} {
		out, err := exec.Command("ss", flag).Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "127.0.0.1:"+port) || strings.Contains(line, "[::1]:"+port) {
				return true
			}
		}
	}
	return false
}


// DoctorCheck is one health line for CLI + JSON API.
type DoctorCheck struct {
	ID     string `json:"id"`
	Status string `json:"status"` // ok | fail | warn | info
	Detail string `json:"detail,omitempty"`
}

// DoctorReport is structured doctor output.
type DoctorReport struct {
	OK      bool          `json:"ok"`
	Role    string        `json:"role"`
	Host    string        `json:"host"`
	Checks  []DoctorCheck `json:"checks"`
	Summary struct {
		OK   int `json:"ok"`
		Fail int `json:"fail"`
		Warn int `json:"warn"`
	} `json:"summary"`
}

// CollectDoctor runs the same checks as CLI doctor and returns JSON-friendly report.
func CollectDoctor() DoctorReport {
	doctorQuiet = true
	defer func() { doctorQuiet = false }()
	code := runDoctorNative()
	// runDoctorNative fills lastDoctorReport
	r := lastDoctorReport
	r.OK = code == 0 && r.Summary.Fail == 0
	return r
}

var lastDoctorReport DoctorReport
var doctorQuiet bool

func doctorPrintf(format string, args ...any) {
	if doctorQuiet {
		return
	}
	doctorPrintf(format, args...)
}
func doctorPrintln(args ...any) {
	if doctorQuiet {
		return
	}
	doctorPrintln(args...)
}

func runDoctorNative() int {
	ok, fail, warn := 0, 0, 0
	lastDoctorReport = DoctorReport{}
	check := func(name string, good bool) {
		st := "ok"
		if good {
			doctorPrintf("%s %s\n", cli18n.T("doctor.ok"), name)
			ok++
		} else {
			doctorPrintf("%s %s\n", cli18n.T("doctor.fail"), name)
			fail++
			st = "fail"
		}
		lastDoctorReport.Checks = append(lastDoctorReport.Checks, DoctorCheck{ID: name, Status: st})
	}
	warnCheck := func(name string, good bool) {
		st := "ok"
		if good {
			doctorPrintf("%s %s\n", cli18n.T("doctor.ok"), name)
			ok++
		} else {
			doctorPrintf("%s %s\n", cli18n.T("doctor.warn"), name)
			warn++
			st = "warn"
		}
		lastDoctorReport.Checks = append(lastDoctorReport.Checks, DoctorCheck{ID: name, Status: st})
	}
	exists := func(p string) bool {
		_, err := os.Stat(p)
		return err == nil
	}
	curlOK := func(url string) bool {
		return exec.Command("curl", "-fsS", "--max-time", "2", "-o", "/dev/null", url).Run() == nil
	}
	curlOKInsecure := func(url string) bool {
		return exec.Command("curl", "-fskS", "--max-time", "2", "-o", "/dev/null", url).Run() == nil
	}

	host, _ := os.Hostname()
	role := detectRole()
	lastDoctorReport.Role = role
	lastDoctorReport.Host = host
	doctorPrintf(cli18n.T("doctor.header")+"\n", role, host)
	etc := paths.EtcDir()
	state := paths.StateDir()

	if b, err := os.ReadFile(filepath.Join(state, "installed_version")); err == nil {
		doctorPrintf(cli18n.T("doctor.installed_version")+"\n", strings.TrimSpace(string(b)))
	}
	check("debian", exists("/etc/debian_version"))
	check("secrets", exists(filepath.Join(etc, "secrets")))
	check("netductor", lookPath("netductor") != "")
	check("sing-box binary", exists("/usr/local/bin/sing-box"))
	check("sing-box unit", activeUnit("sing-box"))
	if exists("/usr/local/bin/sing-box") && exists("/usr/local/etc/sing-box/config.json") {
		check("sing-box config", exec.Command("/usr/local/bin/sing-box", "check", "-c", "/usr/local/etc/sing-box/config.json").Run() == nil)
		fi, err := os.Stat("/usr/local/etc/sing-box/config.json")
		warnCheck("sing-box config perms 600", err == nil && fi.Mode().Perm()&0o077 == 0)
	} else {
		check("sing-box config", false)
	}

	pa := sshdT("passwordauthentication")
	check("ssh passwordauth no", pa == "no")
	x11 := sshdT("x11forwarding")
	warnCheck("ssh x11forwarding no", x11 == "no" || x11 == "")
	wantPort := hardening.SSHPort()
	gotPort := sshdT("port")
	warnCheck(fmt.Sprintf("ssh port %d (got %s)", wantPort, gotPort), gotPort == fmt.Sprintf("%d", wantPort) || gotPort == "")
	if os.Getenv("NETDUCTOR_FAIL2BAN") != "0" {
		warnCheck("fail2ban active", activeUnit("fail2ban"))
	}
	warnCheck("no zabbix agent", !activeUnit("zabbix-agent") && !activeUnit("zabbix-agent2") && !listeningOnAll("10050"))

	switch role {
	case "primary":
		check("READY.txt", exists(filepath.Join(etc, "READY.txt")))
		check("vpn-users.json", exists(filepath.Join(etc, "vpn-users.json")))
		check("blocky unit", activeUnit("blocky"))
		if activeUnit("blocky") {
			if listeningOnAll("53") {
				doctorPrintln(cli18n.T("doctor.blocky_all"))
				warn++
			} else if listeningLocalhost("53") {
				doctorPrintln(cli18n.T("doctor.blocky_local"))
				ok++
			} else {
				doctorPrintln(cli18n.T("doctor.blocky_unknown"))
				warn++
			}
		}
		check("netductor-api", activeUnit("netductor-api"))
		// Telegram is an optional add-on — only require unit if installed
		if exists("/etc/systemd/system/netductor-telegram-bot.service") || exists("/lib/systemd/system/netductor-telegram-bot.service") {
			check("netductor-telegram-bot", activeUnit("netductor-telegram-bot"))
		} else {
			doctorPrintln(cli18n.T("doctor.tg_skip"))
		}
		// import redirect for TG deep-link buttons (port 80, already allowed for ACME)
		warnCheck("netductor-redirect unit", activeUnit("netductor-redirect"))
		if strings.TrimSpace(os.Getenv("NETDUCTOR_REDIRECT_BASE")) == "" {
			doctorPrintln(cli18n.T("doctor.redirect_unset"))
		} else {
			doctorPrintf(cli18n.T("doctor.redirect_ok")+"\n", os.Getenv("NETDUCTOR_REDIRECT_BASE"))
		}
		if activeUnit("netductor-redirect") || listeningOnAll("80") || listeningLocalhost("80") {
			if curlOK("http://127.0.0.1/healthz") {
				doctorPrintln(cli18n.T("doctor.redirect_health_ok"))
				ok++
			} else {
				doctorPrintln(cli18n.T("doctor.redirect_health_bad"))
				warn++
			}
		}
		if activeUnit("netductor-api") {
			check("api health :8787", curlOK("http://127.0.0.1:8787/health") || curlOKInsecure("https://127.0.0.1:8787/health"))
			warnCheck("api :8787 localhost only", listeningLocalhost("8787") && !listeningOnAll("8787"))
		}
		if exists("/etc/systemd/system/netductor-metrics.timer") {
			warnCheck("metrics timer", activeUnit("netductor-metrics.timer"))
			warnCheck("metrics latest.json", exists(filepath.Join(state, "metrics", "latest.json")))
		}
		// SCP peer removed — offsite is agent backup_pull
		warnCheck("tg admin id", exists(filepath.Join(etc, "secrets", "telegram_admin_id")) || os.Getenv("NETDUCTOR_TG_ADMIN") != "")
		if os.Getenv("CLAIM_FIRST") == "1" || os.Getenv("NETDUCTOR_TG_CLAIM_FIRST") == "1" {
			doctorPrintln(cli18n.T("doctor.claim_first"))
		}
		warnCheck("tg bot token", exists(filepath.Join(etc, "secrets", "telegram_bot_token")) || os.Getenv("NETDUCTOR_TG_TOKEN") != "")
		secReg := exists(paths.SecondaryDevicesFile())
		if secReg {
			doctorPrintln(cli18n.T("doctor.secondary_reg"))
		} else {
			warnCheck("secondary registry", false)
		}
		if listeningOnAll("8788") {
			doctorPrintln(cli18n.T("doctor.plain_8788_fail"))
			warn++
		} else {
			doctorPrintln(cli18n.T("doctor.plain_8788_ok"))
			ok++
		}
		if listeningOnAll("8790") {
			// Expected during short recovery arm (DR from new primary over WAN).
			doctorPrintln(cli18n.T("doctor.recovery_wan_armed"))
			warn++
		} else if listeningLocalhost("8790") {
			doctorPrintln(cli18n.T("doctor.recovery_loopback"))
			ok++
		}
		if mtls.ServerReady() {
			doctorPrintln(cli18n.T("doctor.mtls_server_ok"))
			ok++
			if listeningOnAll(mtls.AgentTLSPort) {
				doctorPrintf(cli18n.T("doctor.mtls_listen_ok")+"\n", mtls.AgentTLSPort)
				ok++
			} else {
				doctorPrintf(cli18n.T("doctor.mtls_listen_fail")+"\n", mtls.AgentTLSPort)
				warn++
			}
		} else {
			doctorPrintln(cli18n.T("doctor.mtls_missing"))
			warn++
		}
		// cert expiry (CA/server/default client + per-node clients)
		for _, pc := range mtls.ListPlaneCerts() {
			if pc.DaysLeft < 0 {
				doctorPrintf(cli18n.T("doctor.mtls_expired")+"\n", pc.Name, pc.NotAfter.Format("2006-01-02"))
				warn++
			} else if pc.DaysLeft <= 30 {
				doctorPrintf(cli18n.T("doctor.mtls_expires_soon")+"\n", pc.Name, pc.DaysLeft, pc.NotAfter.Format("2006-01-02"))
				warn++
			} else {
				doctorPrintf(cli18n.T("doctor.mtls_valid")+"\n", pc.Name, pc.DaysLeft)
				ok++
			}
		}
		if pend, err := mtls.PendingRotates(); err == nil && len(pend) > 0 {
			for _, p := range pend {
				doctorPrintf(cli18n.T("doctor.mtls_pending")+"\n", p.NodeID, p.OldSerial, p.NewSerial)
				warn++
			}
		}
		if clients, err := mtls.ListClientCerts(); err == nil {
			for _, c := range clients {
				tag := "OK  "
				if c.Revoked {
					doctorPrintf(cli18n.T("doctor.mtls_client_revoked")+"\n", c.NodeID, c.Serial)
					warn++
					continue
				}
				if c.DaysLeft <= 30 {
					tag = "WARN"
					warn++
				} else {
					ok++
				}
				doctorPrintf(cli18n.T("doctor.mtls_client_line")+"\n", tag, c.NodeID, c.DaysLeft, c.Serial)
			}
		}

		// self-host git + registry + isolated CI (optional components)
		if st, err := os.Stat(gitstore.Root()); err == nil && st.IsDir() {
			doctorPrintf("OK   git root %s\n", gitstore.Root())
			ok++
		} else {
			doctorPrintf("INFO git root absent (%s) — netductor git init <name>\n", gitstore.Root())
		}
		doctorPrintf("INFO ci %s\n", ci.StatusLine())
		if ci.IsolationEnabled() && ci.Engine() == "" {
			doctorPrintln("WARN ci isolation on but no docker/podman")
			warn++
		} else if ci.IsolationEnabled() {
			doctorPrintln("OK   ci builds isolated in containers")
			ok++
		} else {
			doctorPrintln("WARN ci NETDUCTOR_CI_HOST=1 (host builds)")
			warn++
		}
		rst := registry.StatusInfo()
		if rst.OK {
			doctorPrintf("OK   registry %s (auth=%v)\n", rst.Addr, rst.Auth)
			ok++
		} else if rst.Engine == "" {
			doctorPrintf("INFO registry skipped (no docker/podman)\n")
		} else {
			doctorPrintf("WARN registry not ready: %s\n", rst.Error)
			if rst.Error == "" {
				doctorPrintf("WARN registry not running — netductor registry ensure\n")
			}
			warn++
		}

	case "secondary":
		check("vpn-users.json", exists(filepath.Join(etc, "vpn-users.json")))
		agentOK := activeUnit("netductor-secondary-agent")
		check("secondary agent", agentOK)
		if exists("/opt/netductor/lampac") || dirHasDockerLampac() {
			warnCheck("lampac container", dockerLampacHealthy())
			warnCheck("lampac localhost only", listeningLocalhost("9118") && !listeningOnAll("9118"))
		}
		if strings.HasPrefix(host, "nd-secondary") || host == "nd-secondary" {
			doctorPrintf(cli18n.T("doctor.hostname_ok")+"\n", host)
			ok++
		} else {
			doctorPrintf(cli18n.T("doctor.hostname_warn")+"\n", host)
			warn++
		}
		warnCheck("mtls client certs", mtls.ClientReady())
		if activeUnit("blocky") {
			doctorPrintln(cli18n.T("doctor.blocky_on_sec"))
			warn++
		}

	default:
		warnCheck("READY.txt", exists(filepath.Join(etc, "READY.txt")))
		warnCheck("vpn-users.json", exists(filepath.Join(etc, "vpn-users.json")))
		warnCheck("blocky unit", activeUnit("blocky"))
		warnCheck("netductor-api", activeUnit("netductor-api"))
	}

	sniFile := filepath.Join(etc, "secrets", "singbox_reality_sni")
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

	doctorPrintf(cli18n.T("doctor.summary")+"\n", ok, fail, warn)
	lastDoctorReport.Summary.OK = ok
	lastDoctorReport.Summary.Fail = fail
	lastDoctorReport.Summary.Warn = warn
	lastDoctorReport.OK = fail == 0
	if fail > 0 {
		return 1
	}

	return 0
}

func dirHasDockerLampac() bool {
	return exec.Command("docker", "ps", "-q", "-f", "name=netductor-lampac").Run() == nil
}

func dockerLampacHealthy() bool {
	out, err := exec.Command("docker", "inspect", "-f", "{{.State.Health.Status}}", "netductor-lampac").Output()
	if err != nil {
		return exec.Command("docker", "ps", "-q", "-f", "name=netductor-lampac", "-f", "status=running").Run() == nil
	}
	s := strings.TrimSpace(string(out))
	return s == "healthy" || s == ""
}
