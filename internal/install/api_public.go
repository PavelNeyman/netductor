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

func openAPIPublicStateDir() (int, error) {
	if err := os.MkdirAll(filepath.Dir(apiPublicStatePath), 0o700); err != nil {
		return -1, fmt.Errorf("create API public state directory: %w", err)
	}
	fd, err := unix.Open(filepath.Dir(apiPublicStatePath), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return -1, fmt.Errorf("open API public state directory: %w", err)
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return -1, fmt.Errorf("stat API public state directory: %w", err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != 0 || st.Mode&0o022 != 0 {
		_ = unix.Close(fd)
		return -1, fmt.Errorf("API public state directory is not trusted")
	}
	return fd, nil
}

func saveAPIPublicState(st apiPublicState) error {
	dirFD, err := openAPIPublicStateDir()
	if err != nil {
		return err
	}
	defer unix.Close(dirFD)
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encode API public state: %w", err)
	}
	data = append(data, '\\n')
	const tmpName = ".api-public-arm.tmp"
	_ = unix.Unlinkat(dirFD, tmpName, 0)
	fd, err := unix.Openat(dirFD, tmpName, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return fmt.Errorf("create temporary API public state: %w", err)
	}
	file := os.NewFile(uintptr(fd), tmpName)
	if n, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = unix.Unlinkat(dirFD, tmpName, 0)
		return fmt.Errorf("write API public state: %w", err)
	} else if n != len(data) {
		_ = file.Close()
		_ = unix.Unlinkat(dirFD, tmpName, 0)
		return fmt.Errorf("write API public state: short write (%d of %d bytes)", n, len(data))
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		_ = unix.Unlinkat(dirFD, tmpName, 0)
		return fmt.Errorf("sync API public state: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = unix.Unlinkat(dirFD, tmpName, 0)
		return fmt.Errorf("close API public state: %w", err)
	}
	if err := unix.Renameat(dirFD, tmpName, dirFD, filepath.Base(apiPublicStatePath)); err != nil {
		_ = unix.Unlinkat(dirFD, tmpName, 0)
		return fmt.Errorf("install API public state: %w", err)
	}
	if err := unix.Fsync(dirFD); err != nil {
		return fmt.Errorf("sync API public state directory: %w", err)
	}
	return nil
}

func loadAPIPublicState() (apiPublicState, error) {
	var st apiPublicState
	dirFD, err := openAPIPublicStateDir()
	if err != nil {
		return st, err
	}
	defer unix.Close(dirFD)
	fd, err := unix.Openat(dirFD, filepath.Base(apiPublicStatePath), unix.O_RDONLY|unix.O_CLOEXEC|unix.O_NOFOLLOW|unix.O_NONBLOCK, 0)
	if err != nil {
		return st, err
	}
	file := os.NewFile(uintptr(fd), apiPublicStatePath)
	var fs unix.Stat_t
	if err := unix.Fstat(fd, &fs); err != nil {
		_ = file.Close()
		return st, fmt.Errorf("stat API public state: %w", err)
	}
	if fs.Mode&unix.S_IFMT != unix.S_IFREG || fs.Uid != 0 || fs.Mode&0o077 != 0 || fs.Size > 4096 {
		_ = file.Close()
		return st, fmt.Errorf("API public state file is not a trusted root-owned regular file")
	}
	dec := json.NewDecoder(file)
	if err := dec.Decode(&st); err != nil {
		_ = file.Close()
		return st, fmt.Errorf("decode API public state: %w", err)
	}
	if err := file.Close(); err != nil {
		return st, fmt.Errorf("close API public state: %w", err)
	}
	if st.Until.IsZero() || st.ArmedAt.IsZero() || st.Until.Before(st.ArmedAt) {
		return st, fmt.Errorf("API public state has invalid timestamps")
	}
	return st, nil
}

func removeAPIPublicState() error {
	dirFD, err := openAPIPublicStateDir()
	if err != nil {
		return err
	}
	defer unix.Close(dirFD)
	if err := unix.Unlinkat(dirFD, filepath.Base(apiPublicStatePath), 0); err != nil && err != unix.ENOENT {
		return fmt.Errorf("remove API public state: %w", err)
	}
	if err := unix.Fsync(dirFD); err != nil {
		return fmt.Errorf("sync API public state directory after removal: %w", err)
	}
	return nil
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

func syncAPIPublicTimerEnablement() error {
	dir := filepath.Join(apiPublicUnitDir, "timers.target.wants")
	link := filepath.Join(dir, apiPublicTimer)
	linkInfo, err := os.Lstat(link)
	if err != nil {
		return fmt.Errorf("stat enabled disarm timer link: %w", err)
	}
	if linkInfo.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("enabled disarm timer is not a symlink")
	}
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		return fmt.Errorf("resolve enabled disarm timer link: %w", err)
	}
	if filepath.Clean(resolved) != filepath.Join(apiPublicUnitDir, apiPublicTimer) {
		return fmt.Errorf("enabled disarm timer link targets unexpected unit %q", resolved)
	}
	fd, err := unix.Open(dir, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return fmt.Errorf("open timer enablement directory: %w", err)
	}
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		_ = unix.Close(fd)
		return fmt.Errorf("stat timer enablement directory: %w", err)
	}
	if st.Mode&unix.S_IFMT != unix.S_IFDIR || st.Uid != 0 || st.Mode&0o022 != 0 {
		_ = unix.Close(fd)
		return fmt.Errorf("timer enablement directory is not trusted")
	}
	f := os.NewFile(uintptr(fd), dir)
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return fmt.Errorf("sync timer enablement directory: %w", err)
	}
	return f.Close()
}

func disableAPIPublicTimerIfPresent() error {
	activeOut, _ := exec.Command("systemctl", "is-active", apiPublicTimer).CombinedOutput()
	if strings.TrimSpace(string(activeOut)) == "active" {
		if out, err := exec.Command("systemctl", "stop", apiPublicTimer).CombinedOutput(); err != nil {
			return fmt.Errorf("stop disarm timer: %v (%s)", err, strings.TrimSpace(string(out)))
		}
	}
	enabledOut, _ := exec.Command("systemctl", "is-enabled", apiPublicTimer).CombinedOutput()
	enabled := strings.TrimSpace(string(enabledOut))
	if enabled == "enabled" || enabled == "enabled-runtime" || enabled == "linked" || enabled == "linked-runtime" {
		if out, err := exec.Command("systemctl", "disable", apiPublicTimer).CombinedOutput(); err != nil {
			return fmt.Errorf("disable disarm timer: %v (%s)", err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func removeAPIPublicTimerUnits() error {
	if err := disableAPIPublicTimerIfPresent(); err != nil {
		return err
	}
	if err := removeAPIPublicUnit(apiPublicTimer); err != nil {
		return err
	}
	if err := removeAPIPublicUnit(apiPublicService); err != nil {
		return err
	}
	// Remove transient units written by releases before the reboot-safe timer.
	for _, name := range []string{apiPublicTimer, apiPublicService} {
		path := filepath.Join("/run/systemd/system", name)
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove legacy transient unit %s: %w", name, err)
		}
	}
	if out, err := exec.Command("systemctl", "daemon-reload").CombinedOutput(); err != nil {
		return fmt.Errorf("systemd daemon-reload: %v (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
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

// deleteUFWUnrestrictedPort removes only unrestricted ALLOW IN Anywhere rules
// for a TCP port. CIDR-scoped service rules are left untouched.
func deleteUFWUnrestrictedPort(port string) error {
	out, err := exec.Command("ufw", "status", "numbered").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ufw status numbered: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	want := port + "/tcp"
	var nums []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "[") {
			continue
		}
		end := strings.Index(line, "]")
		if end < 0 {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(line[end+1:]))
		if len(fields) < 4 || fields[0] != want {
			continue
		}
		i := 1
		if fields[i] == "(v6)" {
			i++
		}
		if len(fields) > i+2 && fields[i] == "ALLOW" && fields[i+1] == "IN" && fields[i+2] == "Anywhere" {
			nums = append([]string{strings.TrimSpace(line[1:end])}, nums...)
		}
	}
	for _, n := range nums {
		cmd := exec.Command("ufw", "--force", "delete", n)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("ufw delete unrestricted %s rule %s: %w (%s)", port, n, err, strings.TrimSpace(string(out)))
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
	if err := removeAPIPublicTimerUnits(); err != nil {
		return fmt.Errorf("clear previous API public disarm timer: %w", err)
	}
	rollback := func(cause error) error {
		var cleanup []error
		if err := deleteUFWByComment(apiPublicComment); err != nil {
			cleanup = append(cleanup, fmt.Errorf("remove temporary public UFW rule: %w", err))
		}
		if err := removeAPIPublicTimerUnits(); err != nil {
			cleanup = append(cleanup, err)
		}
		if err := removeAPIPublicState(); err != nil {
			cleanup = append(cleanup, err)
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
	if err := saveAPIPublicState(st); err != nil {
		return rollback(fmt.Errorf("save API public state: %w", err))
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
	if err := syncAPIPublicTimerEnablement(); err != nil {
		return rollback(fmt.Errorf("persist disarm timer enablement: %w", err))
	}
	if out, err := exec.Command("systemctl", "is-active", apiPublicTimer).CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "active" {
		return rollback(fmt.Errorf("disarm timer not active after enable: %s", strings.TrimSpace(string(out))))
	}
	// The persistent absolute-time disarm is armed before the persistent UFW
	// exception is created. A crash at any later point cannot strand public access.
	if err := ufwQuiet("allow", apiPublicPort+"/tcp", "comment", apiPublicComment); err != nil {
		return rollback(fmt.Errorf("ufw allow public %s: %w", apiPublicPort, err))
	}
	return nil
}

// DisarmAPIPublic removes temporary public allow; keeps service-plane rules intact.
func DisarmAPIPublic() error {
	if _, err := exec.LookPath("ufw"); err != nil {
		if err := removeAPIPublicTimerUnits(); err != nil {
			return err
		}
		if err := removeAPIPublicState(); err != nil {
			return err
		}
		return nil
	}
	if err := deleteUFWByComment(apiPublicComment); err != nil {
		return fmt.Errorf("remove temporary public API rules: %w", err)
	}
	// Remove the public exception first, then restore the restricted service-plane policy.
	if err := applyAgentFirewallQuiet(); err != nil {
		return fmt.Errorf("restore restricted agent firewall: %w", err)
	}
	if err := removeAPIPublicTimerUnits(); err != nil {
		return err
	}
	if err := os.Remove(apiPublicStatePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove API public state: %w", err)
	}
	return nil
}

// APIPublicStatus returns armed flag and until time.
func APIPublicStatusChecked() (armed bool, until time.Time, err error) {
	st, err := loadAPIPublicState()
	if err != nil {
		if os.IsNotExist(err) {
			return false, time.Time{}, nil
		}
		return false, time.Time{}, err
	}
	if !time.Now().UTC().Before(st.Until) {
		if err := DisarmAPIPublic(); err != nil {
			return false, st.Until, fmt.Errorf("API public TTL expired but disarm failed: %w", err)
		}
		return false, time.Time{}, nil
	}
	return true, st.Until, nil
}

// APIPublicStatus is retained for callers that only need a best-effort display.
func APIPublicStatus() (bool, time.Time) {
	armed, until, _ := APIPublicStatusChecked()
	return armed, until
}
