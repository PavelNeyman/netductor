package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/nvr"
)

func buildAPIMux() http.Handler {
	edge.EnsureDefaultTemplate()
	mux := http.NewServeMux()

	registerNodesAPI(mux)
	registerAddonsAPI(mux)
	registerSecondaryAPI(mux)
	registerSSHHostsAPI(mux)
	registerEdgeAPI(mux)
	registerNVRAPI(mux)
	nvr.StartBackground()
	registerSessionAPI(mux)
	registerGitAPI(mux)
	registerDNSAPI(mux)
	registerRegistryAPI(mux)
	registerVPNHTTP(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true, "service": "netductor", "version": version, "time": time.Now().UTC().Format(time.RFC3339)})
	})
	mux.HandleFunc("/api/bot-status", func(w http.ResponseWriter, r *http.Request) {
		out, _ := exec.Command("systemctl", "is-active", "netductor-telegram-bot").Output()
		active := strings.TrimSpace(string(out)) == "active"
		writeJSON(w, 200, map[string]any{"ok": active, "bot": strings.TrimSpace(string(out))})
	})

	// Product UI is netductor-op on Mac — no VPS /admin.
	mux.HandleFunc("/admin", legacyAdminGone)
	mux.HandleFunc("/admin/", legacyAdminGone)
	return mux
}

func legacyAdminGone(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(410)
	_, _ = w.Write([]byte(`{"ok":false,"error":"VPS admin UI removed; use netductor-op on Mac"}
`))
}

func runServe(args []string) {
	bind := envOr("NETDUCTOR_API_BIND", "127.0.0.1")
	port := envOr("NETDUCTOR_API_PORT", "8787")
	tlsCert := envOr("NETDUCTOR_TLS_CERT", "")
	tlsKey := envOr("NETDUCTOR_TLS_KEY", "")
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--bind":
			if i+1 < len(args) {
				bind, i = args[i+1], i+1
			}
		case "--port":
			if i+1 < len(args) {
				port, i = args[i+1], i+1
			}
		case "--help", "-h":
			printHelp()
			return
		}
	}
	if bind != "127.0.0.1" && bind != "localhost" {
		fmt.Fprintln(os.Stderr, "refusing non-local API bind (operator is netductor-op on Mac; NETDUCTOR_API_PUBLIC removed)")
		os.Exit(2)
	}
	mux := buildAPIMux()
	addr := bind + ":" + port
	fmt.Fprintf(os.Stderr, "netductor serve on http://%s API-only (Mac client = UI)\n", addr)
	_ = mtls.EnsureAll(os.Getenv("NETDUCTOR_PUBLIC_IP"))
	StartAgentPlane()
	if tlsCert != "" && tlsKey != "" {
		fmt.Fprintf(os.Stderr, "netductor serve TLS on https://%s\n", addr)
		if err := http.ListenAndServeTLS(addr, tlsCert, tlsKey, withSecurity(mux)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := http.ListenAndServe(addr, withSecurity(mux)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
