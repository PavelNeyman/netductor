package operator

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/operator/web"
)

// ServeOpts for localhost operator HTTP API + embed UI.
type ServeOpts struct {
	Bind string // must be 127.0.0.1 or localhost
	Port string
}

// Serve runs a blocking HTTP server (loopback only).
func Serve(o ServeOpts) error {
	bind := strings.TrimSpace(o.Bind)
	if bind == "" {
		bind = "127.0.0.1"
	}
	if bind != "127.0.0.1" && bind != "localhost" && bind != "::1" {
		return fmt.Errorf("operator serve: bind must be loopback (got %q)", bind)
	}
	port := strings.TrimSpace(o.Port)
	if port == "" {
		port = "7373"
	}
	addr := net.JoinHostPort(bind, port)
	mux := http.NewServeMux()
	sub, err := fs.Sub(web.FS, ".")
	if err != nil {
		return err
	}
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.HandleFunc("/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/v1/fleet", handleFleet)
	fmt.Fprintf(os.Stderr, "operator serve: http://%s/  (loopback only)\n", addr)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	return srv.ListenAndServe()
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
}

func handleFleet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", 405)
		return
	}
	var body fleetJSON
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	f := FleetSpec{
		DoPrimary:   body.DoPrimary,
		DoSecondary: body.DoSecondary,
		Primary: PrimarySpec{
			Host: body.PrimaryHost, User: "root", Password: body.PrimaryPassword,
			SNI: orDefault(body.SNI, "api.vk.me"), DomainBase: body.DomainBase, DomainEmail: body.LEEmail,
			WithLampac: body.WithLampac, GenerateKey: body.Key == "",
			SSHPrivateKey: expandHome(body.Key),
		},
		Secondary: SecondarySpec{
			SecondaryHost: body.SecondaryHost, SecondaryUser: "root", SecondaryPass: body.SecondaryPassword,
			SNI: orDefault(body.SNI, "api.vk.me"),
		},
	}
	ApplyDomainFlags(&f.Primary)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	flusher, _ := w.(http.Flusher)
	rep := func(st Step) {
		line := fmt.Sprintf("step %s: %s\n", st.ID, st.Message)
		if st.Err != "" {
			line = fmt.Sprintf("step %s ERROR: %s\n", st.ID, st.Err)
		} else if st.Done {
			line = fmt.Sprintf("step %s done: %s\n", st.ID, st.Message)
		}
		_, _ = fmt.Fprint(w, line)
		if flusher != nil {
			flusher.Flush()
		}
	}
	if err := FleetDeployWithReport(f, rep); err != nil {
		_, _ = fmt.Fprintf(w, "ERROR: %v\n", err)
		return
	}
	_, _ = fmt.Fprint(w, "OK\n")
}
