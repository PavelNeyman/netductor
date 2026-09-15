package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

func runTLS(args []string) {
	if len(args) == 0 || args[0] == "help" {
		fmt.Print(`netductor tls self-signed [hostname]  — write TLS cert/key under secrets/tls/
netductor tls show                    — print env for HTTPS admin bind

Public admin (optional):
  NETDUCTOR_API_PUBLIC=1 NETDUCTOR_API_BIND=0.0.0.0 NETDUCTOR_API_PORT=443 \
  NETDUCTOR_TLS_CERT=... NETDUCTOR_TLS_KEY=... \
  systemctl restart netductor-api

Prefer Let's Encrypt (certbot) on the host; self-signed is for lab only.
`)
		return
	}
	switch args[0] {
	case "self-signed":
		host := "netductor.work.gd"
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
		// openssl one-shot
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
		fmt.Println("Set:")
		fmt.Printf("  export NETDUCTOR_TLS_CERT=%s\n  export NETDUCTOR_TLS_KEY=%s\n", cert, key)
		fmt.Println("  export NETDUCTOR_API_PUBLIC=1 NETDUCTOR_API_BIND=0.0.0.0 NETDUCTOR_API_PORT=443")
	case "show":
		cert := filepath.Join(paths.EtcDir(), "secrets", "tls", "server.crt")
		key := filepath.Join(paths.EtcDir(), "secrets", "tls", "server.key")
		fmt.Println("cert", cert, "exists", fileExists(cert))
		fmt.Println("key", key, "exists", fileExists(key))
	default:
		fmt.Fprintln(os.Stderr, "unknown tls subcommand")
		os.Exit(2)
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
