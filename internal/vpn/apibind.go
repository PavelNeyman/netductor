package vpn

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// APIBind sets listen address for netductor-api (legacy freshvps-api units cleaned up).
// mode: localhost | detect | <ip>
func APIBind(mode string, openUFW bool) error {
	port := "8787"
	if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "api_port")); err == nil {
		port = strings.TrimSpace(string(b))
	}
	if v := os.Getenv("VPN_API_PORT"); v != "" {
		port = v
	}
	bind := "127.0.0.1"
	switch mode {
	case "", "localhost", "127.0.0.1":
		bind = "127.0.0.1"
	case "detect":
		bind = "0.0.0.0"
	default:
		bind = mode
	}
	_ = os.MkdirAll(paths.EtcDir(), 0o755)
	if err := os.WriteFile(filepath.Join(paths.EtcDir(), "api_bind"), []byte(bind+"\n"), 0o644); err != nil {
		return err
	}
	drop := func(unit string) {
		d := fmt.Sprintf("/etc/systemd/system/%s.service.d", unit)
		_ = os.MkdirAll(d, 0o755)
		conf := fmt.Sprintf("[Service]\nEnvironment=VPN_API_BIND=%s\nEnvironment=VPN_API_PORT=%s\nEnvironment=NETDUCTOR_API_BIND=%s\nEnvironment=NETDUCTOR_API_PORT=%s\n",
			bind, port, bind, port)
		// netductor parallel uses 8790 by default — only override bind for legacy 8787 unit
		if unit == "netductor-api" {
			// keep port from unit file; only set bind via env if serve reads VPN_API_BIND
			conf = fmt.Sprintf("[Service]\nEnvironment=VPN_API_BIND=%s\nEnvironment=NETDUCTOR_API_BIND=%s\n", bind, bind)
		}
		_ = os.WriteFile(filepath.Join(d, "bind.conf"), []byte(conf), 0o644)
		_ = exec.Command("systemctl", "daemon-reload").Run()
		_ = exec.Command("systemctl", "try-restart", unit).Run()
	}
	drop("freshvps-api")
	drop("netductor-api")
	fmt.Printf("API bind → %s (port file %s)\n", bind, port)
	if openUFW && bind != "127.0.0.1" {
		if _, err := exec.LookPath("ufw"); err == nil {
			_ = exec.Command("ufw", "allow", port+"/tcp", "comment", "netductor-api").Run()
			fmt.Printf("UFW opened %s/tcp\n", port)
		}
	} else if bind != "127.0.0.1" {
		fmt.Println("Note: firewall not changed; pass --ufw to open port")
	}
	return nil
}
