package dnsblock

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const blockyConfig = "/etc/blocky/config.yml"

type ListEntry struct {
	ID      string `json:"id"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

func currentDenyURLs() []string {
	b, err := os.ReadFile(blockyConfig)
	if err != nil {
		return nil
	}
	var urls []string
	re := regexp.MustCompile(`(?m)^\s*-\s+(https?://\S+)`)
	inAds := false
	for _, line := range strings.Split(string(b), "\n") {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "ads:") {
			inAds = true
			continue
		}
		if inAds {
			if strings.HasPrefix(trim, "-") {
				m := re.FindStringSubmatch(line)
				if len(m) > 1 {
					urls = append(urls, m[1])
				} else {
					u := strings.TrimSpace(strings.TrimPrefix(trim, "-"))
					if strings.HasPrefix(u, "http") {
						urls = append(urls, u)
					}
				}
				continue
			}
			if trim != "" && !strings.HasPrefix(trim, "#") {
				inAds = false
			}
		}
	}
	if len(urls) == 0 {
		for _, m := range re.FindAllStringSubmatch(string(b), -1) {
			urls = append(urls, m[1])
		}
	}
	return urls
}

func Catalog() []ListEntry {
	enabled := map[string]bool{}
	for _, u := range currentDenyURLs() {
		enabled[u] = true
	}
	// stable order from map keys of AdGuardStyleLists
	ids := []string{"adguard-dns-filter", "adguard-tracking", "adguard-mobile-ads", "hagezi-multi", "stevenblack-hosts"}
	var out []ListEntry
	seen := map[string]bool{}
	for _, id := range ids {
		url, ok := AdGuardStyleLists[id]
		if !ok {
			continue
		}
		out = append(out, ListEntry{ID: id, URL: url, Enabled: enabled[url]})
		seen[url] = true
	}
	for u := range enabled {
		if seen[u] {
			continue
		}
		out = append(out, ListEntry{ID: "custom", URL: u, Enabled: true})
	}
	return out
}

func SetEnabled(idOrURL string, on bool) error {
	url := idOrURL
	if u, ok := AdGuardStyleLists[idOrURL]; ok {
		url = u
	}
	b, err := os.ReadFile(blockyConfig)
	if err != nil {
		return err
	}
	text := string(b)
	cur := currentDenyURLs()
	have := false
	var next []string
	for _, u := range cur {
		if u == url {
			have = true
			if on {
				next = append(next, u)
			}
			continue
		}
		next = append(next, u)
	}
	if on && !have {
		next = append(next, url)
	}
	var adsBlock strings.Builder
	adsBlock.WriteString("    ads:\n")
	for _, u := range next {
		adsBlock.WriteString("      - " + u + "\n")
	}
	reAds := regexp.MustCompile(`(?ms)(denylists:\s*\n)(\s+ads:\n(?:\s+-\s+\S+\n)*)`)
	if reAds.MatchString(text) {
		text = reAds.ReplaceAllString(text, "${1}"+adsBlock.String())
	} else {
		text += "\nblocking:\n  denylists:\n" + adsBlock.String()
		text += "  clientGroupsBlock:\n    default:\n      - ads\n"
	}
	if err := os.WriteFile(blockyConfig, []byte(text), 0o644); err != nil {
		return err
	}
	return ReloadBlocky()
}

func AddCustomURL(url string) error {
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("url must be http(s)")
	}
	return SetEnabled(url, true)
}

// ReloadBlocky refreshes lists via HTTP API (no restart → no DNS flap / probe noise).
func ReloadBlocky() error {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Post("http://127.0.0.1:4000/api/lists/refresh", "application/json", nil)
	if err != nil {
		return exec.Command("systemctl", "restart", "blocky").Run()
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return exec.Command("systemctl", "restart", "blocky").Run()
	}
	return nil
}

func FormatCatalogHTML() string {
	nl := "\n"
	var b strings.Builder
	b.WriteString("🛡 <b>DNS block lists</b>" + nl)
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>#</th><th>list</th><th>on</th></tr>" + nl)
	for i, e := range Catalog() {
		on := "☐"
		if e.Enabled {
			on = "☑"
		}
		b.WriteString(fmt.Sprintf("<tr><td>%d</td><td><code>%s</code></td><td>%s</td></tr>%s", i+1, e.ID, on, nl))
	}
	b.WriteString("</table>" + nl)
	b.WriteString("<i>Tap a list to toggle. Applied via blocky API (no restart).</i>")
	return b.String()
}
