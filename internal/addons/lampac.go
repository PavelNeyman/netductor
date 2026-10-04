package addons

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// LampacStatus is operator-facing view of the optional Lampac addon.
type LampacStatus struct {
	Installed   bool   `json:"installed"`
	Running     bool   `json:"running"`
	Healthy     bool   `json:"healthy"`
	Container   string `json:"container,omitempty"`
	Image       string `json:"image,omitempty"`
	Bind        string `json:"bind"` // e.g. 127.0.0.1:9118
	VersionHash string `json:"version_hash,omitempty"`
	PingOK      bool   `json:"ping_ok"`
	ChromiumOK  bool   `json:"chromium_ok"`
	CPU         string `json:"cpu,omitempty"`
	Mem         string `json:"mem,omitempty"`
	AdminURL    string `json:"admin_url"` // relative hint for UI
	UIURL       string `json:"ui_url"`
	Error       string `json:"error,omitempty"`
}

func lampacBase() string {
	port := os.Getenv("NETDUCTOR_LAMPAC_PORT")
	if port == "" {
		port = "9118"
	}
	bind := os.Getenv("NETDUCTOR_LAMPAC_BIND")
	if bind == "" {
		bind = "127.0.0.1"
	}
	return fmt.Sprintf("http://%s:%s", bind, port)
}

func httpGet(url string, timeout time.Duration) (int, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	return resp.StatusCode, strings.TrimSpace(string(b)), nil
}

// CollectLampac probes docker + local HTTP API (loopback only).
func CollectLampac() LampacStatus {
	st := LampacStatus{
		Bind:     "127.0.0.1:9118",
		AdminURL: "http://127.0.0.1:9118/admin",
		UIURL:    "http://127.0.0.1:9118/",
	}
	if v := os.Getenv("NETDUCTOR_LAMPAC_PORT"); v != "" {
		st.Bind = "127.0.0.1:" + v
		st.AdminURL = "http://127.0.0.1:" + v + "/admin"
		st.UIURL = "http://127.0.0.1:" + v + "/"
	}

	out, err := exec.Command("docker", "inspect",
		"--format", `{{json .}}`, "netductor-lampac").CombinedOutput()
	if err != nil {
		st.Error = "container not found"
		return st
	}
	st.Installed = true
	st.Container = "netductor-lampac"
	var ins struct {
		State struct {
			Running bool `json:"Running"`
			Health  struct {
				Status string `json:"Status"`
			} `json:"Health"`
		} `json:"State"`
		Config struct {
			Image string `json:"Image"`
		} `json:"Config"`
	}
	_ = json.Unmarshal(out, &ins)
	st.Running = ins.State.Running
	st.Healthy = strings.EqualFold(ins.State.Health.Status, "healthy") || (st.Running && ins.State.Health.Status == "")
	st.Image = ins.Config.Image

	if stats, err := exec.Command("docker", "stats", "--no-stream",
		"--format", "{{.CPUPerc}}\t{{.MemUsage}}", "netductor-lampac").Output(); err == nil {
		parts := strings.SplitN(strings.TrimSpace(string(stats)), "\t", 2)
		if len(parts) == 2 {
			st.CPU, st.Mem = parts[0], parts[1]
		}
	}

	base := lampacBase()
	if code, body, err := httpGet(base+"/version?type=hash", 2*time.Second); err == nil && code < 400 {
		st.VersionHash = body
		if len(st.VersionHash) > 64 {
			st.VersionHash = st.VersionHash[:64]
		}
	}
	if code, body, err := httpGet(base+"/ping", 2*time.Second); err == nil && code < 400 {
		st.PingOK = body == "pong" || code == 200
	}
	if code, body, err := httpGet(base+"/api/chromium/ping", 3*time.Second); err == nil && code < 400 {
		st.ChromiumOK = strings.Contains(strings.ToLower(body), "pong") || code == 200
	}
	return st
}

// ListAddons returns all optional components status.
func ListAddons() map[string]any {
	return map[string]any{
		"lampac": CollectLampac(),
		"versions": Versions(),
	}
}
