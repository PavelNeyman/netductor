package fleet

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// SyncPaths copies state directories to peer so manual/auto failover keeps progress.
// Does not move docker images — only data (lampac volume, components, fleet policy).
func SyncPaths(peerSSH string) error {
	peerSSH = strings.TrimSpace(peerSSH)
	if peerSSH == "" {
		// derive from backup.offsite target root@host:...
		peerSSH = peerHostFromBackupOffsite()
	}
	if peerSSH == "" {
		return fmt.Errorf("peer SSH user@host required (or configure backup peer-set)")
	}
	if !strings.Contains(peerSSH, "@") {
		peerSSH = "root@" + peerSSH
	}

	pathsToSync := []struct {
		local string
		remote string
	}{
		{filepath.Join(paths.OptDir(), "lampac"), "/opt/netductor/lampac"},
		{filepath.Join(paths.StateDir(), "components.json"), "/var/lib/netductor/components.json"},
		{filepath.Join(paths.StateDir(), "fleet"), "/var/lib/netductor/fleet"},
	}

	sshOpts := []string{"-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes"}
	var errs []string
	for _, p := range pathsToSync {
		if _, err := os.Stat(p.local); err != nil {
			continue
		}
		// ensure remote parent
		parent := filepath.Dir(p.remote)
		_ = exec.Command("ssh", append(sshOpts, peerSSH, "mkdir", "-p", parent)...).Run()
		args := append([]string{}, sshOpts...)
		// rsync if available, else scp -r
		if _, err := exec.LookPath("rsync"); err == nil {
			args = append([]string{"-az", "-e", "ssh " + strings.Join(sshOpts, " ")}, p.local+"/", peerSSH+":"+p.remote+"/")
			cmd := exec.Command("rsync", args...)
			if out, err := cmd.CombinedOutput(); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %s %v", p.local, strings.TrimSpace(string(out)), err))
			}
		} else {
			cmd := exec.Command("scp", append(sshOpts, "-r", p.local, peerSSH+":"+p.remote)...)
			if out, err := cmd.CombinedOutput(); err != nil {
				errs = append(errs, fmt.Sprintf("%s: %s %v", p.local, strings.TrimSpace(string(out)), err))
			}
		}
	}
	// stamp
	_ = os.MkdirAll(filepath.Join(paths.StateDir(), "fleet"), 0o700)
	_ = os.WriteFile(filepath.Join(paths.StateDir(), "fleet", "last_sync"),
		[]byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0o644)
	if len(errs) > 0 {
		return fmt.Errorf("sync partial: %s", strings.Join(errs, "; "))
	}
	return nil
}

func peerHostFromBackupOffsite() string {
	b, err := os.ReadFile(filepath.Join(paths.EtcDir(), "backup.offsite"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "target=") {
			t := strings.TrimPrefix(line, "target=")
			// root@host:/path
			if i := strings.Index(t, ":"); i > 0 {
				return t[:i]
			}
			return t
		}
	}
	return ""
}
