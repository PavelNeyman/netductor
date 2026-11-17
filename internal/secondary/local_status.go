package secondary

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/svcpaths"
	"github.com/PavelNeyman/netductor/internal/version"
)

// Local status HTTP is loopback-only (Phase 4 thin member API).
// Port 8791 — distinct from recovery arm :8790.
const localStatusAddr = "127.0.0.1:8791"

// LocalStatus is the read-only egress-node snapshot (no SoT, no mutations).
type LocalStatus struct {
	OK        bool   `json:"ok"`
	Role      string `json:"role"`
	Hostname  string `json:"hostname"`
	Version   string `json:"version"`
	PublicIP  string `json:"public_ip,omitempty"`
	SingBoxOK bool   `json:"singbox_ok"`
	UplinkOK  bool   `json:"uplink_ok"`
	AgentUnit string `json:"agent_unit,omitempty"`
	TS        string `json:"ts"`
	Note      string `json:"note,omitempty"`
}

// CollectLocalStatus gathers live facts without talking to control.
func CollectLocalStatus() LocalStatus {
	hn, _ := os.Hostname()
	st := LocalStatus{
		OK:       true,
		Role:     "egress",
		Hostname: hn,
		Version:  version.Release,
		TS:       time.Now().UTC().Format(time.RFC3339),
		Note:     "thin local status; mutations only via control or SSH",
	}
	if out, err := exec.Command("systemctl", "is-active", "sing-box").Output(); err == nil {
		st.SingBoxOK = strings.TrimSpace(string(out)) == "active"
	}
	if out, err := exec.Command("systemctl", "is-active", "netductor-secondary-agent").Output(); err == nil {
		st.AgentUnit = strings.TrimSpace(string(out))
	}
	st.PublicIP = publicIP()
	pSP, pPS, err := svcpaths.PrimaryTunnelAddrsStrict()
	if err != nil {
		st.UplinkOK = false
	} else if pSP != "" && exec.Command("ping", "-c", "1", "-W", "2", pSP).Run() == nil {
		st.UplinkOK = true
	} else if pPS != "" && exec.Command("ping", "-c", "1", "-W", "2", pPS).Run() == nil {
		st.UplinkOK = true
	}
	return st
}

// StartLocalStatusServer serves GET /status and /healthz on loopback only.
func StartLocalStatusServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method", 405)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CollectLocalStatus())
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/status", http.StatusFound)
	})
	go func() {
		ln, err := net.Listen("tcp", localStatusAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "local status listen %s: %v\n", localStatusAddr, err)
			return
		}
		fmt.Fprintf(os.Stderr, "local status http://%s/status (loopback only)\n", localStatusAddr)
		s := &http.Server{
			Handler:           mux,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      15 * time.Second,
		}
		if err := s.Serve(ln); err != nil {
			fmt.Fprintf(os.Stderr, "local status serve: %v\n", err)
		}
	}()
}
