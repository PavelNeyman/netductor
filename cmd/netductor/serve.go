package main

import (
	"context"
	"fmt"
	ndver "github.com/PavelNeyman/netductor/internal/version"
	"net/http"

	"github.com/PavelNeyman/netductor/internal/httpx"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
	gitstore "github.com/PavelNeyman/netductor/internal/git"
	"github.com/PavelNeyman/netductor/internal/migrate"
	"github.com/PavelNeyman/netductor/internal/nvr"
	"github.com/PavelNeyman/netductor/internal/policy"
	"github.com/PavelNeyman/netductor/internal/stack"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func buildAPIMux() http.Handler {
	policy.ApplyHook = vpn.ApplyAccessPolicies
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
	registerBuildHostAPI(mux)
	gitstore.StartVpsBuildWorker()
	gitstore.StartShipWorker()
	_ = gitstore.EnsureGitOpsTimers()
	registerDNSAPI(mux)
	registerRegistryAPI(mux)
	registerVPNHTTP(mux)
	registerSvcPathsAPI(mux)
	registerFirewallAPI(mux)
	registerSecurityAPI(mux)
	registerLogsAPI(mux)
	registerStackAPI(mux)
	registerUpdateAPI(mux)
	registerMigrateAPI(mux)
	registerReleaseAPI(mux)
	registerFleetAPI(mux)
	registerSecondariesAPI(mux)
	registerCleanupAPI(mux)
	registerTGAlertsAPI(mux)
	registerPolicyAPI(mux)
	registerDevicesAPI(mux)
	registerIncidentAPI(mux)
	registerCanaryAPI(mux)
	registerSmokeAPI(mux)
	registerJobsAPI(mux)
	registerOnboardAPI(mux)

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !AgentPlaneReady() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"ok": false, "service": "netductor", "error": "mTLS agent plane unavailable"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "service": "netductor", "version": ndver.Release, "time": time.Now().UTC().Format(time.RFC3339)})
	})
	mux.HandleFunc("/api/bot-status", func(w http.ResponseWriter, r *http.Request) {
		out, _ := exec.Command("systemctl", "is-active", "netductor-telegram-bot").Output()
		active := strings.TrimSpace(string(out)) == "active"
		writeJSON(w, 200, map[string]any{"ok": active, "bot": strings.TrimSpace(string(out))})
	})

	// Product UI is netductor-op on Mac — no VPS /admin.
	mux.HandleFunc("/admin", legacyAdminGone)
	mux.HandleFunc("/admin/", legacyAdminGone)
	// R6: session gate for /api (agent paths exempt)
	return apiSessionGate(mux)
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
	// Crash recovery before API. If recovery restores the previous pair, this
	// process image is still the new executable mapped before recovery; re-exec
	// the binary now on disk before opening listeners.
	rolledBack, recoverErr := stack.RecoverIncompleteApplyForStartup()
	if recoverErr != nil {
		fmt.Fprintln(os.Stderr, recoverErr)
		os.Exit(1)
	}
	diskExe := "/usr/local/bin/netductor"
	procInfo, procErr := os.Stat("/proc/self/exe")
	diskInfo, diskErr := os.Stat(diskExe)
	if procErr != nil || diskErr != nil {
		fmt.Fprintln(os.Stderr, "verify executable inode after recovery:", procErr, diskErr)
		os.Exit(1)
	}
	if rolledBack || !os.SameFile(procInfo, diskInfo) {
		if err := syscall.Exec(diskExe, os.Args, os.Environ()); err != nil {
			fmt.Fprintln(os.Stderr, "re-exec restored netductor binary:", err)
			os.Exit(1)
		}
		os.Exit(1)
	}
	// Startup migrations and post-check are mandatory before Ready.
	mr, merr := migrate.RunAll(context.Background())
	if merr != nil {
		fmt.Fprintln(os.Stderr, "startup migrate:", merr)
		os.Exit(1)
	}
	if len(mr.Ran) > 0 {
		fmt.Fprintln(os.Stderr, "startup migrate ran:", strings.Join(mr.Ran, ", "))
	}
	mux := buildAPIMux()
	addr := bind + ":" + port
	fmt.Fprintf(os.Stderr, "netductor serve on http://%s API-only (Mac client = UI)\n", addr)
	if err := StartAgentPlane(); err != nil {
		fmt.Fprintln(os.Stderr, "start mTLS agent plane:", err)
		os.Exit(1)
	}
	srv := httpx.NewServer(addr, withSecurity(mux))
	var err error
	if tlsCert != "" && tlsKey != "" {
		fmt.Fprintf(os.Stderr, "netductor serve TLS on https://%s\n", addr)
		err = srv.ListenAndServeTLS(tlsCert, tlsKey)
	} else {
		err = srv.ListenAndServe()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
