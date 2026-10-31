package migrate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
)

// PostCheck verifies minimal host state after migrations (role-aware).
// Does not start services; filesystem + optional systemd when enabled.
func PostCheck() error {
	var errs []string
	for _, d := range criticalDirs() {
		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			errs = append(errs, "missing dir "+d)
			continue
		}
	}
	// Sensitive dirs must not be group/other-writable or readable
	for _, d := range []string{
		filepath.Join(paths.EtcDir(), "secrets"),
		filepath.Join(paths.EtcDir(), "sessions"),
		filepath.Join(paths.EtcDir(), "clients"),
		filepath.Join(paths.StateDir(), "migrate"),
	} {
		st, err := os.Stat(d)
		if err != nil {
			errs = append(errs, "missing sensitive "+d)
			continue
		}
		if st.Mode().Perm()&0o077 != 0 {
			errs = append(errs, fmt.Sprintf("mode %04o on %s (want 0700)", st.Mode().Perm(), d))
		}
	}
	roleFile := filepath.Join(paths.EtcDir(), "role")
	if b, err := os.ReadFile(roleFile); err != nil {
		errs = append(errs, "role file missing: "+roleFile)
	} else if len(strings.TrimSpace(string(b))) == 0 {
		errs = append(errs, "empty role file")
	}
	jp := journalPath()
	if _, err := os.Stat(jp); err != nil {
		errs = append(errs, "migrate journal missing after run")
	} else {
		j, err := loadJournal()
		if err != nil {
			errs = append(errs, "journal unreadable: "+err.Error())
		} else {
			for id, rec := range j.Records {
				if rec.Status == StatusRunning {
					errs = append(errs, "interrupted migration still running: "+id)
				}
				if rec.Status == StatusFailed {
					errs = append(errs, "failed migration recorded: "+id)
				}
			}
		}
	}
	// Optional: live units (opt-in; skips in unit tests / containers without systemd)
	if os.Getenv("NETDUCTOR_POSTCHECK_UNITS") == "1" {
		if _, err := os.Stat("/run/systemd/system"); err == nil {
			role := strings.TrimSpace(string(mustRead(roleFile)))
			units := []string{"netductor-api.service"}
			if role == "secondary" {
				units = []string{"netductor-secondary-agent.service"}
			}
			for _, u := range units {
				out, err := exec.Command("systemctl", "is-active", u).CombinedOutput()
				st := strings.TrimSpace(string(out))
				if err != nil || (st != "active" && st != "activating") {
					errs = append(errs, "unit "+u+" not active: "+st)
				}
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	msg := "post-check:"
	for _, e := range errs {
		msg += " " + e + ";"
	}
	return fmt.Errorf("%s", msg)
}

func mustRead(path string) []byte {
	b, _ := os.ReadFile(path)
	return b
}
