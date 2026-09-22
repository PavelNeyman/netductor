package install

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// InstallLampac pulls optional media aggregator (Docker). Off by default.
// Network: bound to 127.0.0.1 only — not on public WAN. Reach via VPN
// (proxy mode → http://127.0.0.1:9118 on server) or SSH -L 9118:127.0.0.1:9118.
func InstallLampac() error {
	if err := ensureDocker(); err != nil {
		return fmt.Errorf("docker for lampac: %w", err)
	}
	_ = os.MkdirAll(filepath.Join(paths.OptDir(), "lampac"), 0o755)
	_ = exec.Command("docker", "rm", "-f", "netductor-lampac").Run()

	img := env("NETDUCTOR_LAMPAC_IMAGE", "ghcr.io/immisterio/lampac:latest")
	// fallback list if primary image 404s
	alts := []string{
		img,
		"ghcr.io/lampac-nextgen/lampac:latest",
		"immisterio/lampac:latest",
	}

	port := env("NETDUCTOR_LAMPAC_PORT", "9118")
	bind := env("NETDUCTOR_LAMPAC_BIND", "127.0.0.1")
	publish := bind + ":" + port + ":9118"

	var lastErr error
	var used string
	for _, image := range alts {
		_ = exec.Command("docker", "rm", "-f", "netductor-lampac").Run()
		cmd := exec.Command("docker", "run", "-d",
			"--name", "netductor-lampac",
			"--restart", "unless-stopped",
			"-p", publish,
			"-v", filepath.Join(paths.OptDir(), "lampac")+":/home",
			image,
		)
		out, err := cmd.CombinedOutput()
		if err == nil {
			used = image
			lastErr = nil
			break
		}
		lastErr = fmt.Errorf("%s: %s", image, strings.TrimSpace(string(out)))
	}
	if lastErr != nil {
		return fmt.Errorf("lampac: %w", lastErr)
	}

	lockLampacToLocalhost(port)
	MarkComponentInstalled("lampac")
	fmt.Fprintf(os.Stderr, "lampac on %s:%s only (not public WAN); image %s\n", bind, port, used)
	return nil
}

func ensureDocker() error {
	if _, err := exec.LookPath("docker"); err == nil {
		if exec.Command("docker", "info").Run() == nil {
			return nil
		}
		_ = exec.Command("systemctl", "enable", "--now", "docker").Run()
		_ = exec.Command("systemctl", "start", "docker").Run()
		time.Sleep(2 * time.Second)
		if exec.Command("docker", "info").Run() == nil {
			return nil
		}
	}
	// Debian/Ubuntu: docker.io = daemon; docker-cli = client (Trixie splits them)
	_ = aptInstall("docker.io")
	_ = aptInstall("docker-cli")
	if _, err := exec.LookPath("docker"); err != nil {
		fmt.Fprintln(os.Stderr, "apt docker packages missing CLI, trying get.docker.com…")
		script := exec.Command("sh", "-c", "curl -fsSL https://get.docker.com | sh || wget -qO- https://get.docker.com | sh")
		script.Stdout, script.Stderr = os.Stdout, os.Stderr
		_ = script.Run()
	}
	_ = exec.Command("systemctl", "enable", "--now", "docker").Run()
	_ = exec.Command("systemctl", "start", "docker").Run()
	for i := 0; i < 15; i++ {
		if exec.Command("docker", "info").Run() == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("docker not installed")
	}
	return fmt.Errorf("docker installed but daemon not ready")
}

func lockLampacToLocalhost(port string) {
	if out, err := exec.Command("ufw", "status").Output(); err == nil && strings.Contains(string(out), "active") {
		_ = exec.Command("ufw", "deny", port+"/tcp").Run()
		_ = exec.Command("ufw", "allow", "from", "127.0.0.1", "to", "any", "port", port, "proto", "tcp").Run()
	}
	if err := exec.Command("iptables", "-C", "INPUT", "-i", "lo", "-p", "tcp", "--dport", port, "-j", "ACCEPT").Run(); err != nil {
		_ = exec.Command("iptables", "-I", "INPUT", "1", "-i", "lo", "-p", "tcp", "--dport", port, "-j", "ACCEPT").Run()
	}
	if err := exec.Command("iptables", "-C", "INPUT", "-p", "tcp", "--dport", port, "-j", "DROP").Run(); err != nil {
		_ = exec.Command("iptables", "-A", "INPUT", "-p", "tcp", "--dport", port, "-j", "DROP").Run()
	}
}
