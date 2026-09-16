package dnsblock

// AdGuard Hostlists Registry + common lists for blocky denylists.
// Lists are **downloaded by blocky** from these URLs on refresh (not bundled in netductor).
var AdGuardStyleLists = map[string]string{
	"adguard-dns-filter": "https://adguardteam.github.io/HostlistsRegistry/assets/filter_1.txt",
	"adguard-mobile-ads": "https://adguardteam.github.io/HostlistsRegistry/assets/filter_11.txt",
	"adguard-tracking":   "https://adguardteam.github.io/HostlistsRegistry/assets/filter_3.txt",
	"hagezi-multi":       "https://cdn.jsdelivr.net/gh/hagezi/dns-blocklists@latest/wildcard/multi.txt",
	"stevenblack-hosts":  "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
}

// ListMeta human description + homepage for UI.
var ListMeta = map[string]struct {
	Title string
	Desc  string
	Home  string
}{
	"adguard-dns-filter": {
		Title: "AdGuard DNS filter",
		Desc:  "Base ads/trackers (AdGuard Hostlists #1)",
		Home:  "https://adguardteam.github.io/HostlistsRegistry/",
	},
	"adguard-mobile-ads": {
		Title: "AdGuard Mobile ads",
		Desc:  "Mobile ad networks (Hostlists #11)",
		Home:  "https://adguardteam.github.io/HostlistsRegistry/",
	},
	"adguard-tracking": {
		Title: "AdGuard Tracking",
		Desc:  "Tracking protection (Hostlists #3)",
		Home:  "https://adguardteam.github.io/HostlistsRegistry/",
	},
	"hagezi-multi": {
		Title: "HaGeZi multi",
		Desc:  "Aggressive multi-purpose blocklist",
		Home:  "https://github.com/hagezi/dns-blocklists",
	},
	"stevenblack-hosts": {
		Title: "StevenBlack hosts",
		Desc:  "Classic hosts-format adware/malware list",
		Home:  "https://github.com/StevenBlack/hosts",
	},
}

func DefaultBlockyDenyURLs() []string {
	return []string{
		AdGuardStyleLists["adguard-dns-filter"],
		AdGuardStyleLists["hagezi-multi"],
	}
}
