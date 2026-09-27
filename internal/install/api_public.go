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

// ArmAPIPublic opens :8789 from anywhere for ttl, then schedules disarm via systemd-run.
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
	// remove previous temp allow if any
	_ = DisarmAPIPublic()

	if err := run("ufw", "allow", apiPublicPort+"/tcp", "comment", apiPublicComment); err != nil {
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
	// schedule disarm
	sec := int(ttl.Seconds())
	if sec < 60 {
		sec = 60
	}
	// systemd-run one-shot
	_ = exec.Command("systemctl", "reset-failed", "nd-api-public-disarm.service").Run()
	_ = exec.Command("systemctl", "stop", "nd-api-public-disarm.timer").Run()
	unit := fmt.Sprintf(`[Unit]
Description=Disarm netductor API public after arm TTL
[Service]
Type=oneshot
ExecStart=/usr/local/bin/netductor api-public disarm
`)
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
	fmt.Fprintf(os.Stderr, "api-public: :%s OPEN until %s (TTL %s)\n", apiPublicPort, st.Until.Format(time.RFC3339), ttl)
	return nil
}

// DisarmAPIPublic removes temporary public allow and re-applies restricted rules.
func DisarmAPIPublic() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		_ = os.Remove(apiPublicStatePath)
		return nil
	}
	// delete rules with our comment (best-effort)
	out, _ := exec.Command("ufw", "status", "numbered").CombinedOutput()
	lines := strings.Split(string(out), "\n")
	// delete from highest number to lowest
	var nums []string
	for _, ln := range lines {
		if !strings.Contains(ln, apiPublicComment) && !strings.Contains(ln, "netductor-api-public") {
			continue
		}
		ln = strings.TrimSpace(ln)
		if !strings.HasPrefix(ln, "[") {
			continue
		}
		end := strings.Index(ln, "]")
		if end > 1 {
			nums = append([]string{strings.TrimSpace(ln[1:end])}, nums...)
		}
	}
	for _, n := range nums {
		_ = exec.Command("bash", "-c", fmt.Sprintf("echo y | ufw delete %s", n)).Run()
	}
	_ = run("ufw", "delete", "allow", apiPublicPort+"/tcp")
	_ = os.Remove(apiPublicStatePath)
	_ = exec.Command("systemctl", "stop", "nd-api-public-disarm.timer").Run()
	// restore restricted policy
	_ = ApplyAgentFirewall()
	fmt.Fprintln(os.Stderr, "api-public: disarmed, :8789 restricted again")
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
