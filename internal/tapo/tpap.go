package tapo

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloudPassword is the TP-Link account password used for TPAP/SPAKE2+ (V4)
// on newer firmware (e.g. C200 5.0 / 1.4.6). Distinct from Camera Account (RTSP).
// When empty, Login still tries classic/KLAP only.

func tpapScript() string {
	if v := os.Getenv("NETDUCTOR_TAPO_V4"); v != "" {
		return v
	}
	// relative to binary or install paths
	candidates := []string{
		"/opt/netductor/scripts/tapo_v4/cli.py",
		"/usr/local/share/netductor/tapo_v4/cli.py",
		"scripts/tapo_v4/cli.py",
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append([]string{filepath.Join(filepath.Dir(exe), "scripts", "tapo_v4", "cli.py")}, candidates...)
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return "scripts/tapo_v4/cli.py"
}

func (c *Client) cloudPass() string {
	if c.CloudPassword != "" {
		return c.CloudPassword
	}
	return c.Password // last resort: sometimes same
}

// loginTPAP uses freeKC V4 helper (SPAKE2+). Requires python3 + pycryptodome.
func (c *Client) loginTPAP() error {
	script := tpapScript()
	out, err := exec.Command("python3", script, "login", c.Host, c.cloudPass()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("tpap login: %v: %s", err, strings.TrimSpace(string(out)))
	}
	var res struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Stok  string `json:"stok"`
	}
	if e := json.Unmarshal(out, &res); e != nil {
		return fmt.Errorf("tpap login parse: %w (%s)", e, string(out))
	}
	if !res.OK {
		return fmt.Errorf("tpap login: %s", res.Error)
	}
	c.stok = res.Stok
	c.tpap = true
	return nil
}

// executeTPAP runs one method via V4 multipleRequest wrapper.
func (c *Client) executeTPAP(method string, params map[string]any) (map[string]any, error) {
	script := tpapScript()
	pj, _ := json.Marshal(params)
	if params == nil {
		pj = []byte("{}")
	}
	out, err := exec.Command("python3", script, "exec", c.Host, c.cloudPass(), method, string(pj)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tpap exec: %v: %s", err, strings.TrimSpace(string(out)))
	}
	var res struct {
		OK        bool             `json:"ok"`
		Error     string           `json:"error"`
		Responses []map[string]any `json:"responses"`
	}
	if e := json.Unmarshal(out, &res); e != nil {
		return nil, fmt.Errorf("tpap exec parse: %w", e)
	}
	if !res.OK {
		return nil, fmt.Errorf("tpap exec: %s", res.Error)
	}
	if len(res.Responses) == 0 {
		return map[string]any{}, nil
	}
	return res.Responses[0], nil
}
