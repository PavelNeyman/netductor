package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/paths"
)

func detectRole() string {
	etc := paths.EtcDir()
	if b, err := os.ReadFile(filepath.Join(etc, "role")); err == nil {
		s := strings.TrimSpace(strings.ToLower(string(b)))
		if s == "primary" || s == "core" {
			return "primary"
		}
		if s == "secondary" || s == "relay" {
			return "secondary"
		}
	}
	if b, err := os.ReadFile(filepath.Join(etc, "READY.txt")); err == nil {
		s := strings.ToLower(string(b))
		if strings.Contains(s, "primary") || strings.Contains(s, "core") {
			return "primary"
		}
		if strings.Contains(s, "secondary") || strings.Contains(s, "relay") {
			return "secondary"
		}
	}
	host, _ := os.Hostname()
	hl := strings.ToLower(host)
	if strings.Contains(hl, "primary") || strings.HasPrefix(hl, "nd-core") {
		return "primary"
	}
	if strings.Contains(hl, "secondary") || strings.Contains(hl, "relay") {
		return "secondary"
	}
	if activeUnit("netductor-secondary-agent") || activeUnit("netductor-relay-agent") {
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

func runDoctorNative() int {
	ok, fail, warn := 0, 0, 0
	check := func(name string, good bool) {
		if good {
			fmt.Printf("OK   %s\n", name)
			ok++
		} else {
			fmt.Printf("FAIL %s\n", name)
			fail++
		}
	}
	warnCheck := func(name string, good bool) {
		if good {
			fmt.Printf("OK   %s\n", name)
			ok++
		} else {
			fmt.Printf("WARN %s\n", name)
			warn++
		}
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
	fmt.Printf("Netductor doctor — %s (role=%s)\n", host, role)
	etc := paths.EtcDir()
	state := paths.StateDir()

	if b, err := os.ReadFile(filepath.Join(state, "installed_version")); err == nil {
		fmt.Printf("INFO installed_version=%s\n", strings.TrimSpace(string(b)))
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
	warnCheck("no zabbix agent", !activeUnit("zabbix-agent") && !activeUnit("zabbix-agent2") && !listeningOnAll("10050"))

	switch role {
	case "primary":
		check("READY.txt", exists(filepath.Join(etc, "READY.txt")))
		check("vpn-users.json", exists(filepath.Join(etc, "vpn-users.json")))
		check("blocky unit", activeUnit("blocky"))
		if activeUnit("blocky") {
			if listeningOnAll("53") {
				fmt.Printf("WARN blocky :53 on 0.0.0.0 (prefer 127.0.0.1)\n")
				warn++
			} else if listeningLocalhost("53") {
				fmt.Printf("OK   blocky :53 localhost only\n")
				ok++
			} else {
				fmt.Printf("WARN blocky :53 listen mode unknown\n")
				warn++
			}
		}
		check("netductor-api", activeUnit("netductor-api"))
		check("netductor-telegram-bot", activeUnit("netductor-telegram-bot"))
		// import redirect for TG deep-link buttons (port 80, already allowed for ACME)
		warnCheck("netductor-redirect unit", activeUnit("netductor-redirect"))
		if activeUnit("netductor-redirect") || listeningOnAll("80") || listeningLocalhost("80") {
			if curlOK("http://127.0.0.1/healthz") {
				fmt.Printf("OK   redirect :80 /healthz\n")
				ok++
			} else {
				fmt.Printf("WARN redirect :80 /healthz not OK\n")
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
		warnCheck("backup.offsite peer", exists(filepath.Join(etc, "backup.offsite")))
		warnCheck("tg admin id", exists(filepath.Join(etc, "secrets", "telegram_admin_id")) || os.Getenv("NETDUCTOR_TG_ADMIN") != "")
		warnCheck("tg bot token", exists(filepath.Join(etc, "secrets", "telegram_bot_token")) || os.Getenv("NETDUCTOR_TG_TOKEN") != "")
		secReg := exists(filepath.Join(state, "secondary", "devices.json")) || exists(filepath.Join(state, "relay", "devices.json"))
		if secReg {
			fmt.Printf("INFO secondary registry present\n")
		} else {
			warnCheck("secondary registry", false)
		}
		if listeningOnAll("8788") {
			fmt.Printf("INFO agent plane :8788 plain (legacy)\n")
		}
		if mtls.ServerReady() {
			fmt.Printf("OK   mtls server certs\n")
			ok++
			if listeningOnAll(mtls.AgentTLSPort) {
				fmt.Printf("OK   agent plane mTLS :%s\n", mtls.AgentTLSPort)
				ok++
			} else {
				fmt.Printf("WARN agent plane mTLS :%s not listening\n", mtls.AgentTLSPort)
				warn++
			}
		} else {
			fmt.Printf("WARN mtls server certs missing\n")
			warn++
		}

	case "secondary":
		check("vpn-users.json", exists(filepath.Join(etc, "vpn-users.json")))
		agentOK := activeUnit("netductor-secondary-agent") || activeUnit("netductor-relay-agent")
		check("secondary agent", agentOK)
		if exists("/opt/netductor/lampac") || dirHasDockerLampac() {
			warnCheck("lampac container", dockerLampacHealthy())
			warnCheck("lampac localhost only", listeningLocalhost("9118") && !listeningOnAll("9118"))
		}
		if strings.HasPrefix(host, "nd-secondary") || host == "nd-secondary" {
			fmt.Printf("OK   hostname=%s\n", host)
			ok++
		} else {
			fmt.Printf("WARN hostname=%s (expected nd-secondary*)\n", host)
			warn++
		}
		warnCheck("mtls client certs", mtls.ClientReady())
		if activeUnit("blocky") {
			fmt.Printf("WARN blocky running on secondary (usually primary-only)\n")
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
	fmt.Printf("INFO reality_sni=%s\n", sniVal)
	fmt.Printf("\nSummary: ok=%d fail=%d warn=%d\n", ok, fail, warn)
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

func bindAllInterfaces(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsUnspecified()
}
