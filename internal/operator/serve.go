package operator

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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

	addr := net.JoinHostPort(bind, port)
	mux := http.NewServeMux()
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
		// inject token for same-origin fetches (loopback page only)
		html := strings.Replace(string(b), "/*__ND_TOKEN__*/", token, 1)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(html))
	})
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/v1/fleet", func(w http.ResponseWriter, r *http.Request) {
		handleFleet(w, r, token)
	})

	fmt.Fprintf(os.Stderr, "operator serve: http://%s/  (loopback only)\n", addr)
	fmt.Fprintf(os.Stderr, "operator token: %s  (header X-Netductor-Token)\n", token)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      2 * time.Hour, // deploy can be long
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

func tokenOK(want, got string) bool {
	if want == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(want), []byte(got)) == 1
}

type fleetJSON struct {
	DoPrimary         bool   `json:"do_primary"`
	DoSecondary       bool   `json:"do_secondary"`
	PrimaryHost       string `json:"primary_host"`
	PrimaryPassword   string `json:"primary_password"`
	SecondaryHost     string `json:"secondary_host"`
	SecondaryPassword string `json:"secondary_password"`
	DomainBase        string `json:"domain_base"`
	LEEmail           string `json:"le_email"`
	SNI               string `json:"sni"`
	Key               string `json:"key"`
	WithLampac        bool   `json:"with_lampac"`
	WithGit           bool   `json:"with_git"`
}

func handleFleet(w http.ResponseWriter, r *http.Request, token string) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	got := r.Header.Get("X-Netductor-Token")
	if got == "" {
		got = r.Header.Get("Authorization")
		got = strings.TrimPrefix(got, "Bearer ")
		got = strings.TrimPrefix(got, "bearer ")
	}
	if !tokenOK(token, strings.TrimSpace(got)) {
		http.Error(w, "unauthorized", 401)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
	var body fleetJSON
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		http.Error(w, err.Error(), 400)
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

	if !fleetMu.TryLock() {
		http.Error(w, "another deploy is running", 409)
		return
	}
	defer fleetMu.Unlock()

	f := FleetSpec{
		DoPrimary:   body.DoPrimary,
		DoSecondary: body.DoSecondary,
		Primary: PrimarySpec{
			Host: body.PrimaryHost, User: "root", Password: body.PrimaryPassword,
			SNI: orDefault(body.SNI, "api.vk.me"), DomainBase: body.DomainBase, DomainEmail: body.LEEmail,
			WithLampac: body.WithLampac, WithGitRegistry: body.WithGit,
			GenerateKey: strings.TrimSpace(body.Key) == "",
			SSHPrivateKey: expandHome(body.Key),
			Version:       deploy.Release,
		},
		Secondary: SecondarySpec{
			SecondaryHost: body.SecondaryHost, SecondaryUser: "root", SecondaryPass: body.SecondaryPassword,
			SNI: orDefault(body.SNI, "api.vk.me"),
		},
	}
	ApplyDomainFlags(&f.Primary)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	flusher, _ := w.(http.Flusher)
	rep := func(st Step) {
		line := fmt.Sprintf("step %s: %s\n", st.ID, st.Message)
		if st.Err != "" {
			line = fmt.Sprintf("step %s ERROR: %s\n", st.ID, st.Err)
		} else if st.Done {
			line = fmt.Sprintf("step %s done: %s\n", st.ID, st.Message)
		}
		_, _ = io.WriteString(w, line)
		if flusher != nil {
			flusher.Flush()
		}
	}
	if err := FleetDeployWithReport(f, rep); err != nil {
		_, _ = fmt.Fprintf(w, "ERROR: %v\n", err)
		return
	}
	_, _ = io.WriteString(w, "OK\n")
}
