package install

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
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
	if err := verifyBlockyChecksum(tag, tgz, filepath.Base(url)); err != nil {
		return err
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
      - tcp-tls:dns.quad9.net:853
      - tcp-tls:cloudflare-dns.com:853
      - 9.9.9.9
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

func verifyBlockyChecksum(tag, archivePath, assetName string) error {
	sumURL := fmt.Sprintf("https://github.com/0xERR0R/blocky/releases/download/%s/blocky_checksums.txt", tag)
	tmp, err := os.CreateTemp("", "blocky-sum-")
	if err != nil {
		return err
	}
	tmp.Close()
	defer os.Remove(tmp.Name())
	if err := httpDownload(sumURL, tmp.Name()); err != nil {
		return fmt.Errorf("blocky checksums: %w", err)
	}
	b, err := os.ReadFile(tmp.Name())
	if err != nil {
		return err
	}
	want := ""
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		fields := strings.Fields(line)
		if len(fields) >= 2 && (fields[1] == assetName || strings.HasSuffix(fields[1], "/"+assetName)) {
			want = fields[0]
			break
		}
		if len(fields) >= 2 && strings.Contains(fields[len(fields)-1], assetName) {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("blocky: no checksum for %s", assetName)
	}
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("blocky checksum mismatch")
	}
	return nil
}


// EnsureBlockyDoT upgrades existing /etc/blocky/config.yml upstreams to prefer DoT (tcp-tls :853)
// while keeping a plaintext fallback. Idempotent if tcp-tls already present.
func EnsureBlockyDoT() error {
	cfg := "/etc/blocky/config.yml"
	b, err := os.ReadFile(cfg)
	if err != nil {
		return err
	}
	s := string(b)
	if strings.Contains(s, "tcp-tls:dns.quad9.net") || strings.Contains(s, "tcp-tls:cloudflare-dns.com") {
		return nil
	}
	// naive: replace simple upstream list under default: if present
	dotBlock := `    default:
      - tcp-tls:dns.quad9.net:853
      - tcp-tls:cloudflare-dns.com:853
      - 9.9.9.9
`
	// If we find "groups:" and "default:", try to rewrite first default list only via marker
	if !strings.Contains(s, "upstreams:") {
		// prepend upstreams section
		s = "upstreams:\n  groups:\n" + dotBlock + "  strategy: parallel_best\n" + s
	} else if strings.Contains(s, "default:") {
		// leave structure; append note file for operator — safer than brittle YAML rewrite
		_ = os.WriteFile("/etc/blocky/netductor-dot.snippet.yml", []byte(
			"# Suggested upstreams (merge into config.yml, then systemctl restart blocky):\n"+
				"upstreams:\n  groups:\n"+dotBlock+"  strategy: parallel_best\n"), 0o644)
		fmt.Fprintln(os.Stderr, "blocky: existing config kept; wrote /etc/blocky/netductor-dot.snippet.yml — merge DoT upstreams manually or set NETDUCTOR_BLOCKY_DOT=1 force")
		if os.Getenv("NETDUCTOR_BLOCKY_DOT") != "1" {
			return nil
		}
	}
	// Force path: backup and write minimal merge is risky — only if env set, replace upstreams section with sed-like
	if os.Getenv("NETDUCTOR_BLOCKY_DOT") == "1" {
		_ = os.WriteFile(cfg+".bak-dot", b, 0o644)
		// Very small configs only: if original default template shape
		if strings.Contains(s, "- 9.9.9.9") && strings.Contains(s, "- 1.1.1.1") {
			s = strings.Replace(s, "      - 9.9.9.9\n      - 1.1.1.1", "      - tcp-tls:dns.quad9.net:853\n      - tcp-tls:cloudflare-dns.com:853\n      - 9.9.9.9", 1)
			if err := os.WriteFile(cfg, []byte(s), 0o644); err != nil {
				return err
			}
			_ = run("systemctl", "restart", "blocky")
			fmt.Fprintln(os.Stderr, "blocky: DoT upstreams applied (NETDUCTOR_BLOCKY_DOT=1)")
		}
	}
	return nil
}
