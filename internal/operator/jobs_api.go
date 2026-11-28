package operator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/PavelNeyman/netductor/internal/deploy"
	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/jobs"
	"github.com/PavelNeyman/netductor/internal/vpscheck"
)

var (
	opJobsOnce sync.Once
	opJobs     *jobs.Store
)

// OpJobStore returns the process-wide operator job store (serve-owned).
func OpJobStore() *jobs.Store {
	opJobsOnce.Do(func() {
		home, _ := os.UserHomeDir()
		dir := filepath.Join(home, ".cache", "netductor", "jobs")
		opJobs = jobs.NewStore(dir)
		registerOpRunners(opJobs)
	})
	return opJobs
}

func registerOpRunners(s *jobs.Store) {
	s.Register("edge.probe", runEdgeProbe)
	s.Register("edge.bootstrap", runEdgeBootstrap)
	s.Register("edge.provision", runEdgeProvision)
	s.Register("edge.offline_prep", runEdgeOfflinePrep)
	s.Register("fleet.deploy", runFleetDeploy)
	s.Register("primary.deploy", runPrimaryDeploy)
	s.Register("secondary.deploy", runSecondaryDeploy)
	s.Register("buildhost.deploy", runBuildHostDeploy)
	s.Register("vps.precheck", runVpsPrecheck)
}

func registerJobsAPI(mux *http.ServeMux, token string) {
	_ = OpJobStore() // init
	mux.HandleFunc("/v1/jobs", func(w http.ResponseWriter, r *http.Request) {
		if !checkToken(r, token) {
			http.Error(w, "unauthorized", 401)
			return
		}
		switch r.Method {
		case http.MethodGet:
			st := jobs.Status(r.URL.Query().Get("status"))
			typ := r.URL.Query().Get("type")
			writeJSONOp(w, 200, map[string]any{"jobs": OpJobStore().List(st, typ)})
		case http.MethodPost:
			var body struct {
				Type   string         `json:"type"`
				Params map[string]any `json:"params"`
			}
			if !decodeJSON(w, r, &body) {
				return
			}
			if body.Type == "" {
				writeJSONOp(w, 400, map[string]string{"error": "type required"})
				return
			}
			j, err := OpJobStore().Start(body.Type, body.Params)
			if err != nil {
				writeJSONOp(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSONOp(w, 202, j.Snapshot())
		default:
			http.Error(w, "GET or POST", 405)
		}
	})
	mux.HandleFunc("/v1/jobs/", func(w http.ResponseWriter, r *http.Request) {
		if !checkToken(r, token) {
			http.Error(w, "unauthorized", 401)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/v1/jobs/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			http.NotFound(w, r)
			return
		}
		id := parts[0]
		if len(parts) == 2 && parts[1] == "cancel" && r.Method == http.MethodPost {
			if err := OpJobStore().Cancel(id); err != nil {
				writeJSONOp(w, 400, map[string]string{"error": err.Error()})
				return
			}
			writeJSONOp(w, 200, map[string]any{"ok": true, "id": id})
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "GET or POST cancel", 405)
			return
		}
		j := OpJobStore().Get(id)
		if j == nil {
			writeJSONOp(w, 404, map[string]string{"error": "not found"})
			return
		}
		writeJSONOp(w, 200, j.Snapshot())
	})
}

func checkToken(r *http.Request, token string) bool {
	if token == "" {
		return true
	}
	got := r.Header.Get("X-Netductor-Token")
	if got == "" {
		got = r.URL.Query().Get("token")
	}
	return subtleConstEq(got, token)
}

func subtleConstEq(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func strParam(p map[string]any, keys ...string) string {
	if p == nil {
		return ""
	}
	for _, k := range keys {
		if v, ok := p[k].(string); ok {
			if s := strings.TrimSpace(v); s != "" {
				return s
			}
		}
	}
	return ""
}

func boolParam(p map[string]any, k string, def bool) bool {
	if p == nil {
		return def
	}
	v, ok := p[k]
	if !ok {
		return def
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "1" || strings.EqualFold(t, "true") || t == "yes"
	default:
		return def
	}
}

func runEdgeProbe(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	p := j.Params
	host := strParam(p, "router_host")
	user := strParam(p, "router_user")
	if user == "" {
		user = "root"
	}
	pass := strParam(p, "router_pass")
	if pass == "" {
		pass = strParam(p, "router_password")
	}
	key := expandHome(strParam(p, "primary_key"))
	keyPass := strParam(p, "key_pass")
	if keyPass == "" {
		keyPass = strParam(p, "key_passphrase")
	}
	if host == "" {
		return nil, fmt.Errorf("router_host required")
	}
	setStep(jobs.Step{ID: "probe", Message: "SSH probe (read-only)"})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	facts, raw, err := deploy.ProbeDeviceFacts(pass, key, user, host, keyPass, false)
	appendLog(raw)
	if err != nil {
		setStep(jobs.Step{ID: "probe", Message: err.Error(), Err: err.Error()})
		return nil, err
	}
	setStep(jobs.Step{ID: "probe", Message: "facts ok", Done: true})

	// Plan (same as handleEdgePreview)
	ssid := strParam(p, "wifi_ssid")
	if ssid == "" {
		ssid = strParam(p, "wifi_ssid_24")
	}
	vpnOn := boolParam(p, "vpn_enable", true)
	mods, _ := p["modules"].(map[string]any)
	enabled := map[string]bool{}
	for k, v := range mods {
		if b, ok := v.(bool); ok {
			enabled[k] = b
		}
	}
	req := edge.PlanRequest{
		Preset: strParam(p, "preset"),
		Facts:  &facts,
		Intent: edge.DeployIntent{
			ConfigureNet: boolParam(p, "net_configure", true),
			GuestEnable:  boolParam(p, "guest_enable", false),
			VPNEnable:    vpnOn,
			WiFiSSID:     ssid,
			ExpandFS:     enabled[edge.ModFSExpand],
			OverlayExt:   enabled[edge.ModOverlay],
		},
		Selection: edge.ModuleSelection{Advanced: boolParam(p, "advanced", false), Enabled: enabled},
	}
	resp, err := edge.ResolvePlan(req)
	if err != nil {
		return nil, err
	}
	setStep(jobs.Step{ID: "plan", Message: "plan resolved", Done: true})
	installed, _ := deploy.ProbeInstalledPackages(pass, key, user, host, keyPass)
	presets := deploy.EvaluateBootstrapPresets(installed, bootstrapOptsFromFacts(facts))
	return map[string]any{"facts": facts, "plan": resp, "bootstrap_presets": presets}, nil
}

func runEdgeBootstrap(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	p := j.Params
	host := strParam(p, "router_host")
	user := strParam(p, "router_user")
	if user == "" {
		user = "root"
	}
	pass := strParam(p, "router_pass")
	if pass == "" {
		pass = strParam(p, "router_password")
	}
	key := expandHome(strParam(p, "primary_key"))
	keyPass := strParam(p, "key_pass")
	if keyPass == "" {
		keyPass = strParam(p, "key_passphrase")
	}
	preset := strParam(p, "preset")
	if preset == "" {
		preset = "usb-storage"
	}
	if host == "" {
		return nil, fmt.Errorf("router_host required")
	}
	setStep(jobs.Step{ID: "bootstrap", Message: "install preset " + preset})
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	done := make(chan error, 1)
	go func() {
		done <- deploy.WithSSHContext(ctx, func() error {
			return deploy.BootstrapPresetOnRouter(preset, pass, key, user, host, keyPass)
		})
	}()
	select {
	case <-ctx.Done():
		setStep(jobs.Step{ID: "bootstrap", Message: "cancel requested", Err: "cancelled"})
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			setStep(jobs.Step{ID: "bootstrap", Message: err.Error(), Err: err.Error()})
			return nil, err
		}
	}
	setStep(jobs.Step{ID: "bootstrap", Message: "packages installed", Done: true})
	setStep(jobs.Step{ID: "reprobe", Message: "re-probe after bootstrap"})
	facts, raw, err := deploy.ProbeDeviceFacts(pass, key, user, host, keyPass, false)
	appendLog(raw)
	if err != nil {
		setStep(jobs.Step{ID: "reprobe", Message: err.Error(), Err: err.Error()})
		return nil, err
	}
	installed, _ := deploy.ProbeInstalledPackages(pass, key, user, host, keyPass)
	presets := deploy.EvaluateBootstrapPresets(installed, bootstrapOptsFromFacts(facts))
	setStep(jobs.Step{ID: "reprobe", Message: "facts updated", Done: true})
	return map[string]any{"facts": facts, "bootstrap_presets": presets, "preset": preset}, nil
}

func runEdgeProvision(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	setStep(jobs.Step{ID: "provision", Message: "starting edge deploy"})
	b, _ := json.Marshal(j.Params)
	var s EdgeSpec
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("edge params: %w", err)
	}
	// map common snake_case from web / TUI (json.Unmarshal misses untagged EdgeSpec fields)
	if s.RouterHost == "" {
		s.RouterHost = strParam(j.Params, "router_host")
	}
	if s.RouterUser == "" {
		s.RouterUser = strParam(j.Params, "router_user")
	}
	if s.RouterPass == "" {
		s.RouterPass = strParam(j.Params, "router_pass", "router_password", "password")
	}
	if s.PrimaryKey == "" {
		s.PrimaryKey = strParam(j.Params, "primary_key", "PrimaryKey")
	}
	if s.PrimaryKeyPassphrase == "" {
		s.PrimaryKeyPassphrase = strParam(j.Params, "key_pass", "key_passphrase", "primary_key_passphrase")
	}
	if s.DeviceID == "" {
		s.DeviceID = strParam(j.Params, "device_id", "DeviceID")
	}
	if s.NewRootPassword == "" {
		s.NewRootPassword = strParam(j.Params, "new_root_password", "NewRootPassword")
	}
	if !s.SkipRootPass {
		s.SkipRootPass = truthyParam(j.Params, "skip_root_pass", "SkipRootPass")
	}
	if s.ServerURL == "" {
		s.ServerURL = strParam(j.Params, "server_url", "server", "ServerURL")
	}
	if s.PrimaryHost == "" {
		s.PrimaryHost = strParam(j.Params, "primary_host", "PrimaryHost", "control_host")
	}
	if s.Version == "" {
		s.Version = strParam(j.Params, "version", "Version")
	}
	if s.AgentArch == "" {
		s.AgentArch = strParam(j.Params, "agent_arch", "AgentArch", "arch")
	}
	if s.BootstrapToken == "" {
		s.BootstrapToken = strParam(j.Params, "bootstrap_token", "BootstrapToken", "token")
	}
	if !s.NetConfigure {
		s.NetConfigure = truthyParam(j.Params, "net_configure", "NetConfigure", "net")
	}
	if !s.Reboot {
		s.Reboot = truthyParam(j.Params, "reboot", "Reboot")
	}
	if s.MTLSCAFile == "" {
		s.MTLSCAFile = strParam(j.Params, "mtls_ca", "mtls_ca_file", "MTLSCAFile")
	}
	if s.MTLSCertFile == "" {
		s.MTLSCertFile = strParam(j.Params, "mtls_cert", "mtls_cert_file", "MTLSCertFile")
	}
	if s.MTLSKeyFile == "" {
		s.MTLSKeyFile = strParam(j.Params, "mtls_key", "mtls_key_file", "MTLSKeyFile")
	}

	// Web wizard sends modules map + snake_case WiFi/LAN — EdgeSpec has no json tags.
	if mods, ok := j.Params["modules"].(map[string]any); ok && len(mods) > 0 {
		if s.Selection.Enabled == nil {
			s.Selection.Enabled = map[string]bool{}
		}
		for k, v := range mods {
			switch x := v.(type) {
			case bool:
				s.Selection.Enabled[k] = x
			case string:
				s.Selection.Enabled[k] = x == "1" || strings.EqualFold(x, "true") || strings.EqualFold(x, "yes") || strings.EqualFold(x, "on")
			}
		}
		if s.Selection.Enabled[edge.ModOverlay] {
			s.OverlayExt = true
		}
		if s.Selection.Enabled[edge.ModFSExpand] {
			s.ExpandFS = true
		}
		if s.Selection.Enabled[edge.ModGuest] || s.Selection.Enabled["guest"] {
			s.GuestEnable = true
		}
	}
	if s.WiFiSSID24 == "" {
		s.WiFiSSID24 = strParam(j.Params, "wifi_ssid_24", "wifi_ssid", "ssid_24")
	}
	if s.WiFiKey24 == "" {
		s.WiFiKey24 = strParam(j.Params, "wifi_key_24", "wifi_key", "key_24")
	}
	if s.WiFiSSID5 == "" {
		s.WiFiSSID5 = strParam(j.Params, "wifi_ssid_5", "ssid_5")
	}
	if s.WiFiKey5 == "" {
		s.WiFiKey5 = strParam(j.Params, "wifi_key_5", "key_5")
	}
	if s.WiFiSSID == "" {
		s.WiFiSSID = firstNonEmptyStr(s.WiFiSSID24, strParam(j.Params, "wifi_ssid"))
	}
	if s.WiFiKey == "" {
		s.WiFiKey = firstNonEmptyStr(s.WiFiKey24, strParam(j.Params, "wifi_key"))
	}
	if s.LANIP == "" {
		s.LANIP = strParam(j.Params, "lan_ip", "LANIP")
	}
	if s.LANMask == "" {
		s.LANMask = strParam(j.Params, "lan_mask", "LANMask")
	}
	if s.DHCPStart == "" {
		s.DHCPStart = strParam(j.Params, "dhcp_start")
	}
	if s.DHCPLimit == "" {
		s.DHCPLimit = strParam(j.Params, "dhcp_limit")
	}
	if s.WANProto == "" {
		s.WANProto = strParam(j.Params, "wan_proto")
	}
	if s.WANIP == "" {
		s.WANIP = strParam(j.Params, "wan_ip")
	}
	if s.WANMask == "" {
		s.WANMask = strParam(j.Params, "wan_mask")
	}
	if s.WANGateway == "" {
		s.WANGateway = strParam(j.Params, "wan_gateway")
	}
	if s.WANDNS == "" {
		s.WANDNS = strParam(j.Params, "wan_dns")
	}
	if s.GuestSSID == "" {
		s.GuestSSID = strParam(j.Params, "guest_ssid")
	}
	if s.GuestPIN == "" {
		s.GuestPIN = strParam(j.Params, "guest_pin")
	}
	if s.GuestPSK == "" {
		s.GuestPSK = strParam(j.Params, "guest_psk")
	}
	if s.Preset == "" {
		s.Preset = strParam(j.Params, "preset")
	}
	if truthyParam(j.Params, "guest_enable", "GuestEnable") {
		s.GuestEnable = true
	}
	// Default visible SSID (GuestVisible=true). Only hide when guest_hidden is explicit yes/1/true.
	if s.GuestEnable {
		s.GuestVisible = guestSSIDVisible(strParam(j.Params, "guest_hidden", "GuestHidden"))
	}
	if truthyParam(j.Params, "overlay_ext", "OverlayExt") {
		s.OverlayExt = true
		if s.Selection.Enabled == nil {
			s.Selection.Enabled = map[string]bool{}
		}
		s.Selection.Enabled[edge.ModOverlay] = true
	}
	if truthyParam(j.Params, "vpn_enable") || (s.Selection.Enabled != nil && s.Selection.Enabled[edge.ModVPNClient]) {
		if s.Selection.Enabled == nil {
			s.Selection.Enabled = map[string]bool{}
		}
		s.Selection.Enabled[edge.ModVPNClient] = true
	}

	// Offline pack defaults when UI did not pass token/mTLS paths.
	if s.BootstrapToken == "" || s.MTLSCAFile == "" || s.MTLSCertFile == "" {
		off := deploy.OfflineEdgeDir()
		if s.BootstrapToken == "" {
			if b, err := os.ReadFile(filepath.Join(off, "edge_bootstrap_token")); err == nil {
				s.BootstrapToken = strings.TrimSpace(string(b))
			}
		}
		if s.MTLSCAFile == "" {
			if p := filepath.Join(off, "mtls-ca.crt"); fileExistsLocal(p) {
				s.MTLSCAFile = p
			}
		}
		if s.MTLSCertFile == "" {
			if s.DeviceID != "" {
				p := filepath.Join(off, "clients", s.DeviceID, "client.crt")
				if fileExistsLocal(p) {
					s.MTLSCertFile = p
					s.MTLSKeyFile = filepath.Join(off, "clients", s.DeviceID, "client.key")
				}
			}
			if s.MTLSCertFile == "" {
				p := filepath.Join(off, "mtls-client.crt")
				if fileExistsLocal(p) {
					s.MTLSCertFile = p
					s.MTLSKeyFile = filepath.Join(off, "mtls-client.key")
				}
			}
		}
	}
	s.Log = func(line string) {
		line = strings.TrimRight(line, "\r\n")
		if line != "" {
			appendLog(line)
		}
	}
	done := make(chan error, 1)
	go func() { done <- deploy.WithSSHContext(ctx, func() error { return DeployEdge(s) }) }()
	select {
	case <-ctx.Done():
		setStep(jobs.Step{ID: "provision", Message: "cancel requested", Err: "cancelled"})
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			setStep(jobs.Step{ID: "provision", Message: err.Error(), Err: err.Error()})
			return nil, err
		}
		setStep(jobs.Step{ID: "provision", Message: "done", Done: true})
		return map[string]any{"ok": true}, nil
	}
}

func runEdgeOfflinePrep(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	p := j.Params
	setStep(jobs.Step{ID: "offline-prep", Message: "caching agents + OpenWrt USB packages"})
	host := strParam(p, "primary_host")
	user := strParam(p, "primary_user")
	if user == "" {
		user = "root"
	}
	key := expandHome(strParam(p, "primary_key"))
	keyPass := strParam(p, "key_pass")
	deviceID := strParam(p, "device_id")
	version := strParam(p, "version")
	force := boolParam(p, "force_refresh", false)
	m, err := deploy.PrepareEdgeOfflineOpts(host, user, key, keyPass, deviceID, version, force)
	if err != nil {
		return nil, err
	}
	appendLog(fmt.Sprintf("version=%s agents=%d partial=%v", m.Version, len(m.Agents), m.Partial))
	setStep(jobs.Step{ID: "offline-prep", Message: m.Status, Done: true})
	return m, nil
}

func runFleetDeploy(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	f, err := NormalizeFleetParams(j.Params)
	if err != nil {
		return nil, err
	}
	rep := func(st Step) {
		setStep(jobs.Step{ID: st.ID, Message: st.Message, Done: st.Done, Err: st.Err})
		appendLog(st.ID + ": " + st.Message)
	}
	done := make(chan error, 1)
	go func() { done <- deploy.WithSSHContext(ctx, func() error { return FleetDeployWithReport(f, rep) }) }()
	select {
	case <-ctx.Done():
		setStep(jobs.Step{ID: "fleet", Message: "cancel requested", Err: "cancelled"})
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	}
}

func runPrimaryDeploy(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	b, _ := json.Marshal(j.Params)
	var s PrimarySpec
	_ = json.Unmarshal(b, &s)
	// Web flat snake_case (overrides empty fields from untyped decode).
	flat := PrimaryFromFlatParams(j.Params)
	mergePrimary(&s, flat)
	if s.Host == "" {
		return nil, fmt.Errorf("primary host required")
	}
	s.SSHPrivateKey = expandHome(s.SSHPrivateKey)
	setStep(jobs.Step{ID: "primary", Message: "deploy primary on " + s.Host})
	ApplyDomainFlags(&s)
	done := make(chan error, 1)
	go func() { done <- deploy.WithSSHContext(ctx, func() error { return DeployPrimary(s) }) }()
	select {
	case <-ctx.Done():
		setStep(jobs.Step{ID: "primary", Message: "cancel requested", Err: "cancelled"})
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			return nil, err
		}
		setStep(jobs.Step{ID: "primary", Message: "ok", Done: true})
		return map[string]any{"ok": true}, nil
	}
}

func runSecondaryDeploy(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	b, _ := json.Marshal(j.Params)
	var s SecondarySpec
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	// Accept alternate keys from older UI / TUI field names.
	if s.SecondaryHost == "" {
		s.SecondaryHost = strParam(j.Params, "secondary_host", "SecondaryHost", "host", "sec_host")
	}
	if s.PrimaryHost == "" {
		s.PrimaryHost = strParam(j.Params, "primary_host", "PrimaryHost")
	}
	if s.PrimaryKey == "" {
		s.PrimaryKey = strParam(j.Params, "primary_key", "PrimaryKey", "key")
	}
	if s.PrimaryKeyPassphrase == "" {
		s.PrimaryKeyPassphrase = strParam(j.Params, "primary_key_passphrase", "key_passphrase", "key_pass")
	}
	if s.SNI == "" {
		s.SNI = strParam(j.Params, "sni", "SNI")
	}
	if s.Region == "" {
		s.Region = strParam(j.Params, "region", "Region")
	}
	if s.SecondaryPass == "" {
		s.SecondaryPass = strParam(j.Params, "secondary_password", "SecondaryPass", "password", "sec_password")
	}
	if s.SecondaryUser == "" {
		s.SecondaryUser = strParam(j.Params, "secondary_user", "user")
	}
	if s.SecondaryUser == "" {
		s.SecondaryUser = "root"
	}
	s.PrimaryKey = expandHome(s.PrimaryKey)
	s.SecondarySSHKey = expandHome(s.SecondarySSHKey)
	if port, ok := portParam(j.Params, "primary_ssh_port", "PrimarySSHPort"); ok {
		n, err := ParseSSHPort(port)
		if err != nil {
			return nil, err
		}
		s.PrimarySSHPort = n
	}
	setStep(jobs.Step{ID: "secondary", Message: "deploy secondary on " + s.SecondaryHost})
	done := make(chan error, 1)
	go func() { done <- deploy.WithSSHContext(ctx, func() error { return DeploySecondary(s) }) }()
	select {
	case <-ctx.Done():
		setStep(jobs.Step{ID: "secondary", Message: "cancel requested", Err: "cancelled"})
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			return nil, err
		}
		setStep(jobs.Step{ID: "secondary", Message: "ok", Done: true})
		return map[string]any{"ok": true}, nil
	}
}

func runBuildHostDeploy(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	b, _ := json.Marshal(j.Params)
	var s BuildHostSpec
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, err
	}
	setStep(jobs.Step{ID: "build-host", Message: "deploy build-host"})
	done := make(chan error, 1)
	go func() { done <- deploy.WithSSHContext(ctx, func() error { return DeployBuildHost(s) }) }()
	select {
	case <-ctx.Done():
		setStep(jobs.Step{ID: "build-host", Message: "cancel requested", Err: "cancelled"})
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			return nil, err
		}
		setStep(jobs.Step{ID: "build-host", Message: "ok", Done: true})
		return map[string]any{"ok": true}, nil
	}
}

func bootstrapOptsFromFacts(facts edge.DeviceFacts) deploy.BootstrapEvalOpts {
	var usbKB int64
	for _, d := range facts.Storage.USBDisks {
		if d.SizeKB > usbKB {
			usbKB = d.SizeKB
		}
	}
	return deploy.BootstrapEvalOpts{
		USBVisible:    facts.Storage.HasUsableUSB(),
		OverlayFreeKB: facts.Storage.OverlayFreeKB,
		USBSizeKB:     usbKB,
	}
}

func runVpsPrecheck(ctx context.Context, j *jobs.Job, appendLog func(string), setStep func(jobs.Step)) (any, error) {
	p := j.Params
	host := strParam(p, "host")
	if host == "" {
		host = strParam(p, "router_host")
	}
	user := strParam(p, "user")
	if user == "" {
		user = strParam(p, "router_user")
	}
	if user == "" {
		user = "root"
	}
	pass := strParam(p, "password")
	if pass == "" {
		pass = strParam(p, "router_pass")
	}
	key := expandHome(strParam(p, "key"))
	if key == "" {
		key = expandHome(strParam(p, "primary_key"))
	}
	keyPass := strParam(p, "key_passphrase")
	if keyPass == "" {
		keyPass = strParam(p, "key_pass")
	}
	port := strParam(p, "port")
	if port == "" {
		port = "22"
	}
	profile := strParam(p, "profile")
	if profile == "" {
		profile = "secondary"
	}
	primary := strParam(p, "primary")
	skip := boolParam(p, "skip_install", false)
	if host == "" {
		return nil, fmt.Errorf("host required")
	}
	setStep(jobs.Step{ID: "vpscheck", Message: "running pre-check on " + host})
	opt := vpscheck.SSHOpts{
		Host: host, User: user, Password: pass, KeyPath: key, KeyPassphrase: keyPass,
		Port: port, Profile: vpscheck.ParseProfile(profile), PrimaryIP: primary, SkipInstall: skip,
	}
	var rep vpscheck.Report
	var err error
	done := make(chan error, 1)
	go func() {
		rep, err = vpscheck.RunViaSSH(ctx, opt)
		done <- err
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-done:
		if err != nil {
			setStep(jobs.Step{ID: "vpscheck", Message: err.Error(), Err: err.Error()})
			return nil, err
		}
	}
	appendLog(rep.Text())
	setStep(jobs.Step{ID: "vpscheck", Message: rep.SummaryEN, Done: true})
	return rep, nil
}

func truthyParam(p map[string]any, keys ...string) bool {
	for _, k := range keys {
		v, ok := p[k]
		if !ok || v == nil {
			continue
		}
		switch x := v.(type) {
		case bool:
			if x {
				return true
			}
		case string:
			s := strings.ToLower(strings.TrimSpace(x))
			if s == "1" || s == "true" || s == "yes" || s == "on" {
				return true
			}
		}
	}
	return false
}

func fileExistsLocal(p string) bool {
	if p == "" {
		return false
	}
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func firstNonEmptyStr(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}
