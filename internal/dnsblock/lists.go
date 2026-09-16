package dnsblock

// AdGuard Hostlists Registry + common lists for blocky denylists.
var AdGuardStyleLists = map[string]string{
	"adguard-dns-filter": "https://adguardteam.github.io/HostlistsRegistry/assets/filter_1.txt",
	"adguard-mobile-ads": "https://adguardteam.github.io/HostlistsRegistry/assets/filter_11.txt",
	"adguard-tracking":   "https://adguardteam.github.io/HostlistsRegistry/assets/filter_3.txt",
	"hagezi-multi":       "https://cdn.jsdelivr.net/gh/hagezi/dns-blocklists@latest/wildcard/multi.txt",
	"stevenblack-hosts":  "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
}

func DefaultBlockyDenyURLs() []string {
	return []string{
		AdGuardStyleLists["adguard-dns-filter"],
		AdGuardStyleLists["hagezi-multi"],
	}
}
