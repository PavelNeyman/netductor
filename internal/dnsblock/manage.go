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
	// Do not reload here — operator presses 🔄 Reload after choosing lists.
	return nil
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
	nl := string([]byte{10})
	var b strings.Builder
	b.WriteString("🛡 <b>DNS block lists</b>" + nl)
	b.WriteString("<i>Списки <b>скачивает blocky</b> по URL при Reload (в netductor только ссылки в config.yml).</i>" + nl)
	b.WriteString("<table bordered striped>" + nl)
	b.WriteString("<tr><th>list</th><th>state</th><th>action</th></tr>" + nl)
	for _, e := range Catalog() {
		meta, ok := ListMeta[e.ID]
		title := e.ID
		if ok && meta.Title != "" {
			title = meta.Title
		}
		on := "OFF"
		style := ""
		act := "m:dns:on:" + e.ID
		label := "Enable"
		if e.Enabled {
			on = "ON"
			style = ` style="success"`
			act = "m:dns:off:" + e.ID
			label = "Disable"
		}
		desc := ""
		if ok {
			desc = meta.Desc
			if meta.Home != "" {
				desc += ` · <a href="` + meta.Home + `">source</a>`
			}
		}
		b.WriteString("<tr>")
		b.WriteString("<td><b>" + title + "</b>")
		if desc != "" {
			b.WriteString("<br/><i>" + desc + "</i>")
		}
		b.WriteString("</td>")
		b.WriteString("<td>" + on + "</td>")
		b.WriteString(`<td><tg-button type="callback_data"` + style + ` data="` + act + `">` + label + `</tg-button></td>`)
		b.WriteString("</tr>" + nl)
	}
	b.WriteString("</table>" + nl)
	b.WriteString(`<tg-button-row>`)
	b.WriteString(`<tg-button type="callback_data" style="primary" data="m:dns:reload">🔄 Reload lists</tg-button>`)
	b.WriteString(`</tg-button-row>`)
	return b.String()
}
