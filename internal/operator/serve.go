package operator

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PavelNeyman/netductor/internal/opcatalog"
	"github.com/PavelNeyman/netductor/internal/deploy"
	"github.com/PavelNeyman/netductor/internal/operator/web"
)

// ServeOpts for localhost operator HTTP API + embed UI.
type ServeOpts struct {
	Bind  string // must be loopback
	Port  string
	Token string // empty → generate; or NETDUCTOR_OPERATOR_TOKEN
}

var fleetMu sync.Mutex

// Serve runs a blocking HTTP server (loopback only).
func Serve(o ServeOpts) error {
	bind := strings.TrimSpace(o.Bind)
	if bind == "" {
		bind = "127.0.0.1"
	}
	if !isLoopbackHost(bind) {
		return fmt.Errorf("operator serve: bind must be loopback (got %q)", bind)
	}
	port := strings.TrimSpace(o.Port)
	if port == "" {
		port = "7373"
	}
	token := strings.TrimSpace(o.Token)
	if token == "" {
		token = strings.TrimSpace(os.Getenv("NETDUCTOR_OPERATOR_TOKEN"))
	}
	if token == "" {
		var err error
		token, err = randomToken(16)
		if err != nil {
			return err
		}
	}
	if !safeOperatorToken(token) {
		return fmt.Errorf("operator token must be 16–128 chars [A-Za-z0-9_-] (reject HTML/shell metacharacters)")
	}

	addr := net.JoinHostPort(bind, port)
	mux := http.NewServeMux()
	mux.HandleFunc("/static/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method", 405)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/static/")
		name = filepath.Base(name) // no traversal
		switch name {
		case "app.css":
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case "app.js":
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		default:
			http.NotFound(w, r)
			return
		}
		b, err := web.FS.ReadFile(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			http.NotFound(w, r)
			return
		}
		b, err := web.FS.ReadFile("index.html")
		if err != nil {
			http.Error(w, "ui missing", 500)
			return
		}
		html := strings.Replace(string(b), "/*__ND_TOKEN__*/", token, 1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(html))
	})
	mux.HandleFunc("/v1/catalog", handleCatalog)
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/v1/meta", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", 405)
			return
		}
		home, _ := os.UserHomeDir()
		cred := filepath.Join(home, ".netductor", "credentials")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version":          deploy.Release,
			"bind":             addr,
			"credentials_dir":  cred,
			"endpoints":        []string{"/v1/fleet", "/v1/primary", "/v1/secondary", "/v1/credentials", "/v1/health", "/v1/meta"},
			"auth":             "X-Netductor-Token",
		})
	})
	mux.HandleFunc("/v1/fleet", func(w http.ResponseWriter, r *http.Request) { handleFleet(w, r, token) })
	mux.HandleFunc("/v1/primary", func(w http.ResponseWriter, r *http.Request) { handlePrimary(w, r, token) })
	mux.HandleFunc("/v1/secondary", func(w http.ResponseWriter, r *http.Request) { handleSecondary(w, r, token) })
	mux.HandleFunc("/v1/credentials", func(w http.ResponseWriter, r *http.Request) { handleCredentials(w, r, token) })
	mux.HandleFunc("/v1/edge", func(w http.ResponseWriter, r *http.Request) { handleEdge(w, r, token) })
	mux.HandleFunc("/v1/site", func(w http.ResponseWriter, r *http.Request) { handleSite(w, r, token) })
	mux.HandleFunc("/v1/mikrotik", func(w http.ResponseWriter, r *http.Request) { handleMikroTik(w, r, token) })
	mux.HandleFunc("/v1/node/", func(w http.ResponseWriter, r *http.Request) { ProxyNodeAPI(w, r, token) })
	mux.HandleFunc("/v1/tunnel/status", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "GET only", 405)
			return
		}
		if !requireToken(r, token) {
			http.Error(w, "unauthorized", 401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(TunnelStatus())
	})
	mux.HandleFunc("/v1/tunnel/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}
		if !requireToken(r, token) {
			http.Error(w, "unauthorized", 401)
			return
		}
		var body struct {
			Host      string `json:"host"`
			VPNHost   string `json:"vpn_host"`
			PreferVPN *bool  `json:"prefer_vpn"`
			User      string `json:"user"`
			Key       string `json:"key"`
			LocalPort string `json:"local_port"`
			SSHPort   string `json:"ssh_port"`
			APIBase   string `json:"api_base"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		prefer := true
		if body.PreferVPN != nil {
			prefer = *body.PreferVPN
		}
		opts := TunnelOpts{Host: body.Host, VPNHost: body.VPNHost, PreferVPN: prefer, User: body.User, Key: body.Key, LocalPort: body.LocalPort, SSHPort: body.SSHPort}
		res, err := EnsureAPIPath(opts, body.APIBase)
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(500)
			_ = json.NewEncoder(w).Encode(res)
			return
		}
		_ = json.NewEncoder(w).Encode(res)
	})
	mux.HandleFunc("/v1/tunnel/stop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", 405)
			return
		}
		if !requireToken(r, token) {
			http.Error(w, "unauthorized", 401)
			return
		}
		TunnelStop()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": TunnelStatus()})
	})
	mux.HandleFunc("/v1/session/issue", handleSessionIssue(token))
	mux.HandleFunc("/v1/session/local", handleSessionLocal(token))

	fmt.Fprintf(os.Stderr, "operator serve: http://%s/  (loopback only)\n", addr)
	fmt.Fprintf(os.Stderr, "operator token: %s  (header X-Netductor-Token)\n", token)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; base-uri 'none'; form-action 'self'; frame-ancestors 'none'; img-src 'self' data:; style-src 'self'; script-src 'self'; connect-src 'self'")
		mux.ServeHTTP(w, r)
	})
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Hour,
		IdleTimeout:       120 * time.Second,
	}
	return srv.ListenAndServe()
}

func isLoopbackHost(h string) bool {
	h = strings.TrimSpace(strings.ToLower(h))
	if h == "127.0.0.1" || h == "localhost" || h == "::1" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func safeOperatorToken(t string) bool {
	if len(t) < 16 || len(t) > 128 {
		return false
	}
	for _, c := range t {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return true
}

func tokenOK(want, got string) bool {
	if want == "" || got == "" {
		return false
	}
	a := sha256.Sum256([]byte(want))
	b := sha256.Sum256([]byte(got))
	return subtle.ConstantTimeCompare(a[:], b[:]) == 1
}

func requireToken(r *http.Request, token string) bool {
	got := r.Header.Get("X-Netductor-Token")
	if got == "" {
		got = r.Header.Get("Authorization")
		got = strings.TrimPrefix(got, "Bearer ")
		got = strings.TrimPrefix(got, "bearer ")
	}
	return tokenOK(token, strings.TrimSpace(got))
}

func streamStart(w http.ResponseWriter) (http.Flusher, bool) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	flusher, ok := w.(http.Flusher)
	return flusher, ok
}

func streamRep(w http.ResponseWriter, flusher http.Flusher, okFlush bool) Reporter {
	return func(st Step) {
		line := fmt.Sprintf("step %s: %s\n", st.ID, st.Message)
		if st.Err != "" {
			line = fmt.Sprintf("step %s ERROR: %s\n", st.ID, st.Err)
		} else if st.Done {
			line = fmt.Sprintf("step %s done: %s\n", st.ID, st.Message)
		}
		_, _ = io.WriteString(w, line)
		if okFlush && flusher != nil {
			flusher.Flush()
		}
	}
}

type fleetJSON struct {
	DoPrimary           bool   `json:"do_primary"`
	DoSecondary         bool   `json:"do_secondary"`
	PrimaryHost         string `json:"primary_host"`
	PrimaryUser         string `json:"primary_user"`
	PrimaryPassword     string `json:"primary_password"`
	SecondaryHost       string `json:"secondary_host"`
	SecondaryUser       string `json:"secondary_user"`
	SecondaryPassword   string `json:"secondary_password"`
	DomainBase          string `json:"domain_base"`
	LEEmail             string `json:"le_email"`
	CFProxy             bool   `json:"cf_proxy"`
	SNI                 string `json:"sni"`
	Key                 string `json:"key"`
	KeyPassphrase       string `json:"key_passphrase"`
	WithLampac          bool   `json:"with_lampac"`
	WithGit             bool   `json:"with_git"`
	TelegramToken       string `json:"tg_token"`
	TelegramAdminID     string `json:"tg_admin"`
	WithTelegram        bool   `json:"with_telegram"`
}

type primaryJSON struct {
	Host            string `json:"host"`
	User            string `json:"user"`
	Password        string `json:"password"`
	DomainBase      string `json:"domain_base"`
	LEEmail         string `json:"le_email"`
	CFProxy         bool   `json:"cf_proxy"`
	SNI             string `json:"sni"`
	Key             string `json:"key"`
	KeyPassphrase   string `json:"key_passphrase"`
	WithLampac      bool   `json:"with_lampac"`
	WithGit         bool   `json:"with_git"`
	TelegramToken   string `json:"tg_token"`
	TelegramAdminID string `json:"tg_admin"`
	WithTelegram    bool   `json:"with_telegram"`
}

type secondaryJSON struct {
	PrimaryHost       string `json:"primary_host"`
	PrimaryUser       string `json:"primary_user"`
	PrimaryKey        string `json:"primary_key"`
	PrimaryKeyPass    string `json:"primary_key_passphrase"`
	SecondaryHost     string `json:"secondary_host"`
	SecondaryUser     string `json:"secondary_user"`
	SecondaryPassword string `json:"secondary_password"`
	SNI               string `json:"sni"`
}

type credJSON struct {
	Role string `json:"role"`
	Host string `json:"host"`
	User string `json:"user"`
	Key  string `json:"key"`
	Pass string `json:"key_passphrase"`
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		http.Error(w, err.Error(), 400)
		return false
	}
	return true
}

func tryLockDeploy(w http.ResponseWriter) bool {
	if !fleetMu.TryLock() {
		http.Error(w, "another deploy is running", 409)
		return false
	}
	return true
}

func handleFleet(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if !requireToken(r, token) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var body fleetJSON
	if !decodeJSON(w, r, &body) {
		return
	}
	if body.DoPrimary && strings.TrimSpace(body.PrimaryHost) == "" {
		http.Error(w, "primary_host required", 400)
		return
	}
	if body.DoSecondary && strings.TrimSpace(body.SecondaryHost) == "" {
		http.Error(w, "secondary_host required", 400)
		return
	}
	if !body.DoPrimary && !body.DoSecondary {
		http.Error(w, "do_primary and/or do_secondary required", 400)
		return
	}
	if body.WithTelegram || strings.TrimSpace(body.TelegramToken) != "" || strings.TrimSpace(body.TelegramAdminID) != "" {
		if strings.TrimSpace(body.TelegramToken) == "" || strings.TrimSpace(body.TelegramAdminID) == "" {
			http.Error(w, "telegram requires tg_token and tg_admin", 400)
			return
		}
		if !body.DoPrimary {
			http.Error(w, "telegram installs on primary only (enable do_primary)", 400)
			return
		}
	}
	if !tryLockDeploy(w) {
		return
	}
	defer fleetMu.Unlock()

	pu := orDefault(body.PrimaryUser, "root")
	su := orDefault(body.SecondaryUser, "root")
	f := FleetSpec{
		DoPrimary:   body.DoPrimary,
		DoSecondary: body.DoSecondary,
		Primary: PrimarySpec{
			Host: body.PrimaryHost, User: pu, Password: body.PrimaryPassword,
			SNI: orDefault(body.SNI, "api.vk.me"), DomainBase: body.DomainBase, DomainEmail: body.LEEmail,
			DomainCFProxy: body.CFProxy,
			WithLampac: body.WithLampac, WithGitRegistry: body.WithGit,
			GenerateKey: strings.TrimSpace(body.Key) == "",
			SSHPrivateKey: expandHome(body.Key), KeyPassphrase: body.KeyPassphrase,
			TelegramToken: body.TelegramToken, TelegramAdminID: body.TelegramAdminID,
			Version: deploy.Release,
		},
		Secondary: SecondarySpec{
			SecondaryHost: body.SecondaryHost, SecondaryUser: su, SecondaryPass: body.SecondaryPassword,
			SNI: orDefault(body.SNI, "api.vk.me"),
			PrimaryKeyPassphrase: body.KeyPassphrase,
		},
	}
	ApplyDomainFlags(&f.Primary)
	fl, okf := streamStart(w)
	rep := streamRep(w, fl, okf)
	if err := FleetDeployWithReport(f, rep); err != nil {
		_, _ = fmt.Fprintf(w, "ERROR: %v\n", err)
		return
	}
	_, _ = io.WriteString(w, "OK\n")
}

func handlePrimary(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if !requireToken(r, token) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var body primaryJSON
	if !decodeJSON(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Host) == "" {
		http.Error(w, "host required", 400)
		return
	}
	if body.WithTelegram || strings.TrimSpace(body.TelegramToken) != "" || strings.TrimSpace(body.TelegramAdminID) != "" {
		if strings.TrimSpace(body.TelegramToken) == "" || strings.TrimSpace(body.TelegramAdminID) == "" {
			http.Error(w, "telegram requires tg_token and tg_admin", 400)
			return
		}
	}
	if !tryLockDeploy(w) {
		return
	}
	defer fleetMu.Unlock()
	s := PrimarySpec{
		Host: body.Host, User: orDefault(body.User, "root"), Password: body.Password,
		SNI: orDefault(body.SNI, "api.vk.me"), DomainBase: body.DomainBase, DomainEmail: body.LEEmail,
		DomainCFProxy: body.CFProxy, WithLampac: body.WithLampac, WithGitRegistry: body.WithGit,
		GenerateKey: strings.TrimSpace(body.Key) == "", SSHPrivateKey: expandHome(body.Key),
		KeyPassphrase: body.KeyPassphrase, TelegramToken: body.TelegramToken, TelegramAdminID: body.TelegramAdminID,
		Version: deploy.Release,
	}
	ApplyDomainFlags(&s)
	fl, okf := streamStart(w)
	rep := streamRep(w, fl, okf)
	rep(Step{ID: "primary", Message: "deploy primary"})
	if err := DeployPrimary(s); err != nil {
		reportErr(rep, "primary", err)
		_, _ = fmt.Fprintf(w, "ERROR: %v\n", err)
		return
	}
	reportDone(rep, "primary", "primary ok")
	_, _ = io.WriteString(w, "OK\n")
}

func handleSecondary(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if !requireToken(r, token) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var body secondaryJSON
	if !decodeJSON(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.SecondaryHost) == "" || strings.TrimSpace(body.PrimaryHost) == "" {
		http.Error(w, "primary_host and secondary_host required", 400)
		return
	}
	key := expandHome(orDefault(body.PrimaryKey, "~/.ssh/netductor_primary"))
	if !tryLockDeploy(w) {
		return
	}
	defer fleetMu.Unlock()
	s := SecondarySpec{
		PrimaryHost: body.PrimaryHost, PrimaryUser: orDefault(body.PrimaryUser, "root"),
		PrimaryKey: key, PrimaryKeyPassphrase: body.PrimaryKeyPass,
		SecondaryHost: body.SecondaryHost, SecondaryUser: orDefault(body.SecondaryUser, "root"),
		SecondaryPass: body.SecondaryPassword, SNI: orDefault(body.SNI, "api.vk.me"),
	}
	fl, okf := streamStart(w)
	rep := streamRep(w, fl, okf)
	rep(Step{ID: "secondary", Message: "deploy secondary"})
	if err := DeploySecondary(s); err != nil {
		reportErr(rep, "secondary", err)
		_, _ = fmt.Fprintf(w, "ERROR: %v\n", err)
		return
	}
	reportDone(rep, "secondary", "secondary ok")
	_, _ = io.WriteString(w, "OK\n")
}

func handleCredentials(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if !requireToken(r, token) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var body credJSON
	if !decodeJSON(w, r, &body) {
		return
	}
	if strings.TrimSpace(body.Host) == "" {
		http.Error(w, "host required", 400)
		return
	}
	key := expandHome(orDefault(body.Key, "~/.ssh/netductor_primary"))
	path, err := CollectCredentials(orDefault(body.Role, "primary"), orDefault(body.User, "root"), body.Host, key, body.Pass)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(500)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"ok": "true", "path": path})
}


type edgeBody struct {
	RouterHost   string `json:"router_host"`
	RouterUser   string `json:"router_user"`
	RouterPass   string `json:"router_password"`
	DeviceID     string `json:"device_id"`
	PrimaryHost  string `json:"primary_host"`
	PrimaryUser  string `json:"primary_user"`
	PrimaryKey   string `json:"primary_key"`
	KeyPass      string `json:"key_passphrase"`
	ServerURL    string `json:"server_url"`
	AgentArch    string `json:"agent_arch"`
	NetConfigure bool   `json:"net_configure"`
	LANIP        string `json:"lan_ip"`
	WiFiSSID     string `json:"wifi_ssid"`
	WiFiKey      string `json:"wifi_key"`
	GuestEnable  bool   `json:"guest_enable"`
	WANProto     string `json:"wan_proto"`
}

func handleEdge(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if !requireToken(r, token) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var body edgeBody
	if !decodeJSON(w, r, &body) {
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("step edge start\n"))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	spec := EdgeSpec{
		RouterHost: body.RouterHost, RouterUser: body.RouterUser, RouterPass: body.RouterPass,
		DeviceID: body.DeviceID, PrimaryHost: body.PrimaryHost, PrimaryUser: body.PrimaryUser,
		PrimaryKey: body.PrimaryKey, PrimaryKeyPassphrase: body.KeyPass,
		ServerURL: body.ServerURL, AgentArch: body.AgentArch,
		NetConfigure: body.NetConfigure, LANIP: body.LANIP,
		WiFiSSID24: body.WiFiSSID, WiFiKey24: body.WiFiKey,
		GuestEnable: body.GuestEnable, WANProto: body.WANProto,
	}
	if spec.RouterUser == "" {
		spec.RouterUser = "root"
	}
	if spec.PrimaryUser == "" {
		spec.PrimaryUser = "root"
	}
	if err := DeployEdge(spec); err != nil {
		_, _ = w.Write([]byte("step edge ERROR " + err.Error() + "\n"))
		return
	}
	_, _ = w.Write([]byte("step edge done\n"))
}


func handleSite(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if !requireToken(r, token) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var body struct {
		SiteID   string `json:"site_id"`
		Name     string `json:"name"`
		RPiID    string `json:"rpi_id"`
		MikroTikID string `json:"mikrotik_id"`
		RPiLAN   string `json:"rpi_lan"`
		MTHost   string `json:"mt_host"`
		MTUser   string `json:"mt_user"`
		MTPass   string `json:"mt_password"`
		MTPort   string `json:"mt_port"`
		DoPush   bool   `json:"do_push"`
		KeyPath  string `json:"operator_key"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte("step site start\n"))
	out, err := DeploySite(SiteSpec{
		SiteID: body.SiteID, Name: body.Name, RPiID: body.RPiID, MikroTikID: body.MikroTikID,
		RPiLAN: body.RPiLAN, MTHost: body.MTHost, MTUser: body.MTUser, MTPass: body.MTPass,
		MTPort: ParsePort(body.MTPort, 22), DoPush: body.DoPush, OperatorKeyPath: body.KeyPath,
	})
	_, _ = w.Write([]byte(out))
	if err != nil {
		_, _ = w.Write([]byte("step site ERROR " + err.Error() + "\n"))
		return
	}
	_, _ = w.Write([]byte("step site done\n"))
}

func handleMikroTik(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	if !requireToken(r, token) {
		http.Error(w, "unauthorized", 401)
		return
	}
	var body struct {
		Host     string `json:"host"`
		User     string `json:"user"`
		Password string `json:"password"`
		Port     string `json:"port"`
		Action   string `json:"action"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	out, err := MikroTikAction(body.Host, body.User, body.Password, ParsePort(body.Port, 22), body.Action)
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(400)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "output": out})
}


func handleCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", 405)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
		"actions": opcatalog.ForSurface("web"),
		"by_section": opcatalog.BySection(),
	})
}
