package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func InstallBlocky() error {
	_ = aptInstall("curl", "tar", "dnsutils")
	tag, err := latestTag("0xERR0R/blocky")
	if err != nil {
		return err
	}
	if stateVersion("blocky") == tag {
		if _, err := os.Stat("/usr/local/bin/blocky"); err == nil {
			fmt.Fprintf(os.Stderr, "blocky %s already installed — skip\n", tag)
			return enableStart("blocky")
		}
	}
	a := arch()
	assetArch := a
	if a == "amd64" {
		assetArch = "x86_64"
	}
	url := fmt.Sprintf("https://github.com/0xERR0R/blocky/releases/download/%s/blocky_%s_Linux_%s.tar.gz", tag, tag, assetArch)
	tmp, err := os.MkdirTemp("", "blocky-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	tgz := filepath.Join(tmp, "b.tgz")
	if err := httpDownload(url, tgz); err != nil {
		url = fmt.Sprintf("https://github.com/0xERR0R/blocky/releases/download/%s/blocky_%s_Linux_%s.tar.gz", tag, tag, a)
		if err2 := httpDownload(url, tgz); err2 != nil {
			return err
		}
	}
	if err := run("tar", "-xzf", tgz, "-C", tmp); err != nil {
		return err
	}
	// find binary
	var bin string
	_ = filepath.Walk(tmp, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() && info.Name() == "blocky" {
			bin = path
		}
		return nil
	})
	if bin == "" {
		return fmt.Errorf("blocky binary not in archive")
	}
	if err := run("install", "-m", "755", bin, "/usr/local/bin/blocky"); err != nil {
		return err
	}
	writeStateVersion("blocky", tag)
	_ = os.MkdirAll("/etc/blocky", 0o755)
	cfg := `/etc/blocky/config.yml`
	if _, err := os.Stat(cfg); err != nil {
		content := `upstreams:
  groups:
    default:
      - 9.9.9.9
      - 1.1.1.1
  strategy: parallel_best
blocking:
  denylists:
    ads:
      - https://cdn.jsdelivr.net/gh/hagezi/dns-blocklists@latest/wildcard/multi.txt
  clientGroupsBlock:
    default:
      - ads
  blockType: nxDomain
ports:
  dns: 127.0.0.1:53
  http: 127.0.0.1:4000
log:
  level: info
`
		_ = os.WriteFile(cfg, []byte(content), 0o644)
	}
	unit := `[Unit]
Description=Blocky DNS (Netductor)
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/blocky --config /etc/blocky/config.yml
Restart=on-failure
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
`
	if err := writeUnit("blocky.service", unit); err != nil {
		return err
	}
	// optional resolv.conf - only if not exists conflict
	_ = strings.TrimSpace
	return enableStart("blocky")
}
