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
// Used by client profiles and secondary relay routing.
// Matching is domain_suffix (gosuslugi.ru matches *.gosuslugi.ru).
func RuDirectSuffixes() []string {
	return []string{
		// TLDs
		"ru", "su", "xn--p1ai", "xn--p1acf",
		// VK / Mail
		"vk.com", "vk.ru", "vk.me", "userapi.com", "vkuservideo.net", "vk-cdn.net",
		"mail.ru", "imgsmail.ru", "ok.ru", "odnoklassniki.ru", "mycdn.me",
		// Yandex
		"yandex.ru", "yandex.net", "yandex.com", "ya.ru", "yastatic.net", "yandex.cloud",
		"yandex.by", "yandex.kz", "auto.ru",
		// State / ESIA / regions
		"gosuslugi.ru", "gu-st.ru", "pos.gosuslugi.ru", "gosuslugi-online.ru",
		"mos.ru", "mosreg.ru", "nalog.ru", "cbr.ru", "gov.ru", "government.ru",
		"emias.ru", "rzd.ru", "pochta.ru", "sfr.gov.ru", "pfr.gov.ru",
		"edu.ru", "edu.gov.ru", "myschool.edu.ru", "gup.education",
		// Banks
		"sberbank.ru", "sber.ru", "sberbank.com", "tinkoff.ru", "tbank.ru",
		"vtb.ru", "alfabank.ru", "gazprombank.ru", "open.ru", "raiffeisen.ru",
		"nspk.ru", "mironline.ru", "sbp.nspk.ru",
		// Marketplaces / retail
		"wildberries.ru", "wb.ru", "wbbasket.ru", "ozon.ru", "ozonusercontent.com",
		"avito.ru", "dns-shop.ru", "citilink.ru", "mvideo.ru", "lamoda.ru",
		// Telco
		"mts.ru", "megafon.ru", "beeline.ru", "tele2.ru", "yota.ru", "t2.ru",
		// Maps / media
		"2gis.com", "2gis.ru", "rutube.ru", "ivi.ru", "kinopoisk.ru", "hd.kinopoisk.ru",
		"okko.tv", "wink.ru",
	}
}

// RuDirectKeywords — domain_keyword match (covers odd subdomains / non-.ru hosts).
func RuDirectKeywords() []string {
	return []string{
		"gosuslugi",
		"esia",
		"emias",
		"sberbank",
		"sber",
		"tinkoff",
		"tbank",
		"wildberries",
		"ozon",
	}
}

// ShadowrocketRuDirectRules returns [Rule] lines for Shadowrocket (before FINAL).
func ShadowrocketRuDirectRules() []string {
	var lines []string
	lines = append(lines, "# netductor RU/gov direct — keep foreign exit off these")
	for _, k := range RuDirectKeywords() {
		lines = append(lines, "DOMAIN-KEYWORD,"+k+",DIRECT")
	}
	for _, s := range RuDirectSuffixes() {
		// Shadowrocket: DOMAIN-SUFFIX,ru matches *.ru
		lines = append(lines, "DOMAIN-SUFFIX,"+s+",DIRECT")
	}
	lines = append(lines, "GEOIP,RU,DIRECT")
	return lines
}


// AdminNodeDirectCIDRs returns public IPs of primary + secondaries as /32 CIDRs.
// Used so admin SSH/API to the VPS does not hairpin through the VPN (client → VPS → VPS).
func AdminNodeDirectCIDRs() []string {
	seen := map[string]bool{}
	var out []string
	add := func(ip string) {
		ip = strings.TrimSpace(ip)
		if ip == "" || seen[ip] {
			return
		}
		if net.ParseIP(ip) == nil {
			return
		}
		// only global unicast public-ish; still allow any parsed IP for operator VPS
		seen[ip] = true
		out = append(out, ip+"/32")
	}
	// primary public IP (several historical locations)
	for _, rel := range []string{
		"public_ip",
		filepath.Join("secrets", "public_ip"),
	} {
		b, err := os.ReadFile(filepath.Join(paths.EtcDir(), rel))
		if err == nil {
			add(strings.Split(strings.TrimSpace(string(b)), "\n")[0])
		}
	}
	// registered secondaries (avoid import cycle with secondary → vpn)
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
	return out
}

// ShadowrocketAdminDirectRules — IP-CIDR DIRECT lines (must be first in [Rule]).
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
