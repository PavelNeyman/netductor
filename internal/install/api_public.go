package install

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	apiPublicStatePath = "/var/lib/netductor/api-public-arm.json"
	apiPublicComment   = "netductor-api-public-arm"
	apiPublicPort      = "8789"
)

type apiPublicState struct {
	Until   time.Time `json:"until"`
	ArmedAt time.Time `json:"armed_at"`
	TTL     string    `json:"ttl"`
}

func ufwQuiet(args ...string) error {
	cmd := exec.Command("ufw", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// deleteUFWByComment removes rules whose status line contains comment (high→low numbers).
func deleteUFWByComment(comment string) {
	out, err := exec.Command("ufw", "status", "numbered").CombinedOutput()
	if err != nil {
		return
	}
	var nums []string
	for _, ln := range strings.Split(string(out), "\n") {
		if !strings.Contains(ln, comment) {
			continue
		}
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, "[") {
			continue
		}
		end := strings.Index(ln, "]")
		if end > 1 {
			n := strings.TrimSpace(ln[1:end])
			nums = append([]string{n}, nums...)
		}
	}
	for _, n := range nums {
		cmd := exec.Command("bash", "-c", "echo y | ufw delete "+n)
		cmd.Stdout, cmd.Stderr = nil, nil
		_ = cmd.Run()
	}
}

// ArmAPIPublic opens :8789 from anywhere for ttl, then schedules disarm.
func ArmAPIPublic(ttl time.Duration) error {
	if ttl < time.Minute {
		ttl = time.Minute
	}
	if ttl > 2*time.Hour {
		ttl = 2 * time.Hour
	}
	if _, err := exec.LookPath("ufw"); err != nil {
		return fmt.Errorf("ufw not found")
	}
	// clear only previous temp public rules (do not re-apply full firewall)
	deleteUFWByComment(apiPublicComment)
	_ = exec.Command("systemctl", "stop", "nd-api-public-disarm.timer").Run()

	if err := ufwQuiet("allow", apiPublicPort+"/tcp", "comment", apiPublicComment); err != nil {
		return fmt.Errorf("ufw allow public %s: %w", apiPublicPort, err)
	}
	st := apiPublicState{
		ArmedAt: time.Now().UTC(),
		Until:   time.Now().UTC().Add(ttl),
		TTL:     ttl.String(),
	}
	_ = os.MkdirAll(filepath.Dir(apiPublicStatePath), 0o755)
	b, _ := json.MarshalIndent(st, "", "  ")
	if err := os.WriteFile(apiPublicStatePath, b, 0o600); err != nil {
		return err
	}
	sec := int(ttl.Seconds())
	if sec < 60 {
		sec = 60
	}
	unit := `[Unit]
Description=Disarm netductor API public after arm TTL
[Service]
Type=oneshot
ExecStart=/usr/local/bin/netductor api-public disarm
`
	timer := fmt.Sprintf(`[Unit]
Description=Timer disarm API public
[Timer]
OnActiveSec=%ds
AccuracySec=5s
Unit=nd-api-public-disarm.service
[Install]
WantedBy=timers.target
`, sec)
	_ = os.WriteFile("/run/systemd/system/nd-api-public-disarm.service", []byte(unit), 0o644)
	_ = os.WriteFile("/run/systemd/system/nd-api-public-disarm.timer", []byte(timer), 0o644)
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "start", "nd-api-public-disarm.timer").Run()
	return nil
}

// DisarmAPIPublic removes temporary public allow; keeps service-plane rules intact.
func DisarmAPIPublic() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		_ = os.Remove(apiPublicStatePath)
		return nil
	}
	deleteUFWByComment(apiPublicComment)
	_ = os.Remove(apiPublicStatePath)
	_ = exec.Command("systemctl", "stop", "nd-api-public-disarm.timer").Run()
	// ensure restricted policy present (quiet)
	_ = applyAgentFirewallQuiet()
	return nil
}

// APIPublicStatus returns armed flag and until time.
func APIPublicStatus() (armed bool, until time.Time) {
	b, err := os.ReadFile(apiPublicStatePath)
	if err != nil {
		return false, time.Time{}
	}
	var st apiPublicState
	if json.Unmarshal(b, &st) != nil {
		return false, time.Time{}
	}
	if time.Now().UTC().After(st.Until) {
		_ = DisarmAPIPublic()
		return false, time.Time{}
	}
	return true, st.Until
}
