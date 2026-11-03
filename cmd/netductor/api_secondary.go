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

	"golang.org/x/sys/unix"

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

type secondaryDeviceAPIView struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	PublicIP      string         `json:"public_ip"`
	PBK           string         `json:"pbk"`
	SID           string         `json:"sid"`
	SNI           string         `json:"sni"`
	Version       string         `json:"version"`
	SingBoxOK     bool           `json:"singbox_ok"`
	UplinkOK      bool           `json:"uplink_ok"`
	ConfigVer     int            `json:"config_ver"`
	CPUPercent    float64        `json:"cpu_percent"`
	MemUsedMB     int64          `json:"mem_used_mb"`
	MemTotalMB    int64          `json:"mem_total_mb"`
	Load1         float64        `json:"load1"`
	MismatchTotal int            `json:"mismatch_total,omitempty"`
	MismatchByIP  map[string]int `json:"mismatch_by_ip,omitempty"`
	RecoveryFP    string         `json:"recovery_fp,omitempty"`
	PendingCmds   []string       `json:"pending_cmds,omitempty"`
	LastCmd       string         `json:"last_cmd,omitempty"`
	LastCmdAt     time.Time      `json:"last_cmd_at,omitempty"`
	LastCmdOK     bool           `json:"last_cmd_ok,omitempty"`
	LastCmdLog    string         `json:"last_cmd_log,omitempty"`
	LastSeen      time.Time      `json:"last_seen"`
	CreatedAt     time.Time      `json:"created_at"`
}

func toSecondaryDeviceAPIView(d secondary.Device) secondaryDeviceAPIView {
	return secondaryDeviceAPIView{
		ID: d.ID, Name: d.Name, PublicIP: d.PublicIP, PBK: d.PBK, SID: d.SID, SNI: d.SNI,
		Version: d.Version, SingBoxOK: d.SingBoxOK, UplinkOK: d.UplinkOK, ConfigVer: d.ConfigVer,
		CPUPercent: d.CPUPercent, MemUsedMB: d.MemUsedMB, MemTotalMB: d.MemTotalMB, Load1: d.Load1,
		MismatchTotal: d.MismatchTotal, MismatchByIP: d.MismatchByIP, RecoveryFP: d.RecoveryFP,
		PendingCmds: d.PendingCmds, LastCmd: d.LastCmd, LastCmdAt: d.LastCmdAt, LastCmdOK: d.LastCmdOK,
		LastCmdLog: redactSecondaryCmdLog(d.LastCmd, d.LastCmdLog), LastSeen: d.LastSeen, CreatedAt: d.CreatedAt,
	}
}

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
		devs, err := secondary.ListE()
		if err != nil {
			http.Error(w, "secondary registry unavailable", http.StatusInternalServerError)
			return
		}
		for _, d := range devs {
			if d.ID == id {
				writeJSON(w, 200, toSecondaryDeviceAPIView(d))
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
			switch {
			case errors.Is(err, secondary.ErrInvalidCommand):
				http.Error(w, err.Error(), http.StatusBadRequest)
			case errors.Is(err, os.ErrNotExist):
				http.Error(w, "secondary device not found", http.StatusNotFound)
			default:
				http.Error(w, "failed to persist secondary command", http.StatusInternalServerError)
			}
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "queued": body.Cmd, "id": body.ID})
	})
	mux.HandleFunc("/api/secondary/sync", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		ver, err := secondary.BumpConfigVer()
		if err != nil {
			http.Error(w, "failed to persist secondary config version", http.StatusInternalServerError)
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "config_ver": ver})
	})
	mux.HandleFunc("/api/secondary/status", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		devs, err := secondary.ListE()
		if err != nil {
			http.Error(w, "secondary registry unavailable", http.StatusInternalServerError)
			return
		}
		type row struct {
			secondaryDeviceAPIView
			Online bool `json:"online"`
		}
		var out []row
		for _, d := range devs {
			out = append(out, row{secondaryDeviceAPIView: toSecondaryDeviceAPIView(d), Online: secondary.Online(d, 2*time.Minute)})
		}
		configVer, err := secondary.ConfigVerE()
		if err != nil {
			http.Error(w, "secondary registry unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, 200, map[string]any{
			"config_ver": configVer,
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
			if err := secondary.SetExitEnabled(on); err != nil {
				http.Error(w, "failed to persist secondary exit setting", http.StatusInternalServerError)
				return
			}
			if err := vpn.ApplyConfig(); err != nil {
				http.Error(w, "secondary exit setting saved but VPN apply failed: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}
		exitEnabled, err := secondary.ExitEnabledE()
		if err != nil {
			http.Error(w, "secondary registry unavailable", http.StatusInternalServerError)
			return
		}
		writeJSON(w, 200, map[string]any{"exit_enabled": exitEnabled})
	})
	mux.HandleFunc("/api/secondary/links", func(w http.ResponseWriter, r *http.Request) {
		if !requireSession(w, r) {
			return
		}
		// mobile links from last heartbeat identity
		devs, err := secondary.ListE()
		if err != nil {
			http.Error(w, "secondary registry unavailable", http.StatusInternalServerError)
			return
		}
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
		d, ok := lookupSecondaryAgentDevice(w, r, tok)
		if !ok {
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

func secondaryAgentCertificateMatches(r *http.Request, id string) bool {
	if r == nil || r.TLS == nil || id == "" || len(r.TLS.PeerCertificates) == 0 {
		return false
	}
	cert := r.TLS.PeerCertificates[0]
	return cert != nil && cert.Subject.CommonName == "netductor-agent-"+id
}

func lookupSecondaryAgentDevice(w http.ResponseWriter, r *http.Request, token string) (*secondary.Device, bool) {
	d, err := secondary.LookupByToken(token)
	if err != nil {
		http.Error(w, "secondary registry unavailable", http.StatusInternalServerError)
		return nil, false
	}
	if d == nil || !secondaryAgentCertificateMatches(r, d.ID) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil, false
	}
	return d, true
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
	authenticated, ok := lookupSecondaryAgentDevice(w, r, tok)
	if !ok {
		return
	}
	peerSerial := ""
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 && r.TLS.PeerCertificates[0] != nil && r.TLS.PeerCertificates[0].SerialNumber != nil {
		peerSerial = strings.ToLower(r.TLS.PeerCertificates[0].SerialNumber.Text(16))
	}
	d, ver, err := secondary.HeartbeatWithTLSIdentity(tok, in, peerSerial)
	if err != nil {
		if errors.Is(err, secondary.ErrUnauthorized) || errors.Is(err, os.ErrNotExist) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		http.Error(w, "heartbeat persistence failed", http.StatusInternalServerError)
		return
	}
	if d.ID != authenticated.ID {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	rolloverConfirmed, err := mtls.ConfirmCARolloverClient(d.ID, peerSerial)
	if err != nil {
		http.Error(w, "CA rollover adoption persistence failed", http.StatusInternalServerError)
		return
	}
	materialConfirmed := false
	if peerSerial != "" {
		info, err := mtls.ReadClientInfo(d.ID)
		materialConfirmed = err == nil && !info.Revoked && strings.EqualFold(info.Serial, peerSerial)
	}
	if fp := strings.TrimSpace(in.RecoveryFP); fp != "" {
		if err := secondary.StoreRecoveryPeerFingerprint(d.ID, fp); err != nil {
			http.Error(w, "recovery fingerprint persistence failed", http.StatusInternalServerError)
			return
		}
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
		"mtls_rollover_confirmed": rolloverConfirmed,
		"mtls_material_confirmed":  materialConfirmed,
	})
}

func handleSecondaryAgentConfig(w http.ResponseWriter, r *http.Request) {
	tok := agentToken(r)
	if _, ok := lookupSecondaryAgentDevice(w, r, tok); !ok {
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
		d, ok := lookupSecondaryAgentDevice(w, r, tok)
		if !ok { return }
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
func openRegularFileNoFollow(path string) (*os.File, unix.Stat_t, error) {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, unix.Stat_t{}, err
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return nil, unix.Stat_t{}, err
	}
	if st.Mode&unix.S_IFMT != unix.S_IFREG {
		_ = unix.Close(fd)
		return nil, unix.Stat_t{}, fmt.Errorf("not a regular file")
	}
	if st.Uid != uint32(os.Geteuid()) {
		_ = unix.Close(fd)
		return nil, unix.Stat_t{}, fmt.Errorf("file is not owned by the service user")
	}
	f := os.NewFile(uintptr(fd), path)
	if f == nil {
		_ = unix.Close(fd)
		return nil, unix.Stat_t{}, fmt.Errorf("invalid file descriptor")
	}
	return f, st, nil
}

func handleSecondaryBackupLatest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", 405)
		return
	}
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	tok = strings.TrimSpace(tok)
	if _, ok := lookupSecondaryAgentDevice(w, r, tok); !ok {
		return
	}
	dir := filepath.Join(paths.StateDir(), "backups")
	ents, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "no backups", 404)
		return
	}
	var latest string
	var latestT time.Time
	for _, e := range ents {
		n := e.Name()
		if !strings.HasSuffix(n, ".ndenc") && !strings.HasSuffix(n, ".tar.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		if latest == "" || info.ModTime().After(latestT) {
			latestT = info.ModTime()
			latest = filepath.Join(dir, n)
		}
	}
	if latest == "" {
		http.Error(w, "no backups", 404)
		return
	}
	f, st, err := openRegularFileNoFollow(latest)
	if err != nil {
		http.Error(w, "backup unavailable", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		http.Error(w, "backup read failed", http.StatusInternalServerError)
		return
	}
	// Sidecars as headers.
	if b, err := readSmallRegularFileNoFollow(filepath.Join(dir, "COMPONENTS.txt"), 64<<10); err == nil {
		w.Header().Set("X-Netductor-Components", strings.ReplaceAll(strings.TrimSpace(string(b)), "\n", ","))
	}
	name := filepath.Base(latest)
	w.Header().Set("X-Netductor-Backup-Name", name)
	w.Header().Set("X-Netductor-Backup-SHA256", hex.EncodeToString(h.Sum(nil)))
	w.Header().Set("X-Netductor-Backup-Size", fmt.Sprintf("%d", st.Size))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+safeAttachmentFilename(name)+"\"")
	http.ServeContent(w, r, name, time.Unix(st.Mtim.Sec, st.Mtim.Nsec), f)
}

func readSmallRegularFileNoFollow(path string, maxSize int64) ([]byte, error) {
	f, st, err := openRegularFileNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if st.Size < 0 || st.Size > maxSize {
		return nil, fmt.Errorf("file size exceeds %d bytes", maxSize)
	}
	b, err := io.ReadAll(io.LimitReader(f, maxSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxSize {
		return nil, fmt.Errorf("file size exceeds %d bytes", maxSize)
	}
	return b, nil
}

func handleSecondaryBackupKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", 405)
		return
	}
	tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	tok = strings.TrimSpace(tok)
	if _, ok := lookupSecondaryAgentDevice(w, r, tok); !ok {
		return
	}
	keyPath := filepath.Join(paths.StateDir(), "backups", "BACKUP_KEY.txt")
	b, err := readSmallRegularFileNoFollow(keyPath, 64<<10)
	if errors.Is(err, unix.ENOENT) {
		// Preserve the legacy fallback when the backup directory key is absent.
		b, err = readSmallRegularFileNoFollow(filepath.Join(paths.EtcDir(), "secrets", "backup_key"), 64<<10)
	}
	if err != nil {
		if errors.Is(err, unix.ENOENT) {
			http.Error(w, "no key", 404)
		} else {
			http.Error(w, "backup key unavailable", http.StatusInternalServerError)
		}
		return
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
