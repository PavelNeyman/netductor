package deploy

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/mtls"
)

type EdgeOpts struct {
	PrimaryHost    string
	PrimaryUser    string
	PrimaryKey     string
	RouterHost     string
	RouterUser     string
	RouterPass     string
	DeviceID       string
	ServerURL      string
	AgentArch      string
	Version        string
	AgentDir       string
	OperatorPubKey string
}

func DeployEdge(o EdgeOpts) error {
	if o.RouterHost == "" || o.DeviceID == "" {
		return fmt.Errorf("router host and device id required")
	}
	if o.RouterUser == "" {
		o.RouterUser = "root"
	}
	if o.PrimaryUser == "" {
		o.PrimaryUser = "root"
	}
	if o.Version == "" {
		o.Version = "0.8.5"
	}
	if o.AgentArch == "" {
		o.AgentArch = "arm64"
	}
	if o.AgentDir == "" {
		home, _ := os.UserHomeDir()
		o.AgentDir = filepath.Join(home, ".cache", "netductor", "agents")
	}
	if o.ServerURL == "" && o.PrimaryHost != "" {
		o.ServerURL = "https://" + o.PrimaryHost + ":" + mtls.AgentTLSPort
	}
	if o.ServerURL == "" {
		return fmt.Errorf("server URL or primary host required")
	}
	// Never use plain public admin :8787 for edge control.
	if strings.HasPrefix(o.ServerURL, "http://") {
		u := strings.TrimPrefix(o.ServerURL, "http://")
		u = strings.Replace(u, ":8787", ":"+mtls.AgentTLSPort, 1)
		if !strings.Contains(u, ":") {
			u = u + ":" + mtls.AgentTLSPort
		}
		o.ServerURL = "https://" + u
	}

	token := ""
	var mtlsCA, mtlsCert, mtlsKey []byte
	if o.PrimaryHost != "" && o.PrimaryKey != "" {
		out, err := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
			"cat /etc/netductor/secrets/edge_bootstrap_token 2>/dev/null")
		if err != nil {
			return fmt.Errorf("read bootstrap token from primary: %w\n%s", err, out)
		}
		token = strings.TrimSpace(out)

		// Issue client cert on primary; print base64 lines (OpenWrt-safe path matching ClientDir).
		remote := fmt.Sprintf(`set -e
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
`, shellQuote(o.DeviceID), shellQuote(o.DeviceID))
		mout, err := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, remote)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn: mtls material from primary:", err)
			fmt.Fprintln(os.Stderr, mout)
		} else {
			for _, line := range strings.Split(mout, "\n") {
				line = strings.TrimSpace(line)
				switch {
				case strings.HasPrefix(line, "CA:"):
					mtlsCA, _ = base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "CA:"))
				case strings.HasPrefix(line, "CERT:"):
					mtlsCert, _ = base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "CERT:"))
				case strings.HasPrefix(line, "KEY:"):
					mtlsKey, _ = base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "KEY:"))
				}
			}
		}
	}

	agent, err := EnsureAgentBinary(o.Version, o.AgentArch, o.AgentDir)
	if err != nil {
		return err
	}
	pub := strings.TrimSpace(o.OperatorPubKey)
	if pub == "" && o.PrimaryKey != "" {
		if b, err := os.ReadFile(o.PrimaryKey + ".pub"); err == nil {
			pub = strings.TrimSpace(string(b))
		}
	}
	target := o.RouterUser + "@" + o.RouterHost
	fmt.Fprintln(os.Stderr, "==> edge provision", target, "→", o.ServerURL)
	if len(mtlsCert) > 0 {
		fmt.Fprintln(os.Stderr, "==> mTLS client material will be installed on router")
	} else {
		fmt.Fprintln(os.Stderr, "warn: no mTLS material — agent may fail TLS handshake until certs are present")
	}
	return edge.Provision(edge.ProvisionOpts{
		SSHTarget:      target,
		DeviceID:       o.DeviceID,
		ServerURL:      o.ServerURL,
		AgentBin:       agent,
		Token:          token,
		Password:       o.RouterPass,
		SSHKey:         o.PrimaryKey,
		OperatorPubKey: pub,
		MTLSCA:         mtlsCA,
		MTLSCert:       mtlsCert,
		MTLSKey:        mtlsKey,
	})
}
