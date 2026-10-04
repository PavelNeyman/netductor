package vpn

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SR Config profile IDs (generator + TG/UI).
const (
	SRProfileOperatorMobile    = "operator-mobile"
	SRProfileOperatorFullProxy = "operator-fullproxy"
	SRProfileFamily            = "family"
)

// Defaults aligned with live topology (override via files/env).
const (
	defaultHomeLAN     = "10.9.8.0/24"
	defaultServiceNet  = "10.88.0.0/24" // servicenet.CIDR — keep in sync
	defaultShopCIDR    = "10.120.0.0/16"
)

// SRProfileMeta is UI-facing metadata (EN/RU labels + short help).
type SRProfileMeta struct {
	ID       string
	LabelEN  string
	LabelRU  string
	HelpEN   string
	HelpRU   string
	Operator bool // only for operator/work user in TG
}

// SRProfiles returns ordered profiles for UI.
func SRProfiles() []SRProfileMeta {
	return []SRProfileMeta{
		{
			ID:      SRProfileOperatorMobile,
			LabelEN: "Operator · mobile",
			LabelRU: "Оператор · мобильный",
			HelpEN:  "Cellular / foreign Wi‑Fi: RU+banks DIRECT, work+home DIRECT, ads REJECT, service-net PROXY, else VLESS",
			HelpRU:  "Сотовая / чужой Wi‑Fi: RU+банки DIRECT, работа+дом DIRECT, реклама REJECT, service-net PROXY, остальное VLESS",
			Operator: true,
		},
		{
			ID:      SRProfileOperatorFullProxy,
			LabelEN: "Operator · full proxy",
			LabelRU: "Оператор · весь трафик в VLESS",
			HelpEN:  "Foreign Wi‑Fi when you want almost all via secondary: work+home+nodes DIRECT, no client RU DIRECT (split on node), service-net PROXY",
			HelpRU:  "Чужой Wi‑Fi «всё в VLESS»: работа+дом+ноды DIRECT, без RU DIRECT на клиенте (split на secondary), service-net PROXY",
			Operator: true,
		},
		{
			ID:      SRProfileFamily,
			LabelEN: "Family",
			LabelRU: "Семья",
			HelpEN:  "One config for cellular and Wi‑Fi: RU+banks DIRECT, ads REJECT, FINAL PROXY. No work/shop/service-net",
			HelpRU:  "Один конфиг для LTE и Wi‑Fi: RU+банки DIRECT, реклама REJECT, FINAL PROXY. Без work/магазинов/service-net",
			Operator: false,
		},
	}
}

// NormalizeSRProfile maps aliases to canonical IDs.
func NormalizeSRProfile(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch p {
	case "", "default", "operator", "mobile", "op-mobile", "operator-mobile":
		return SRProfileOperatorMobile
	case "full", "fullproxy", "operator-full", "op-full", "operator-fullproxy":
		return SRProfileOperatorFullProxy
	case "family", "simple", "home":
		return SRProfileFamily
	default:
		return SRProfileOperatorMobile
	}
}

// BuildShadowrocketRoutingConfProfile returns full SR Config text for profile.
func BuildShadowrocketRoutingConfProfile(profile string) string {
	profile = NormalizeSRProfile(profile)
	var rb strings.Builder
	rb.WriteString("# netductor SR Config profile=" + profile + "\n")
	rb.WriteString("# Shadowrocket → Config → import · Global Routing = Config\n")
	rb.WriteString("# VLESS server: add separately (prefer secondary). Apple TV: no SR, VLESS only.\n\n")
	rb.WriteString(ShadowrocketGeneralBlockFor(profile))
	rb.WriteString("\n[Rule]\n")

	// Nodes always DIRECT (avoid hairpin to entry).
	for _, line := range ShadowrocketAdminDirectRules() {
		rb.WriteString(line)
		rb.WriteByte('\n')
	}

	switch profile {
	case SRProfileOperatorMobile:
		writeCIDRRules(&rb, "# home LAN", homeLANCIDRs(), "DIRECT")
		writeCIDRRules(&rb, "# work VPN / corporate (utun) — do not send via VLESS", workDirectCIDRs(), "DIRECT")
		for _, line := range ShadowrocketAdsRejectRules() {
			rb.WriteString(line)
			rb.WriteByte('\n')
		}
		for _, line := range ShadowrocketRemoteRuleSetRules() {
			rb.WriteString(line)
			rb.WriteByte('\n')
		}
		for _, line := range ShadowrocketRuDirectRules() {
			rb.WriteString(line)
			rb.WriteByte('\n')
		}
		writeCIDRRules(&rb, "# service-net (lampac/git/registry/nvr on primary)", serviceNetCIDRs(), "PROXY")
		writeCIDRRules(&rb, "# shop / site overlays (when routed)", shopCIDRs(), "PROXY")
	case SRProfileOperatorFullProxy:
		writeCIDRRules(&rb, "# home LAN", homeLANCIDRs(), "DIRECT")
		writeCIDRRules(&rb, "# work VPN / corporate — required on any Wi‑Fi", workDirectCIDRs(), "DIRECT")
		// No client RU DIRECT — secondary split+DNS handles RU for tunnel traffic.
		// REJECT ads optional; keep light list (duplicates blocky for non-PROXY edge cases).
		for _, line := range ShadowrocketAdsRejectRules() {
			rb.WriteString(line)
			rb.WriteByte('\n')
		}
		writeCIDRRules(&rb, "# service-net", serviceNetCIDRs(), "PROXY")
		writeCIDRRules(&rb, "# shop overlays", shopCIDRs(), "PROXY")
	case SRProfileFamily:
		for _, line := range ShadowrocketAdsRejectRules() {
			rb.WriteString(line)
			rb.WriteByte('\n')
		}
		for _, line := range ShadowrocketRemoteRuleSetRules() {
			rb.WriteString(line)
			rb.WriteByte('\n')
		}
		for _, line := range ShadowrocketRuDirectRules() {
			rb.WriteString(line)
			rb.WriteByte('\n')
		}
	}

	rb.WriteString("FINAL,PROXY\n")
	return rb.String()
}

func writeCIDRRules(rb *strings.Builder, comment string, cidrs []string, policy string) {
	if len(cidrs) == 0 {
		return
	}
	if comment != "" {
		rb.WriteString(comment)
		rb.WriteByte('\n')
	}
	for _, c := range cidrs {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		rb.WriteString("IP-CIDR,")
		rb.WriteString(c)
		rb.WriteByte(',')
		rb.WriteString(policy)
		rb.WriteByte('\n')
	}
}

// homeLANCIDRs — default 10.9.8.0/24; override NETDUCTOR_HOME_LAN or /etc/netductor/home-lan.
func homeLANCIDRs() []string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_HOME_LAN")); v != "" {
		return splitCIDRList(v)
	}
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "home-lan")); err == nil {
		var out []string
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			out = append(out, line)
		}
		if len(out) > 0 {
			return out
		}
	}
	return []string{defaultHomeLAN}
}

// workDirectCIDRs from NETDUCTOR_WORK_DIRECT_CIDRS or /etc/netductor/work-direct-cidrs.
func workDirectCIDRs() []string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_WORK_DIRECT_CIDRS")); v != "" {
		return splitCIDRList(v)
	}
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "work-direct-cidrs")); err == nil {
		var out []string
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			out = append(out, line)
		}
		return out
	}
	return nil
}

func serviceNetCIDRs() []string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_SERVICE_NET")); v != "" {
		return splitCIDRList(v)
	}
	return []string{defaultServiceNet}
}

func shopCIDRs() []string {
	if v := strings.TrimSpace(os.Getenv("NETDUCTOR_SHOP_CIDRS")); v != "" {
		return splitCIDRList(v)
	}
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "shop-cidrs")); err == nil {
		var out []string
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			out = append(out, line)
		}
		if len(out) > 0 {
			return out
		}
	}
	return []string{defaultShopCIDR}
}

func splitCIDRList(v string) []string {
	var out []string
	for _, p := range strings.FieldsFunc(v, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\n' || r == ';'
	}) {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// ShadowrocketAdsRejectRules — client-side ad domain lists (work while SR ON, incl. DIRECT path).
func ShadowrocketAdsRejectRules() []string {
	urls := []string{
		"https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Shadowrocket/Advertising/Advertising.list",
		"https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Shadowrocket/AdvertisingLite/AdvertisingLite.list",
	}
	// Allow override file
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "sr-ads-rulesets")); err == nil {
		var custom []string
		for _, line := range strings.Split(string(b), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			custom = append(custom, line)
		}
		if len(custom) > 0 {
			urls = custom
		}
	}
	if mode := strings.TrimSpace(os.Getenv("NETDUCTOR_SR_ADS_REJECT")); mode == "0" || mode == "off" || mode == "false" {
		return nil
	}
	lines := []string{"# ads RULE-SET → REJECT (client; works on DIRECT while SR ON)"}
	for _, u := range urls {
		lines = append(lines, "RULE-SET,"+u+",REJECT")
	}
	return lines
}

// ShadowrocketGeneralBlockFor — tighter skip-proxy than full 10/8 (home LAN only + RFC1918 still in tun-excluded for TUN).
func ShadowrocketGeneralBlockFor(profile string) string {
	home := defaultHomeLAN
	if cidrs := homeLANCIDRs(); len(cidrs) > 0 {
		home = cidrs[0]
	}
	// skip-proxy: loopback + home LAN + Apple captive — not entire 10/8 (work/service-net must be rule-driven).
	skip := fmt.Sprintf("127.0.0.1, %s, localhost, *.local, captive.apple.com", home)
	_ = profile
	return strings.TrimSpace(`
[General]
# DIRECT uses system DNS; PROXY path DNS is on the node (blocky)
dns-server = system
fallback-dns-server = system
ipv6 = false
prefer-ipv6 = false
private-ip-answer = true
skip-proxy = `+skip+`
tun-excluded-routes = 10.0.0.0/8,100.64.0.0/10,127.0.0.0/8,169.254.0.0/16,172.16.0.0/12,192.0.0.0/24,192.168.0.0/16,224.0.0.0/4,255.255.255.255/32
icmp-auto-reply = false
udp-policy-not-supported-behaviour = REJECT
`) + "\n"
}
