package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/httpx"
	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func registerRelayAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/secondary/device", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id", 400)
			return
		}
		for _, d := range secondary.List() {
			if d.ID == id {
				writeJSON(w, 200, d)
				return
			}
		}
		http.Error(w, "not found", 404)
	})

	mux.HandleFunc("/api/secondary/export", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		sni := r.URL.Query().Get("sni")
		if sni == "" {
			sni = "api.vk.me"
		}
		b, err := vpn.ExportSecondaryBundle(sni)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if id, tok, err := secondary.IssueToken("secondary"); err == nil {
			b.AgentID = id
			b.AgentToken = tok
			b.CoreAgentURL = "https://" + b.CoreIP + ":" + mtls.AgentTLSPort
		}
		writeJSON(w, 200, b)
	})
	mux.HandleFunc("/api/secondary/cmd", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		var body struct {
			ID  string `json:"id"`
			Cmd string `json:"cmd"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.ID == "" || body.Cmd == "" {
			http.Error(w, "id and cmd required", 400)
			return
		}
		if err := secondary.EnqueueCmd(body.ID, body.Cmd); err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "queued": body.Cmd, "id": body.ID})
	})
	mux.HandleFunc("/api/secondary/sync", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		ver := secondary.BumpConfigVer()
		writeJSON(w, 200, map[string]any{"ok": true, "config_ver": ver})
	})
	mux.HandleFunc("/api/secondary/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		devs := secondary.List()
		type row struct {
			secondary.Device
			Online bool `json:"online"`
		}
		var out []row
		for _, d := range devs {
			out = append(out, row{Device: d, Online: secondary.Online(d, 2*time.Minute)})
		}
		writeJSON(w, 200, map[string]any{
			"config_ver": secondary.ConfigVer(),
			"devices":    out,
		})
	})
	mux.HandleFunc("/api/secondary/exit", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		if r.Method == http.MethodPost {
			var body struct {
				Enabled *bool `json:"enabled"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			on := r.URL.Query().Get("enabled") == "1" || r.URL.Query().Get("on") == "1"
			if body.Enabled != nil {
				on = *body.Enabled
			}
			_ = secondary.SetExitEnabled(on)
			_ = vpn.ApplyConfig()
		}
		writeJSON(w, 200, map[string]any{"exit_enabled": secondary.ExitEnabled()})
	})
	mux.HandleFunc("/api/secondary/links", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		// mobile links from last heartbeat identity
		devs := secondary.List()
		var links []map[string]string
		reg, _ := vpn.ExportSecondaryBundle("ya.ru") // users only
		for _, d := range devs {
			if d.PublicIP == "" || d.PBK == "" {
				continue
			}
			for _, u := range reg.Users {
				links = append(links, map[string]string{
					"secondary": d.ID,
					"name":      u.Name,
					"link":      vpn.ClientLinkForSecondary(u.Name, u.UUID, d.PublicIP, d.PBK, d.SID, d.SNI),
				})
			}
		}
		writeJSON(w, 200, map[string]any{"links": links})
	})

	// Agent-facing (token = device token)
	mux.HandleFunc("/api/secondary/agent/heartbeat", handleSecondaryAgentHeartbeat)
	mux.HandleFunc("/api/secondary/agent/config", handleSecondaryAgentConfig)
	mux.HandleFunc("/api/secondary/agent/mtls/material", func(w http.ResponseWriter, r *http.Request) {
		tok := agentToken(r)
		d := secondary.FindByToken(tok)
		if d == nil {
			http.Error(w, "unauthorized", 401)
			return
		}
		ca, cert, key, err := mtls.ReadMaterial(d.ID)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		writeJSON(w, 200, map[string]any{"ca_pem": string(ca), "cert_pem": string(cert), "key_pem": string(key), "node_id": d.ID})
	})
}

func agentToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return r.Header.Get("X-Secondary-Token")
}

func handleSecondaryAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	tok := agentToken(r)
	if tok == "" || len(tok) < 16 {
		http.Error(w, "unauthorized", 401)
		return
	}
	raw, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	var in secondary.HeartbeatIn
	_ = json.Unmarshal(raw, &in)
	d, ver, err := secondary.Heartbeat(tok, in)
	if err != nil {
		http.Error(w, "unauthorized", 401)
		return
	}
	if strings.TrimSpace(in.CmdDone) != "" {
		ok := "✅"
		if !in.CmdOK {
			ok = "❌"
		}
		log := in.CmdLog
		if len(log) > 800 {
			log = log[len(log)-800:]
		}
		_ = notify.Telegram(fmt.Sprintf("%s <b>Secondary cmd</b> <code>%s</code> on <code>%s</code>\n<pre>%s</pre>", ok, in.CmdDone, d.ID, log))
	}
	// Register as node role=secondary so rename / fleet UI work
	host := d.Name
	if host == "" || host == "secondary" {
		host = "nd-secondary"
	}
	st := "online"
	_, _ = nodes.UpsertFromDevice(nodes.Node{
		ID: d.ID, Hostname: host, Role: "secondary", Kind: "vps",
		PublicIP: d.PublicIP, Status: st, LastSeen: time.Now().Unix(),
	})
	desired := ""
	if n, ok, err := nodes.Get(d.ID); err == nil && ok && n.DesiredHN != "" {
		desired = n.DesiredHN
	}
	cmds := secondary.TakeCmds(d.ID)
	writeJSON(w, 200, map[string]any{
		"ok": true, "config_ver": ver, "need_sync": in.ConfigVer < ver, "id": d.ID,
		"desired_hostname": desired,
		"commands":         cmds,
	})
}

func handleSecondaryAgentConfig(w http.ResponseWriter, r *http.Request) {
	tok := agentToken(r)
	if tok == "" || len(tok) < 16 || secondary.FindByToken(tok) == nil {
		http.Error(w, "unauthorized", 401)
		return
	}
	sni := vpn.ActiveSNI()
	if sni == "" {
		sni = "api.vk.me"
	}
	b, err := vpn.ExportSecondaryBundle(sni)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// do not issue new agent token on pull
	b.AgentToken = ""
	b.AgentID = ""
	writeJSON(w, 200, b)
}

// StartAgentPlane serves mTLS agent plane (:8789): secondary + edge + nvr.
// Default: mTLS only on :8789. Plain :8788 only if NETDUCTOR_PLAIN_AGENT=1 (emergency).
func StartAgentPlane() {
	// Shared agent plane (mTLS :8789): secondary + edge + nvr device APIs.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/secondary/agent/heartbeat", handleSecondaryAgentHeartbeat)
	mux.HandleFunc("/api/secondary/agent/config", handleSecondaryAgentConfig)
	mux.HandleFunc("/api/secondary/agent/mtls/material", func(w http.ResponseWriter, r *http.Request) {
		tok := agentToken(r)
		d := secondary.FindByToken(tok)
		if d == nil {
			http.Error(w, "unauthorized", 401)
			return
		}
		ca, cert, key, err := mtls.ReadMaterial(d.ID)
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		writeJSON(w, 200, map[string]any{"ca_pem": string(ca), "cert_pem": string(cert), "key_pem": string(key), "node_id": d.ID})
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/api/bot-status", func(w http.ResponseWriter, r *http.Request) {
		out, _ := exec.Command("systemctl", "is-active", "netductor-telegram-bot").Output()
		active := strings.TrimSpace(string(out)) == "active"
		writeJSON(w, 200, map[string]any{"ok": active, "bot": strings.TrimSpace(string(out))})
	})
	registerEdgeAPI(mux)
	registerNVRAPI(mux)

	// Ensure certs exist (auto-generate CA/server/client if missing).
	_ = mtls.EnsureAll(os.Getenv("NETDUCTOR_PUBLIC_IP"))

	if !mtls.ServerReady() {
		fmt.Fprintln(os.Stderr, "agent plane: mTLS certs missing — run: netductor mtls ensure")
	} else {
		tlsCfg, err := mtls.ServerTLSConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, "mtls: server config:", err)
		} else {
			tlsAddr := os.Getenv("NETDUCTOR_AGENT_MTLS")
			if tlsAddr == "" {
				tlsAddr = ":" + mtls.AgentTLSPort
			}
			go func() {
				lim := httpx.NewPlaneLimiter(180, time.Minute) // 180 req/min/IP
				srv := &http.Server{Addr: tlsAddr, Handler: lim.Middleware(withSecurity(mux)), TLSConfig: tlsCfg}
				fmt.Fprintln(os.Stderr, "agent plane mTLS on", tlsAddr)
				if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
					fmt.Fprintln(os.Stderr, "mtls serve:", err)
				}
			}()
		}
	}

	// Plain :8788 is emergency-only (never default).
	if os.Getenv("NETDUCTOR_PLAIN_AGENT") == "1" {
		addr := os.Getenv("NETDUCTOR_SECONDARY_API")
		if addr == "" {
			addr = ":8788"
		}
		fmt.Fprintln(os.Stderr, "WARN agent plane PLAIN on", addr, "(NETDUCTOR_PLAIN_AGENT=1)")
		go func() {
			lim := httpx.NewPlaneLimiter(60, time.Minute)
			_ = http.ListenAndServe(addr, lim.Middleware(withSecurity(mux)))
		}()
	}
}
