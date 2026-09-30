package vpn

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// RuDirectSuffixes — domains that must NOT exit via foreign primary.
func RuDirectSuffixes() []string {
	return []string{
		"ru", "su", "xn--p1ai", "xn--p1acf",
		"vk.com", "vk.ru", "vk.me", "userapi.com", "vkuservideo.net", "vk-cdn.net",
		"mail.ru", "imgsmail.ru", "ok.ru", "odnoklassniki.ru", "mycdn.me",
		"yandex.ru", "yandex.net", "yandex.com", "ya.ru", "yastatic.net", "yandex.cloud",
		"yandex.by", "yandex.kz", "auto.ru",
		"gosuslugi.ru", "gu-st.ru", "pos.gosuslugi.ru", "gosuslugi-online.ru",
		"mos.ru", "mosreg.ru", "nalog.ru", "cbr.ru", "gov.ru", "government.ru",
		"emias.ru", "rzd.ru", "pochta.ru", "sfr.gov.ru", "pfr.gov.ru",
		"edu.ru", "edu.gov.ru", "myschool.edu.ru", "gup.education",
		"sberbank.ru", "sber.ru", "sberbank.com", "tinkoff.ru", "tbank.ru",
		"vtb.ru", "alfabank.ru", "gazprombank.ru", "open.ru", "raiffeisen.ru",
		"nspk.ru", "mironline.ru", "sbp.nspk.ru",
		"wildberries.ru", "wb.ru", "wbbasket.ru", "ozon.ru", "ozonusercontent.com",
		"avito.ru", "dns-shop.ru", "citilink.ru", "mvideo.ru", "lamoda.ru",
		"mts.ru", "megafon.ru", "beeline.ru", "tele2.ru", "yota.ru", "t2.ru",
		"2gis.com", "2gis.ru", "rutube.ru", "ivi.ru", "kinopoisk.ru", "hd.kinopoisk.ru",
		"okko.tv", "wink.ru",
	}
}

func RuDirectKeywords() []string {
	return []string{
		"gosuslugi", "esia", "emias", "sberbank", "sber", "tinkoff", "tbank", "wildberries", "ozon",
	}
}

func ShadowrocketRuDirectRules() []string {
	var lines []string
	lines = append(lines, "# netductor RU/gov direct — keep foreign exit off these")
	for _, k := range RuDirectKeywords() {
		lines = append(lines, "DOMAIN-KEYWORD,"+k+",DIRECT")
	}
	for _, s := range RuDirectSuffixes() {
		lines = append(lines, "DOMAIN-SUFFIX,"+s+",DIRECT")
	}
	lines = append(lines, "GEOIP,RU,DIRECT")
	return lines
}

// AdminNodeDirectCIDRs returns public IPs of primary + secondaries as /32 CIDRs.
func AdminNodeDirectCIDRs() []string {
	seen := map[string]bool{}
	var out []string
	add := func(ip string) {
		ip = strings.TrimSpace(ip)
		ip = strings.TrimSuffix(ip, "/32")
		if ip == "" || seen[ip] {
			return
		}
		if net.ParseIP(ip) == nil {
			return
		}
		seen[ip] = true
		out = append(out, ip+"/32")
	}
	// explicit override: comma/space separated
	if v := os.Getenv("NETDUCTOR_ADMIN_DIRECT_IPS"); v != "" {
		for _, p := range strings.FieldsFunc(v, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' }) {
			add(p)
		}
	}
	// file override (one IP per line)
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "admin-direct-ips")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			add(line)
		}
	}
	for _, rel := range []string{
		"public_ip",
		filepath.Join("secrets", "public_ip"),
	} {
		if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), rel)); err == nil {
			add(strings.Split(strings.TrimSpace(string(b)), "\n")[0])
		}
	}
	// secondary devices
	type dev struct {
		PublicIP string `json:"public_ip"`
	}
	type reg struct {
		Devices []dev `json:"devices"`
	}
	if b, err := os.ReadFile(filepath.Join(paths.StateDir(), "secondary", "devices.json")); err == nil {
		var r reg
		if json.Unmarshal(b, &r) == nil {
			for _, d := range r.Devices {
				add(d.PublicIP)
			}
		}
	}
	// nodes registry (if present)
	type node struct {
		PublicIP string `json:"public_ip"`
		Role     string `json:"role"`
	}
	if b, err := os.ReadFile(filepath.Join(paths.StateDir(), "nodes", "nodes.json")); err == nil {
		var nodes []node
		if json.Unmarshal(b, &nodes) == nil {
			for _, n := range nodes {
				add(n.PublicIP)
			}
		} else {
			var wrap struct {
				Nodes []node `json:"nodes"`
			}
			if json.Unmarshal(b, &wrap) == nil {
				for _, n := range wrap.Nodes {
					add(n.PublicIP)
				}
			}
		}
	}
	return out
}

func ShadowrocketAdminDirectRules() []string {
	var lines []string
	cidrs := AdminNodeDirectCIDRs()
	if len(cidrs) == 0 {
		return lines
	}
	lines = append(lines, "# netductor nodes — admin SSH/API must not hairpin via PROXY")
	for _, c := range cidrs {
		lines = append(lines, "IP-CIDR,"+c+",DIRECT")
	}
	return lines
}

// Default remote RULE-SET URLs (community-maintained domain lists for Shadowrocket).
// RULE-SET = "load this list of domains/IPs from URL and apply policy" — not our VPS IPs.
func defaultRemoteRuleSetDIRECT() []string {
	// Only lists that must stay on home ISP (DIRECT).
	// Do NOT include domains_community.list — that list is YouTube/Instagram/AI/etc.
	// and is meant for PROXY in community configs, not DIRECT.
	return []string{
		"https://cdn.jsdelivr.net/gh/misha-tgshv/shadowrocket-configuration-file@main/rules/domains_banking.list",
		"https://cdn.jsdelivr.net/gh/misha-tgshv/shadowrocket-configuration-file@main/rules/domains_ipchecker.list",
	}
}

// ShadowrocketRemoteRuleSetRules returns RULE-SET,…,DIRECT lines when enabled.
// Enable: NETDUCTOR_SR_REMOTE_RULESETS=1 or /etc/netductor/sr-remote-rulesets (one URL per line).
// Disable explicitly: NETDUCTOR_SR_REMOTE_RULESETS=0
func ShadowrocketRemoteRuleSetRules() []string {
	mode := strings.TrimSpace(os.Getenv("NETDUCTOR_SR_REMOTE_RULESETS"))
	var urls []string
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "sr-remote-rulesets")); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			urls = append(urls, line)
		}
	}
	switch mode {
	case "0", "false", "off", "no":
		return nil
	case "1", "true", "on", "yes":
		if len(urls) == 0 {
			urls = defaultRemoteRuleSetDIRECT()
		}
	default:
		// default ON with built-in list if no file; file alone also enables
		if len(urls) == 0 {
			urls = defaultRemoteRuleSetDIRECT()
		}
	}
	var lines []string
	lines = append(lines, "# remote RULE-SET (domain lists fetched by Shadowrocket) → DIRECT")
	for _, u := range urls {
		lines = append(lines, "RULE-SET,"+u+",DIRECT")
	}
	return lines
}

// ShadowrocketGeneralBlock — [General] tuned for RU dual-stack ops (inspired by community confs).
func ShadowrocketGeneralBlock() string {
	return strings.TrimSpace(`
[General]
# system DNS for DIRECT; PROXY DNS handled by node
dns-server = system
fallback-dns-server = system
ipv6 = false
prefer-ipv6 = false
private-ip-answer = true
# LAN + Apple captive stay off PROXY path
skip-proxy = 127.0.0.1, 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, localhost, *.local, captive.apple.com
tun-excluded-routes = 10.0.0.0/8,100.64.0.0/10,127.0.0.0/8,169.254.0.0/16,172.16.0.0/12,192.0.0.0/24,192.168.0.0/16,224.0.0.0/4,255.255.255.255/32
icmp-auto-reply = false
udp-policy-not-supported-behaviour = REJECT
`) + "\n"
}
