package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

const (
	apiPublicStatePath = "/var/lib/netductor/api-public-arm.json"
	apiPublicComment   = "netductor-api-public-arm"
	apiPublicPort      = "8789"
	apiPublicUnitDir   = "/etc/systemd/system"
	apiPublicService   = "nd-api-public-disarm.service"
	apiPublicTimer     = "nd-api-public-disarm.timer"
)

type apiPublicState struct {
	Until   time.Time `json:"until"`
	ArmedAt time.Time `json:"armed_at"`
	TTL     string    `json:"ttl"`
}

func syncAPIPublicUnitDir() error {
	fd, err := unix.Open(apiPublicUnitDir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return fmt.Errorf("open systemd unit directory: %w", err)
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return fmt.Errorf("stat systemd unit directory: %w", err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != 0 || st.Mode&0o022 != 0 {
		_ = unix.Close(fd)
		return fmt.Errorf("systemd unit directory is not trusted")
	}
	f := os.NewFile(uintptr(fd), apiPublicUnitDir)
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync systemd unit directory: %w", err)
	}
	return f.Close()
}

func writeAPIPublicUnit(name string, content []byte) error {
	path := filepath.Join(apiPublicUnitDir, name)
	if err := syncAPIPublicUnitDir(); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(apiPublicUnitDir, ".netductor-api-public-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary systemd unit: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temporary systemd unit: %w", err)
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary systemd unit: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary systemd unit: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary systemd unit: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("install systemd unit %s: %w", name, err)
	}
	return syncAPIPublicUnitDir()
}

func removeAPIPublicUnit(name string) error {
	path := filepath.Join(apiPublicUnitDir, name)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove systemd unit %s: %w", name, err)
	}
	return syncAPIPublicUnitDir()
}

func ufwQuiet(args ...string) error {
	cmd := exec.Command("ufw", args...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// deleteUFWByComment removes rules whose status line contains comment (high→low numbers).
func deleteUFWByComment(comment string) error {
	out, err := exec.Command("ufw", "status", "numbered").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw status numbered: %w (%s)", err, strings.TrimSpace(string(out)))
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
		// argv only: rule number is data, must not go through a shell.
		cmd := exec.Command("ufw", "--force", "delete", n)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ufw delete numbered rule %s: %w", n, err)
		}
	}
	return nil
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
	if err := deleteUFWByComment(apiPublicComment); err != nil {
		return fmt.Errorf("remove previous temporary public rules: %w", err)
	}
	_ = exec.Command("systemctl", "stop", apiPublicTimer).Run()

	if err := ufwQuiet("allow", apiPublicPort+"/tcp", "comment", apiPublicComment); err != nil {
		return fmt.Errorf("ufw allow public %s: %w", apiPublicPort, err)
	}
	rollback := func(cause error) error {
		var cleanup []error
		if err := deleteUFWByComment(apiPublicComment); err != nil {
			cleanup = append(cleanup, fmt.Errorf("remove temporary public UFW rule: %w", err))
		}
		if out, err := exec.Command("systemctl", "disable", "--now", apiPublicTimer).CombinedOutput(); err != nil {
			cleanup = append(cleanup, fmt.Errorf("disable disarm timer: %v (%s)", err, strings.TrimSpace(string(out))))
		}
		if err := os.Remove(apiPublicStatePath); err != nil && !os.IsNotExist(err) {
			cleanup = append(cleanup, fmt.Errorf("remove API public state: %w", err))
		}
		if err := removeAPIPublicUnit(apiPublicTimer); err != nil {
			cleanup = append(cleanup, err)
		}
		if err := removeAPIPublicUnit(apiPublicService); err != nil {
			cleanup = append(cleanup, err)
		}
		if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
			cleanup = append(cleanup, fmt.Errorf("systemd daemon-reload: %v (%s)", err, strings.TrimSpace(string(out))))
		}
		if len(cleanup) != 0 {
			return errors.Join(append([]error{cause}, cleanup...)...)
		}
		return cause
	}
	st := apiPublicState{
		ArmedAt: time.Now().UTC(),
		Until:   time.Now().UTC().Add(ttl),
		TTL:     ttl.String(),
	}
	if err := os.MkdirAll(filepath.Dir(apiPublicStatePath), 0o755); err != nil {
		return rollback(fmt.Errorf("create API public state directory: %w", err))
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return rollback(fmt.Errorf("encode API public state: %w", err))
	}
	if err := os.WriteFile(apiPublicStatePath, b, 0o600); err != nil {
		return rollback(fmt.Errorf("write API public state: %w", err))
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
OnCalendar=%s
Persistent=true
AccuracySec=1s
Unit=%s
[Install]
WantedBy=timers.target
`, st.Until.UTC().Format("2006-01-02 15:04:05 UTC"), apiPublicService)
	if err := writeAPIPublicUnit(apiPublicService, []byte(unit)); err != nil {
		return rollback(fmt.Errorf("write disarm service: %w", err))
	}
	if err := writeAPIPublicUnit(apiPublicTimer, []byte(timer)); err != nil {
		return rollback(fmt.Errorf("write disarm timer: %w", err))
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return rollback(fmt.Errorf("daemon-reload: %v (%s)", err, string(out)))
	}
	if out, err := exec.Command("systemctl", "enable", "--now", apiPublicTimer).CombinedOutput(); err != nil {
		return rollback(fmt.Errorf("enable/start disarm timer: %v (%s)", err, string(out)))
	}
	if out, err := exec.Command("systemctl", "is-active", apiPublicTimer).CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "active" {
		return rollback(fmt.Errorf("disarm timer not active after enable: %s", strings.TrimSpace(string(out))))
	}
	return nil
}

// DisarmAPIPublic removes temporary public allow; keeps service-plane rules intact.
func DisarmAPIPublic() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		_ = os.Remove(apiPublicStatePath)
		return nil
	}
	if err := deleteUFWByComment(apiPublicComment); err != nil {
		return fmt.Errorf("remove temporary public API rules: %w", err)
	}
	// Remove the public exception first, then restore the restricted service-plane policy.
	if err := applyAgentFirewallQuiet(); err != nil {
		return fmt.Errorf("restore restricted agent firewall: %w", err)
	}
	if out, err := exec.Command("systemctl", "disable", "--now", apiPublicTimer).CombinedOutput(); err != nil {
		return fmt.Errorf("disable disarm timer: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	if err := removeAPIPublicUnit(apiPublicTimer); err != nil {
		return err
	}
	if err := removeAPIPublicUnit(apiPublicService); err != nil {
		return err
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemd daemon-reload after disarm: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	if err := os.Remove(apiPublicStatePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove API public state: %w", err)
	}
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
