package fleet

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/PavelNeyman/netductor/internal/nodes"
)

// ApplyLampac installs/starts Lampac on the preferred node (policy.LampacNodeID).
// When preferred is remote (secondary), runs via SSH after syncing data.
func ApplyLampac() error {
	p := LoadPolicy()
	id := p.LampacNodeID
	if id == "" {
		return fmt.Errorf("lampac_node unset — fleet bootstrap or set-service lampac")
	}
	n, ok, err := nodes.Get(id)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("lampac node %s not in registry", id)
	}
	localIP := publicIPGuess()
	remote := n.PublicIP != "" && n.PublicIP != localIP && n.Role == "relay"
	if !remote {
		// local install
		cmd := exec.Command("netductor", "install", "lampac")
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		return cmd.Run()
	}
	// ensure data on peer
	_ = SyncPaths("root@" + n.PublicIP)
	script := `set -e
export DEBIAN_FRONTEND=noninteractive
if ! command -v docker >/dev/null; then
  apt-get update -qq
  apt-get install -y -qq docker.io docker-cli >/dev/null
  systemctl enable --now docker
fi
mkdir -p /opt/netductor/lampac
docker rm -f netductor-lampac 2>/dev/null || true
docker pull ghcr.io/lampac-nextgen/lampac:latest
docker run -d --name netductor-lampac --restart unless-stopped \
  -p 127.0.0.1:9118:9118 \
  -v /opt/netductor/lampac:/home \
  ghcr.io/lampac-nextgen/lampac:latest
docker ps --filter name=netductor-lampac --format '{{.Names}} {{.Status}}'
`
	cmd := exec.Command("ssh",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "BatchMode=yes",
		"root@"+n.PublicIP, "bash", "-s")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	fmt.Print(string(out))
	if err != nil {
		return fmt.Errorf("remote lampac: %w", err)
	}
	return nil
}

func publicIPGuess() string {
	out, err := exec.Command("bash", "-c", `curl -4 -fsS --max-time 3 https://ifconfig.me 2>/dev/null || true`).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
