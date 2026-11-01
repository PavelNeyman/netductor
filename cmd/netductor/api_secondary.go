package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PavelNeyman/netductor/internal/httpx"
	"github.com/PavelNeyman/netductor/internal/integrity"
	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/notify"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/svcpaths"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func registerSecondaryAPI(mux *http.ServeMux) {
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
		id, tok, err := secondary.IssueToken("secondary")
		if err != nil {
			http.Error(w, "failed to issue secondary agent credentials", http.StatusInternalServerError)
			return
		}
		b.AgentID = id
		b.AgentToken = tok
		b.CoreAgentURL = "https://" + b.CoreIP + ":" + mtls.AgentTLSPort
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
			if !secondaryLinkComplete(d.PublicIP, d.PBK, d.SID) {
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
	mux.HandleFunc("/api/secondary/agent/backup/latest", handleSecondaryBackupLatest)
	mux.HandleFunc("/api/secondary/agent/backup/key", handleSecondaryBackupKey)
	mux.HandleFunc("/api/secondary/agent/mtls/material", func(w http.ResponseWriter, r *http.Request) {
		tok := agentToken(r)
		d := secondary.FindByToken(tok)
		if d == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
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
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tok := agentToken(r)
	if tok == "" || len(tok) < 16 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		http.Error(w, "invalid heartbeat body", http.StatusBadRequest)
		return
	}
	var in secondary.HeartbeatIn
	if err := json.Unmarshal(raw, &in); err != nil {
		http.Error(w, "invalid heartbeat JSON", http.StatusBadRequest)
		return
	}
	d, ver, err := secondary.Heartbeat(tok, in)
	if err != nil {
		if errors.Is(err, secondary.ErrUnauthorized) || errors.Is(err, os.ErrNotExist) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "heartbeat persistence failed", http.StatusInternalServerError)
		return
	}
	if fp := strings.TrimSpace(in.RecoveryFP); fp != "" {
		secondary.StoreRecoveryPeerFingerprint(d.ID, fp)
	}
	if strings.TrimSpace(in.CmdDone) != "" {
		ok := "✅"
		if !in.CmdOK {
			ok = "❌"
		}
		log := redactSecondaryCmdLog(in.CmdDone, in.CmdLog)
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
	// Do not AllowAgentMTLSFromIP on every heartbeat — spammed ufw + re-opened WAN :8789.
	// Secondary agent uses SP (10.87.10.2); public IP allow is optional via api-allow.cidr / join.
	desired := ""
	if n, ok, err := nodes.Get(d.ID); err == nil && ok && n.DesiredHN != "" {
		desired = n.DesiredHN
	}
	cmds, err := secondary.TakeCmds(d.ID)
	if err != nil {
		http.Error(w, "secondary command queue unavailable", http.StatusInternalServerError)
		return
	}
	pol := svcpaths.LoadPolicy()
	writeJSON(w, 200, map[string]any{
		"ok": true, "config_ver": ver, "need_sync": in.ConfigVer < ver, "id": d.ID,
		"desired_hostname": desired,
		"desired_release":  "", /* operator-driven upgrades only */
		"commands":         cmds,
		"failover_policy":  pol,
	})
}

func handleSecondaryAgentConfig(w http.ResponseWriter, r *http.Request) {
	tok := agentToken(r)
	if tok == "" || len(tok) < 16 || secondary.FindByToken(tok) == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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
// Default: mTLS only on :8789. Plain :8788 unsupported.
var agentPlaneReady atomic.Bool

func AgentPlaneReady() bool { return agentPlaneReady.Load() }

func StartAgentPlane() error {
	// Shared agent plane (mTLS :8789): secondary + edge + nvr device APIs.
	mux := http.NewServeMux()
	mux.HandleFunc("/api/secondary/agent/heartbeat", handleSecondaryAgentHeartbeat)
	mux.HandleFunc("/api/secondary/agent/config", handleSecondaryAgentConfig)
	mux.HandleFunc("/api/secondary/agent/backup/latest", handleSecondaryBackupLatest)
	mux.HandleFunc("/api/secondary/agent/backup/key", handleSecondaryBackupKey)
	mux.HandleFunc("/api/secondary/agent/mtls/material", func(w http.ResponseWriter, r *http.Request) {
		tok := agentToken(r)
		d := secondary.FindByToken(tok)
		if d == nil { http.Error(w, "unauthorized", http.StatusUnauthorized); return }
		ca, cert, key, err := mtls.ReadMaterial(d.ID)
		if err != nil { http.Error(w, err.Error(), 404); return }
		writeJSON(w, 200, map[string]any{"ca_pem": string(ca), "cert_pem": string(cert), "key_pem": string(key), "node_id": d.ID})
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte("ok\n")) })
	mux.HandleFunc("/api/bot-status", func(w http.ResponseWriter, r *http.Request) {
		out, _ := exec.Command("systemctl", "is-active", "netductor-telegram-bot").Output()
		active := strings.TrimSpace(string(out)) == "active"
		writeJSON(w, 200, map[string]any{"ok": active, "bot": strings.TrimSpace(string(out))})
	})
	registerEdgeAPI(mux)
	registerBuildHostAgentAPI(mux)
	registerNVRAPI(mux)

	// Refuse startup before the main API becomes ready if mTLS cannot be served.
	if os.Getenv("NETDUCTOR_PLAIN_AGENT") == "1" {
		return fmt.Errorf("NETDUCTOR_PLAIN_AGENT is unsupported; only mTLS agent plane is allowed")
	}
	if err := mtls.EnsureAll(os.Getenv("NETDUCTOR_PUBLIC_IP")); err != nil { return fmt.Errorf("mtls.EnsureAll: %w", err) }
	if !mtls.ServerReady() { return fmt.Errorf("agent plane mTLS certificates are missing") }
	tlsCfg, err := mtls.ServerTLSConfig()
	if err != nil { return fmt.Errorf("mtls server config: %w", err) }
	tlsAddr := strings.TrimSpace(os.Getenv("NETDUCTOR_AGENT_MTLS"))
	if tlsAddr == "" { tlsAddr = ":" + mtls.AgentTLSPort }
	listener, err := net.Listen("tcp", tlsAddr)
	if err != nil { return fmt.Errorf("agent plane listen %s: %w", tlsAddr, err) }
	lim := httpx.NewPlaneLimiter(180, time.Minute)
	srv := &http.Server{
		Addr: tlsAddr, Handler: lim.Middleware(withSecurity(mux)), TLSConfig: tlsCfg,
		ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second,
		WriteTimeout: 120 * time.Second, IdleTimeout: 120 * time.Second,
	}
	agentPlaneReady.Store(true)
	go func() {
		fmt.Fprintln(os.Stderr, "agent plane mTLS on", tlsAddr)
		if err := srv.ServeTLS(listener, "", ""); err != nil && err != http.ErrServerClosed { fmt.Fprintln(os.Stderr, "mtls serve:", err) }
		agentPlaneReady.Store(false)
	}()
	return nil
}
func handleSecondaryBackupLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", 405)
		return
	}
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	tok = strings.TrimSpace(tok)
	if secondary.FindByToken(tok) == nil {
		http.Error(w, "unauthorized", 401)
		return
	}
	dir := filepath.Join(paths.StateDir(), "backups")
	ents, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "no backups", 404)
		return
	}
	var latest string
	var latestT int64
	for _, e := range ents {
		n := e.Name()
		if !strings.HasSuffix(n, ".ndenc") && !strings.HasSuffix(n, ".tar.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Unix() >= latestT {
			latestT = info.ModTime().Unix()
			latest = filepath.Join(dir, n)
		}
	}
	if latest == "" {
		http.Error(w, "no backups", 404)
		return
	}
	// sidecars as headers
	if b, err := os.ReadFile(filepath.Join(dir, "COMPONENTS.txt")); err == nil {
		w.Header().Set("X-Netductor-Components", strings.ReplaceAll(strings.TrimSpace(string(b)), "\n", ","))
	}
	w.Header().Set("X-Netductor-Backup-Name", filepath.Base(latest))
	if sum, err := fileSHA256Hex(latest); err == nil {
		w.Header().Set("X-Netductor-Backup-SHA256", sum)
	}
	if st, err := os.Stat(latest); err == nil {
		w.Header().Set("X-Netductor-Backup-Size", fmt.Sprintf("%d", st.Size()))
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+safeAttachmentFilename(filepath.Base(latest))+"\"")
	http.ServeFile(w, r, latest)
}

func fileSHA256Hex(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func handleSecondaryBackupKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", 405)
		return
	}
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	tok = strings.TrimSpace(tok)
	if secondary.FindByToken(tok) == nil {
		http.Error(w, "unauthorized", 401)
		return
	}
	keyPath := filepath.Join(paths.StateDir(), "backups", "BACKUP_KEY.txt")
	b, err := os.ReadFile(keyPath)
	if err != nil {
		// fallback secret
		b, err = os.ReadFile(filepath.Join(paths.EtcDir(), "secrets", "backup_key"))
		if err != nil {
			http.Error(w, "no key", 404)
			return
		}
	}
	w.Header().Set("Content-Type", "text/plain")
	_, _ = w.Write(b)
}

// redactSecondaryCmdLog keeps TG alerts free of full authorized_keys lines.
func redactSecondaryCmdLog(cmd, log string) string {
	cmd = strings.TrimSpace(cmd)
	base := cmd
	if i := strings.IndexByte(cmd, ':'); i > 0 {
		base = cmd[:i]
	}
	switch base {
	case "ssh_keys", "ssh_allowlist":
		return integrity.SummarizeAuthorizedKeysForNotify(log)
	}
	if len(log) > 800 {
		return log[len(log)-800:]
	}
	return log
}

func secondaryLinkComplete(publicIP, pbk, sid string) bool {
	return publicIP != "" && pbk != "" && sid != ""
}
