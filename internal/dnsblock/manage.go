package dnsblock

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
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
			if strings.HasPrefix(trim, "http") || strings.HasPrefix(trim, "- http") {
				m := re.FindStringSubmatch(line)
				if len(m) > 1 {
					urls = append(urls, m[1])
				} else if strings.HasPrefix(trim, "- ") {
					urls = append(urls, strings.TrimPrefix(trim, "- "))
				}
				continue
			}
			if trim != "" && !strings.HasPrefix(trim, "-") {
				inAds = false
			}
		}
	}
	// fallback: any list URL under denylists
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
	var out []ListEntry
	for id, url := range AdGuardStyleLists {
		out = append(out, ListEntry{ID: id, URL: url, Enabled: enabled[url]})
	}
	for u := range enabled {
		found := false
		for _, e := range out {
			if e.URL == u {
				found = true
				break
			}
		}
		if !found {
			out = append(out, ListEntry{ID: "custom", URL: u, Enabled: true})
		}
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
	// rebuild ads: block under denylists
	var adsBlock strings.Builder
	adsBlock.WriteString("    ads:\n")
	for _, u := range next {
		adsBlock.WriteString("      - " + u + "\n")
	}
	reAds := regexp.MustCompile(`(?ms)(denylists:\s*\n)(\s+ads:\n(?:\s+-\s+\S+\n)*)`)
	if reAds.MatchString(text) {
		text = reAds.ReplaceAllString(text, "${1}"+adsBlock.String())
	} else {
		// append minimal blocking section
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

func ReloadBlocky() error {
	if err := exec.Command("systemctl", "reload", "blocky").Run(); err != nil {
		return exec.Command("systemctl", "restart", "blocky").Run()
	}
	return nil
}

func FormatCatalogText() string {
	var b strings.Builder
	b.WriteString("DNS block lists\n")
	for _, e := range Catalog() {
		mark := "☐"
		if e.Enabled {
			mark = "☑"
		}
		b.WriteString(fmt.Sprintf("%s %s\n  %s\n", mark, e.ID, e.URL))
	}
	return b.String()
}
