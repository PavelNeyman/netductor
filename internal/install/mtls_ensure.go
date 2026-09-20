package install

import (
	"os"

	"github.com/PavelNeyman/netductor/internal/mtls"
)

func ensureMTLSAtInstall() error {
	ip := os.Getenv("NETDUCTOR_PUBLIC_IP")
	if ip == "" {
		// best-effort; EnsureAll still creates CA/server with hostname SAN
		ip = ""
	}
	return mtls.EnsureAll(ip)
}
