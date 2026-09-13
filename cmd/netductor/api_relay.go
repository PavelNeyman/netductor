package main

import (
	"fmt"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/relay"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func registerRelayAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/relay/device", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id", 400)
			return
		}
		for _, d := range relay.List() {
			if d.ID == id {
				writeJSON(w, 200, d)
				return
			}
		}
		http.Error(w, "not found", 404)
	})

	mux.HandleFunc("/api/relay/export", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		sni := r.URL.Query().Get("sni")
		if sni == "" {
			sni = "ya.ru"
		}
		b, err := vpn.ExportRelayBundle(sni)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		if id, tok, err := relay.IssueToken("relay"); err == nil {
			b.AgentID = id
			b.AgentToken = tok
			b.CoreAgentURL = "http://" + b.CoreIP + ":8788"
		}
		writeJSON(w, 200, b)
	})
	mux.HandleFunc("/api/relay/cmd", func(w http.ResponseWriter, r *http.Request) {
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
		if err := relay.EnqueueCmd(body.ID, body.Cmd); err != nil {
			http.Error(w, err.Error(), 404)
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "queued": body.Cmd, "id": body.ID})
	})
	mux.HandleFunc("/api/relay/sync", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		ver := relay.BumpConfigVer()
		writeJSON(w, 200, map[string]any{"ok": true, "config_ver": ver})
	})
	mux.HandleFunc("/api/relay/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		devs := relay.List()
		type row struct {
			relay.Device
			Online bool `json:"online"`
		}
		var out []row
		for _, d := range devs {
			out = append(out, row{Device: d, Online: relay.Online(d, 2*time.Minute)})
		}
		writeJSON(w, 200, map[string]any{
			"config_ver": relay.ConfigVer(),
			"devices":    out,
		})
	})
	mux.HandleFunc("/api/relay/exit", func(w http.ResponseWriter, r *http.Request) {
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
			_ = relay.SetExitEnabled(on)
			_ = vpn.ApplyConfig()
		}
		writeJSON(w, 200, map[string]any{"exit_enabled": relay.ExitEnabled()})
	})
	mux.HandleFunc("/api/relay/links", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		// mobile links from last heartbeat identity
		devs := relay.List()
		var links []map[string]string
		reg, _ := vpn.ExportRelayBundle("ya.ru") // users only
		for _, d := range devs {
			if d.PublicIP == "" || d.PBK == "" {
				continue
			}
			for _, u := range reg.Users {
				links = append(links, map[string]string{
					"relay": d.ID,
					"name":  u.Name,
					"link":  vpn.ClientLinkForRelay(u.Name, u.UUID, d.PublicIP, d.PBK, d.SID, d.SNI),
				})
			}
		}
		writeJSON(w, 200, map[string]any{"links": links})
	})

	// Agent-facing (token = device token)
	mux.HandleFunc("/api/relay/agent/heartbeat", handleRelayAgentHeartbeat)
	mux.HandleFunc("/api/relay/agent/config", handleRelayAgentConfig)
}

func agentToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return r.Header.Get("X-Relay-Token")
}

func handleRelayAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	tok := agentToken(r)
	if tok == "" {
		http.Error(w, "unauthorized", 401)
		return
	}
	raw, _ := io.ReadAll(r.Body)
	var in relay.HeartbeatIn
	_ = json.Unmarshal(raw, &in)
	d, ver, err := relay.Heartbeat(tok, in)
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
		_ = notify.Telegram(fmt.Sprintf("%s <b>Relay cmd</b> <code>%s</code> on <code>%s</code>\n<pre>%s</pre>", ok, in.CmdDone, d.ID, log))
	}
	// Register as node role=relay so rename / fleet UI work
	host := d.Name
	if host == "" || host == "relay" {
		host = "nd-relay-" + strings.ReplaceAll(d.PublicIP, ".", "-")
	}
	st := "online"
	_, _ = nodes.UpsertFromDevice(nodes.Node{
		ID: d.ID, Hostname: host, Role: "relay", Kind: "vps",
		PublicIP: d.PublicIP, Status: st, LastSeen: time.Now().Unix(),
	})
	desired := ""
	if n, ok, err := nodes.Get(d.ID); err == nil && ok && n.DesiredHN != "" {
		desired = n.DesiredHN
	}
	cmds := relay.TakeCmds(d.ID)
	writeJSON(w, 200, map[string]any{
		"ok": true, "config_ver": ver, "need_sync": in.ConfigVer < ver, "id": d.ID,
		"desired_hostname": desired,
		"commands": cmds,
	})
}

func handleRelayAgentConfig(w http.ResponseWriter, r *http.Request) {
	tok := agentToken(r)
	if relay.FindByToken(tok) == nil {
		http.Error(w, "unauthorized", 401)
		return
	}
	b, err := vpn.ExportRelayBundle("ya.ru")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	// do not issue new agent token on pull
	b.AgentToken = ""
	b.AgentID = ""
	writeJSON(w, 200, b)
}

// startRelayAgentListener binds :8788 for agent plane (all interfaces).
func startRelayAgentListener() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/relay/agent/heartbeat", handleRelayAgentHeartbeat)
	mux.HandleFunc("/api/relay/agent/config", handleRelayAgentConfig)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("ok\n"))
	})
	addr := os.Getenv("NETDUCTOR_RELAY_API")
	if addr == "" {
		addr = ":8788"
	}
	go func() {
		_ = http.ListenAndServe(addr, mux)
	}()
}
