package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/tlsle"
)

func runTLS(args []string) {
	if len(args) == 0 || args[0] == "help" {
		fmt.Print(`netductor tls self-signed [hostname]
netductor tls le --email you@example.com --domains a.example.com,b.example.com
netductor tls le --email you@example.com --base netductor.example.com
  # → primary.<base> + i.<base> (HTTP-01 via certbot standalone)
netductor tls show

After LE: certs under /etc/letsencrypt/live/… and linked in secrets/tls/;
redirect unit gets HTTPS :443 when possible.
`)
		return
	}
	switch args[0] {
	case "self-signed":
		host := "localhost"
		if len(args) > 1 && args[1] != "" {
			host = args[1]
		} else if b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "public_hostname")); err == nil {
			if v := strings.TrimSpace(string(b)); v != "" {
				host = v
			}
		}
		dir := filepath.Join(paths.EtcDir(), "secrets", "tls")
		_ = os.MkdirAll(dir, 0o700)
		cert := filepath.Join(dir, "server.crt")
		key := filepath.Join(dir, "server.key")
		cmd := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048", "-sha256", "-days", "825",
			"-nodes", "-keyout", key, "-out", cert,
			"-subj", "/CN="+host+"/O=netductor",
			"-addext", "subjectAltName=DNS:"+host+",DNS:localhost,IP:127.0.0.1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Fprintln(os.Stderr, string(out), err)
			os.Exit(1)
		}
		_ = os.Chmod(key, 0o600)
		_ = os.Chmod(cert, 0o644)
		fmt.Println("wrote", cert)
		fmt.Println("wrote", key)
	case "le", "letsencrypt", "certbot":
		cfg := tlsle.Config{AgreeTOS: true}
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--email", "-m":
				if i+1 < len(args) {
					cfg.Email = args[i+1]
					i++
				}
			case "--domains", "-d":
				if i+1 < len(args) {
					cfg.Domains = splitCSV(args[i+1])
					i++
				}
			case "--base":
				if i+1 < len(args) {
					cfg.Base = args[i+1]
					i++
				}
			case "--staging":
				cfg.Staging = true
			case "--agree-tos":
				cfg.AgreeTOS = true
			}
		}
		if err := tlsle.Obtain(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "tls le:", err)
			os.Exit(1)
		}
	case "show":
		cert := filepath.Join(paths.EtcDir(), "secrets", "tls", "server.crt")
		key := filepath.Join(paths.EtcDir(), "secrets", "tls", "server.key")
		fmt.Println("cert", cert, "exists", fileExists(cert))
		fmt.Println("key", key, "exists", fileExists(key))
		live := "/etc/letsencrypt/live"
		if entries, err := os.ReadDir(live); err == nil {
			for _, e := range entries {
				if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
					fmt.Println("letsencrypt live:", e.Name())
				}
			}
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown tls subcommand")
		os.Exit(2)
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
