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

	root := adminRoot()
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin/", http.StatusFound)
	})
	mux.Handle("/admin/", http.StripPrefix("/admin/", http.FileServer(http.Dir(root))))
	return mux
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
		if os.Getenv("NETDUCTOR_API_PUBLIC") != "1" {
			fmt.Fprintln(os.Stderr, "refusing non-local API bind without NETDUCTOR_API_PUBLIC=1")
			os.Exit(2)
		}
		if tlsCert == "" || tlsKey == "" {
			fmt.Fprintln(os.Stderr, "public API bind requires --tls-cert and --tls-key (or NETDUCTOR_TLS_*)")
			os.Exit(2)
		}
	}
	mux := buildAPIMux()
	root := adminRoot()
	addr := bind + ":" + port
	fmt.Fprintf(os.Stderr, "netductor serve on http://%s admin=%s\n", addr, root)
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
