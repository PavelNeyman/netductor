package deploy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EdgeOfflineManifest is written under ~/.cache/netductor/offline/edge/manifest.json
type EdgeOfflineManifest struct {
	At              string            `json:"at"`
	Version         string            `json:"version"`
	PrimaryHost     string            `json:"primary_host,omitempty"`
	BootstrapToken  string            `json:"-"` // not in JSON file; token is separate file
	TokenFile       string            `json:"token_file"`
	MTLSCAFile      string            `json:"mtls_ca_file"`
	MTLSCertFile    string            `json:"mtls_cert_file,omitempty"`
	MTLSKeyFile     string            `json:"mtls_key_file,omitempty"`
	DeviceID        string            `json:"device_id,omitempty"`
	AgentDir        string            `json:"agent_dir"`
	Agents          map[string]string `json:"agents"` // arch → path
	Note            string            `json:"note"`
}

// OfflineEdgeDir returns default offline pack directory.
func OfflineEdgeDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "netductor", "offline", "edge")
}

// PrepareEdgeOffline pulls token (+ optional mTLS for device_id) from primary and
// caches agent binaries for all supported arches. Requires network + SSH to primary.
func PrepareEdgeOffline(primaryHost, primaryUser, primaryKey, keyPass, deviceID, version string) (*EdgeOfflineManifest, error) {
	if primaryHost == "" || primaryKey == "" {
		return nil, fmt.Errorf("primary host and key required to prepare offline pack")
	}
	if strings.HasPrefix(primaryKey, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			primaryKey = filepath.Join(home, primaryKey[2:])
		}
	}
	if primaryUser == "" {
		primaryUser = "root"
	}
	if version == "" {
		version = Release
	}
	outDir := OfflineEdgeDir()
	_ = os.MkdirAll(outDir, 0o700)
	home, _ := os.UserHomeDir()
	agentDir := filepath.Join(home, ".cache", "netductor", "agents")
	_ = os.MkdirAll(agentDir, 0o755)

	fmt.Fprintln(os.Stderr, "==> offline pack: need internet + SSH to primary :52222")

	out, err := runSSHOnPort(day2SSHPort(), "", primaryKey, primaryUser, primaryHost,
		"cat /etc/netductor/secrets/edge_bootstrap_token 2>/dev/null", keyPass)
	if err != nil {
		return nil, fmt.Errorf("bootstrap token: %w\n%s", err, out)
	}
	token := strings.TrimSpace(out)
	if token == "" {
		return nil, fmt.Errorf("empty edge_bootstrap_token on primary")
	}
	tokenFile := filepath.Join(outDir, "edge_bootstrap_token")
	if err := os.WriteFile(tokenFile, []byte(token+"\n"), 0o600); err != nil {
		return nil, err
	}
	fmt.Fprintln(os.Stderr, "==> wrote", tokenFile)

	// Always fetch CA; optional per-device client cert.
	caFile := filepath.Join(outDir, "mtls-ca.crt")
	certFile, keyFile := "", ""
	remote := `set -e
CA=/etc/netductor/secrets/mtls/ca.crt
b64() { base64 -w0 "$1" 2>/dev/null || base64 "$1" | tr -d '\n'; }
echo CA:$(b64 "$CA")
`
	if strings.TrimSpace(deviceID) != "" {
		remote = fmt.Sprintf(`set -e
netductor mtls ensure >/dev/null 2>&1 || true
netductor mtls issue-client %s >/dev/null 2>&1
CA=/etc/netductor/secrets/mtls/ca.crt
DIR=/etc/netductor/secrets/mtls/clients/%s
if [ ! -f "$DIR/client.crt" ]; then
  DIR=$(find /etc/netductor/secrets/mtls/clients -maxdepth 1 -type d 2>/dev/null | while read d; do
    [ -f "$d/client.crt" ] || continue
    echo "$d"
  done | head -1)
fi
b64() { base64 -w0 "$1" 2>/dev/null || base64 "$1" | tr -d '\n'; }
echo CA:$(b64 "$CA")
echo CERT:$(b64 "$DIR/client.crt")
echo KEY:$(b64 "$DIR/client.key")
`, shellQuote(deviceID), shellQuote(deviceID))
	}
	mout, err := runSSHOnPort(day2SSHPort(), "", primaryKey, primaryUser, primaryHost, remote, keyPass)
	if err != nil {
		fmt.Fprintln(os.Stderr, "warn: mTLS from primary:", err)
	} else {
		var ca, cert, key []byte
		for _, line := range strings.Split(mout, "\n") {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "CA:"):
				ca, _ = decodeB64(strings.TrimPrefix(line, "CA:"))
			case strings.HasPrefix(line, "CERT:"):
				cert, _ = decodeB64(strings.TrimPrefix(line, "CERT:"))
			case strings.HasPrefix(line, "KEY:"):
				key, _ = decodeB64(strings.TrimPrefix(line, "KEY:"))
			}
		}
		if len(ca) > 0 {
			_ = os.WriteFile(caFile, ca, 0o600)
			fmt.Fprintln(os.Stderr, "==> wrote", caFile)
		}
		if len(cert) > 0 && len(key) > 0 {
			certFile = filepath.Join(outDir, "mtls-client.crt")
			keyFile = filepath.Join(outDir, "mtls-client.key")
			_ = os.WriteFile(certFile, cert, 0o600)
			_ = os.WriteFile(keyFile, key, 0o600)
			fmt.Fprintln(os.Stderr, "==> wrote client cert for device", deviceID)
		}
	}

	agents := map[string]string{}
	for arch := range agentArches {
		// Offline-prep always refreshes agents for the target release version.
		_ = os.Remove(filepath.Join(agentDir, "netductor-agent-linux-"+arch))
		path, err := EnsureAgentBinary(version, arch, agentDir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn: agent", arch, err)
			continue
		}
		agents[arch] = path
	}

	m := &EdgeOfflineManifest{
		At:             time.Now().UTC().Format(time.RFC3339),
		Version:        version,
		PrimaryHost:    primaryHost,
		TokenFile:      tokenFile,
		MTLSCAFile:     caFile,
		MTLSCertFile:   certFile,
		MTLSKeyFile:    keyFile,
		DeviceID:       deviceID,
		AgentDir:       agentDir,
		Agents:         agents,
		Note:           "Refresh when upgrading netductor (agent binaries). Re-pull token if rotated on primary. Client cert is per device_id — re-prep or issue for each new edge id.",
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	manPath := filepath.Join(outDir, "manifest.json")
	if err := os.WriteFile(manPath, b, 0o600); err != nil {
		return nil, err
	}
	fmt.Fprintln(os.Stderr, "==> offline pack ready:", manPath)
	return m, nil
}

// LoadEdgeOfflineManifest reads manifest + token file for offline DeployEdge.
func LoadEdgeOfflineManifest() (*EdgeOfflineManifest, string, error) {
	manPath := filepath.Join(OfflineEdgeDir(), "manifest.json")
	b, err := os.ReadFile(manPath)
	if err != nil {
		return nil, "", fmt.Errorf("no offline pack (%s): run offline-prep first", manPath)
	}
	var m EdgeOfflineManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, "", err
	}
	tok := ""
	if m.TokenFile != "" {
		if tb, err := os.ReadFile(m.TokenFile); err == nil {
			tok = strings.TrimSpace(string(tb))
		}
	}
	if tok == "" {
		if tb, err := os.ReadFile(filepath.Join(OfflineEdgeDir(), "edge_bootstrap_token")); err == nil {
			tok = strings.TrimSpace(string(tb))
		}
	}
	return &m, tok, nil
}

func decodeB64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}
