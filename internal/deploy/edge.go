package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
)

type EdgeOpts struct {
	PrimaryHost string
	PrimaryUser string
	PrimaryKey  string
	RouterHost  string
	RouterUser  string
	RouterPass  string
	DeviceID    string
	ServerURL   string
	AgentArch   string
	Version     string
	AgentDir    string
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
		o.Version = "0.8.2"
	}
	if o.AgentArch == "" {
		o.AgentArch = "arm64"
	}
	if o.AgentDir == "" {
		home, _ := os.UserHomeDir()
		o.AgentDir = filepath.Join(home, ".cache", "netductor", "agents")
	}
	if o.ServerURL == "" && o.PrimaryHost != "" {
		o.ServerURL = "http://" + o.PrimaryHost + ":8787"
	}
	if o.ServerURL == "" {
		return fmt.Errorf("server URL or primary host required")
	}

	token := ""
	if o.PrimaryHost != "" && o.PrimaryKey != "" {
		out, err := runSSH("", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
			"cat /etc/netductor/secrets/edge_bootstrap_token 2>/dev/null")
		if err != nil {
			return fmt.Errorf("read bootstrap token from primary: %w\n%s", err, out)
		}
		token = strings.TrimSpace(out)
	}
	agent, err := EnsureAgentBinary(o.Version, o.AgentArch, o.AgentDir)
	if err != nil {
		return err
	}
	target := o.RouterUser + "@" + o.RouterHost
	fmt.Fprintln(os.Stderr, "==> edge provision", target, "→", o.ServerURL)
	return edge.Provision(edge.ProvisionOpts{
		SSHTarget: target,
		DeviceID:  o.DeviceID,
		ServerURL: o.ServerURL,
		AgentBin:  agent,
		Token:     token,
		Password:  o.RouterPass,
	})
}
