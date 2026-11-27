package vpn

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Karing profile IDs mirror SR profiles (same policy intent).
func NormalizeKaringProfile(p string) string {
	return NormalizeSRProfile(p)
}

// KaringProfiles returns UI metadata (same labels as SR).
func KaringProfiles() []SRProfileMeta {
	return SRProfiles()
}

// BuildClashMetaYAML builds a full Clash Meta config for Karing one-tap import.
// Mirrors Shadowrocket SR Config policy for the given profile id.
// Serve at GET /sub/{token}/karing/{profile}.yaml
func BuildClashMetaYAML(userName, profile string) (string, error) {
	profile = NormalizeKaringProfile(profile)
	body, err := SubscriptionBody(userName)
	if err != nil {
		return "", err
	}
	proxies, names, err := vlessLinesToClashProxies(body)
	if err != nil {
		return "", err
	}
	if len(proxies) == 0 {
		return "", fmt.Errorf("no VLESS proxies in subscription for %s", userName)
	}

	var b strings.Builder
	b.WriteString("# netductor Karing/Clash Meta profile=" + profile + " user=" + userName + "\n")
	b.WriteString("# Import: karing://install-config?url=<this>&name=nd-" + profile + "\n")
	b.WriteString("# Policy mirrors Shadowrocket SR Config of the same profile id.\n")
	b.WriteString("mixed-port: 7890\n")
	b.WriteString("allow-lan: false\n")
	b.WriteString("mode: rule\n")
	b.WriteString("log-level: info\n")
	b.WriteString("ipv6: false\n")
	b.WriteString("unified-delay: true\n")
	b.WriteString("tcp-concurrent: true\n")
	b.WriteString("\nproxies:\n")
	for _, p := range proxies {
		b.WriteString(p)
		b.WriteByte('\n')
	}

	b.WriteString("\nproxy-groups:\n")
	b.WriteString("  - name: PROXY\n")
	b.WriteString("    type: select\n")
	b.WriteString("    proxies:\n")
	for _, n := range names {
		b.WriteString("      - " + yamlQuote(n) + "\n")
	}
	b.WriteString("      - DIRECT\n")
	b.WriteString("  - name: AUTO\n")
	b.WriteString("    type: url-test\n")
	b.WriteString("    url: https://www.gstatic.com/generate_204\n")
	b.WriteString("    interval: 300\n")
	b.WriteString("    proxies:\n")
	for _, n := range names {
		b.WriteString("      - " + yamlQuote(n) + "\n")
	}

	adsURLs := karingAdsRuleProviderURLs()
	if len(adsURLs) > 0 {
		b.WriteString("\nrule-providers:\n")
		for i, u := range adsURLs {
			id := fmt.Sprintf("nd-ads-%d", i)
			b.WriteString("  " + id + ":\n")
			b.WriteString("    type: http\n")
			b.WriteString("    behavior: classical\n")
			b.WriteString("    format: text\n")
			b.WriteString("    url: " + yamlQuote(u) + "\n")
			b.WriteString("    interval: 86400\n")
			b.WriteString("    path: ./ruleset/" + id + ".txt\n")
		}
	}

	b.WriteString("\nrules:\n")
	b.WriteString("  - IP-CIDR,127.0.0.0/8,DIRECT,no-resolve\n")
	b.WriteString("  - IP-CIDR,169.254.0.0/16,DIRECT,no-resolve\n")
	b.WriteString("  - IP-CIDR,224.0.0.0/4,DIRECT,no-resolve\n")

	for _, c := range AdminNodeDirectCIDRs() {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		if !strings.Contains(c, "/") {
			c += "/32"
		}
		b.WriteString("  - IP-CIDR," + c + ",DIRECT,no-resolve\n")
	}

	writeClashCIDRs := func(cidrs []string, policy string) {
		for _, c := range cidrs {
			c = strings.TrimSpace(c)
			if c == "" {
				continue
			}
			b.WriteString("  - IP-CIDR," + c + "," + policy + ",no-resolve\n")
		}
	}
	writeClashDomains := func(domains []string, policy string) {
		for _, d := range domains {
			d = strings.TrimSpace(d)
			d = strings.TrimPrefix(d, "*.")
			if d == "" {
				continue
			}
			b.WriteString("  - DOMAIN-SUFFIX," + d + "," + policy + "\n")
		}
	}
	writeRuDirectFromSR := func() {
		for _, line := range ShadowrocketRuDirectRules() {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Split(line, ",")
			if len(parts) < 3 {
				continue
			}
			switch parts[0] {
			case "DOMAIN-SUFFIX":
				b.WriteString("  - DOMAIN-SUFFIX," + parts[1] + ",DIRECT\n")
			case "IP-CIDR", "IP-CIDR6":
				b.WriteString("  - " + parts[0] + "," + parts[1] + ",DIRECT,no-resolve\n")
			case "GEOIP":
				b.WriteString("  - GEOIP," + parts[1] + ",DIRECT\n")
			}
		}
	}

	switch profile {
	case SRProfileOperatorMobile:
		writeClashCIDRs(homeLANCIDRs(), "DIRECT")
		writeClashCIDRs(workDirectCIDRs(), "DIRECT")
		writeClashDomains(workDirectDomains(), "DIRECT")
		writeClashDomains(mailDirectDomains(), "DIRECT")
		for i := range adsURLs {
			b.WriteString(fmt.Sprintf("  - RULE-SET,nd-ads-%d,REJECT\n", i))
		}
		writeRuDirectFromSR()
		writeClashCIDRs(serviceNetCIDRs(), "PROXY")
		writeClashCIDRs(shopCIDRs(), "PROXY")
	case SRProfileOperatorFullProxy:
		writeClashCIDRs(homeLANCIDRs(), "DIRECT")
		writeClashCIDRs(workDirectCIDRs(), "DIRECT")
		writeClashDomains(workDirectDomains(), "DIRECT")
		writeClashDomains(mailDirectDomains(), "DIRECT")
		for i := range adsURLs {
			b.WriteString(fmt.Sprintf("  - RULE-SET,nd-ads-%d,REJECT\n", i))
		}
		writeClashCIDRs(serviceNetCIDRs(), "PROXY")
		writeClashCIDRs(shopCIDRs(), "PROXY")
	case SRProfileFamily:
		for i := range adsURLs {
			b.WriteString(fmt.Sprintf("  - RULE-SET,nd-ads-%d,REJECT\n", i))
		}
		writeRuDirectFromSR()
	}

	b.WriteString("  - MATCH,PROXY\n")
	return b.String(), nil
}

func karingAdsRuleProviderURLs() []string {
	if mode := strings.TrimSpace(os.Getenv("NETDUCTOR_SR_ADS_REJECT")); mode == "0" || mode == "off" || mode == "false" {
		return nil
	}
	return []string{
		"https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/Advertising/Advertising.list",
		"https://cdn.jsdelivr.net/gh/blackmatrix7/ios_rule_script@master/rule/Clash/AdvertisingLite/AdvertisingLite.list",
	}
}

func yamlQuote(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, ":#{}[],&*?|>!%@`'\" \t") {
		return strconv.Quote(s)
	}
	return s
}

// vlessLinesToClashProxies converts subscription text (vless:// lines or base64 body already decoded) to YAML proxy blocks.
func vlessLinesToClashProxies(body string) (yamlBlocks []string, names []string, err error) {
	seen := map[string]int{}
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(line), "vless://") {
			continue
		}
		block, name, perr := vlessURIToClashYAML(line)
		if perr != nil {
			continue
		}
		base := name
		if n := seen[base]; n > 0 {
			name = fmt.Sprintf("%s-%d", base, n+1)
		}
		seen[base]++
		// fix name inside block if renamed
		if name != base {
			block = strings.Replace(block, "name: "+yamlQuote(base), "name: "+yamlQuote(name), 1)
			block = strings.Replace(block, "name: "+base, "name: "+yamlQuote(name), 1)
		}
		yamlBlocks = append(yamlBlocks, block)
		names = append(names, name)
	}
	if len(yamlBlocks) == 0 {
		return nil, nil, fmt.Errorf("no parseable vless:// lines")
	}
	return yamlBlocks, names, nil
}

func vlessURIToClashYAML(uri string) (block string, name string, err error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", "", err
	}
	if !strings.EqualFold(u.Scheme, "vless") {
		return "", "", fmt.Errorf("not vless")
	}
	uuid := u.User.Username()
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "443"
	}
	q := u.Query()
	name = strings.TrimSpace(u.Fragment)
	if name == "" {
		name = host
	}
	// sanitize name for Clash
	name = strings.ReplaceAll(name, "\n", "")
	sni := firstNonEmptyStr(q.Get("sni"), q.Get("servername"), host)
	pbk := q.Get("pbk")
	sid := q.Get("sid")
	fp := firstNonEmptyStr(q.Get("fp"), "chrome")
	flow := q.Get("flow")
	network := firstNonEmptyStr(q.Get("type"), "tcp")
	security := strings.ToLower(q.Get("security"))

	var b strings.Builder
	b.WriteString("  - name: " + yamlQuote(name) + "\n")
	b.WriteString("    type: vless\n")
	b.WriteString("    server: " + host + "\n")
	b.WriteString("    port: " + port + "\n")
	b.WriteString("    uuid: " + uuid + "\n")
	b.WriteString("    network: " + network + "\n")
	b.WriteString("    udp: true\n")
	if flow != "" {
		b.WriteString("    flow: " + flow + "\n")
	}
	if security == "reality" || pbk != "" {
		b.WriteString("    tls: true\n")
		b.WriteString("    servername: " + yamlQuote(sni) + "\n")
		b.WriteString("    client-fingerprint: " + yamlQuote(fp) + "\n")
		b.WriteString("    reality-opts:\n")
		b.WriteString("      public-key: " + yamlQuote(pbk) + "\n")
		if sid != "" {
			b.WriteString("      short-id: " + yamlQuote(sid) + "\n")
		}
	} else if security == "tls" {
		b.WriteString("    tls: true\n")
		b.WriteString("    servername: " + yamlQuote(sni) + "\n")
		b.WriteString("    client-fingerprint: " + yamlQuote(fp) + "\n")
	}
	return b.String(), name, nil
}

func firstNonEmptyStr(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
