package install

import (
	"github.com/PavelNeyman/netductor/internal/ndconfig"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/vpn"
)

func InstallSingBox() error {
	_ = aptInstall("curl", "tar", "openssl", "ca-certificates")
	binDir := "/usr/local/bin"
	confDir := "/usr/local/etc/sing-box"
	_ = os.MkdirAll(confDir, 0o755)
	_ = os.MkdirAll("/etc/sing-box/certs", 0o755)

	tag, err := latestTag("SagerNet/sing-box")
	if err != nil {
		return err
	}
	if stateVersion("singbox") == tag {
		if _, err := os.Stat("/usr/local/bin/sing-box"); err == nil {
			fmt.Fprintf(os.Stderr, "sing-box %s already installed — skip download\n", tag)
			goto afterBin
		}
	}
	if err := downloadSingBoxVersion(binDir, tag); err != nil {
		return err
	}
	writeStateVersion("singbox", tag)
afterBin:
	shortID := readSecret("singbox_short_id")
	if shortID == "" {
		shortID = randomHex(8)
		_ = writeSecret("singbox_short_id", shortID)
	}
	priv := readSecret("singbox_reality_private")
	pub := readSecret("singbox_reality_public")
	if priv == "" {
		out, err := exec.Command(filepath.Join(binDir, "sing-box"), "generate", "reality-keypair").Output()
		if err != nil {
			return err
		}
		for _, line := range strings.Split(string(out), "\n") {
			f := strings.Fields(line)
			if len(f) >= 2 && strings.Contains(f[0], "Private") {
				priv = f[len(f)-1]
			}
			if len(f) >= 2 && strings.Contains(f[0], "Public") {
				pub = f[len(f)-1]
			}
		}
		_ = writeSecret("singbox_reality_private", priv)
		_ = writeSecret("singbox_reality_public", pub)
	}
	sni := env("SINGBOX_REALITY_SNI", "")
	if sni == "" {
		sni = readSecret("singbox_reality_sni")
	}
	if sni == "" {
		sni = ndconfig.DefaultSNI()
	}
	_ = writeSecret("singbox_reality_sni", sni)
	if ndconfig.HY2Enabled() {
	if _, err := os.Stat("/etc/sing-box/certs/hy2.crt"); err != nil {
		_ = exec.Command("openssl", "req", "-x509", "-nodes", "-newkey", "ec",
			"-pkeyopt", "ec_paramgen_curve:prime256v1",
			"-keyout", "/etc/sing-box/certs/hy2.key",
			"-out", "/etc/sing-box/certs/hy2.crt",
			"-days", "3650", "-subj", "/CN="+sni).Run()
		_ = os.Chmod("/etc/sing-box/certs/hy2.key", 0o600)
	}
	}
	// minimal empty config; users via vpn.ApplyConfig
	if err := vpn.EnsureDirs(); err != nil {
		return err
	}
	if err := vpn.ApplyConfig(); err != nil {
		// first install may fail without users — write empty template
		_ = writeEmptySingBox(confDir, priv, shortID, sni)
	}
	unit := `[Unit]
Description=sing-box (Netductor)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/sing-box run -c /usr/local/etc/sing-box/config.json
Restart=on-failure
RestartSec=3
LimitNOFILE=1048576
AmbientCapabilities=CAP_NET_BIND_SERVICE
CapabilityBoundingSet=CAP_NET_BIND_SERVICE

[Install]
WantedBy=multi-user.target
`
	if err := writeUnit("sing-box.service", unit); err != nil {
		return err
	}
	return enableStart("sing-box")
}

func downloadSingBox(dest string) error {
	tag, err := latestTag("SagerNet/sing-box")
	if err != nil {
		return err
	}
	return downloadSingBoxVersion(dest, tag)
}

func downloadSingBoxVersion(dest, tag string) error {
	a := arch()
	url := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/%s/sing-box-%s-linux-%s.tar.gz", tag, strings.TrimPrefix(tag, "v"), a)
	tmp, err := os.MkdirTemp("", "sb-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	tgz := filepath.Join(tmp, "sb.tgz")
	if err := httpDownload(url, tgz); err != nil {
		return err
	}
	if err := run("tar", "-xzf", tgz, "-C", tmp); err != nil {
		return err
	}
	src := filepath.Join(tmp, fmt.Sprintf("sing-box-%s-linux-%s", strings.TrimPrefix(tag, "v"), a), "sing-box")
	return run("install", "-m", "755", src, filepath.Join(dest, "sing-box"))
}

func writeEmptySingBox(confDir, priv, sid, sni string) error {
	cfg := map[string]any{
		"log": map[string]any{"level": "info", "timestamp": true},
		"inbounds": []any{
			map[string]any{
				"type": "vless", "tag": "vless-reality", "listen": "::", "listen_port": 443,
				"users": []any{},
				"tls": map[string]any{
					"enabled": true, "server_name": sni,
					"reality": map[string]any{
						"enabled": true,
						"handshake": map[string]any{"server": sni, "server_port": 443},
						"private_key": priv, "short_id": []string{sid},
					},
				},
			},
		},
		"outbounds": []any{map[string]any{"type": "direct", "tag": "direct"}},
	}
	b, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(filepath.Join(confDir, "config.json"), append(b, '\n'), 0o600)
}

func latestTag(repo string) (string, error) {
	resp, err := http.Get("https://api.github.com/repos/" + repo + "/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var m struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return "", err
	}
	if m.Tag == "" {
		return "", fmt.Errorf("no tag for %s", repo)
	}
	return m.Tag, nil
}

func httpDownload(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d %s", resp.StatusCode, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func randomHex(n int) string {
	if n <= 0 {
		n = 8
	}
	b := make([]byte, n)
	f, err := os.Open("/dev/urandom")
	if err != nil {
		return "abcd1234abcd1234"[:n]
	}
	defer f.Close()
	if _, err := io.ReadFull(f, b); err != nil {
		return "abcd1234abcd1234"[:n]
	}
	const hexdigits = "0123456789abcdef"
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = hexdigits[int(b[i])%16]
	}
	return string(out)
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
