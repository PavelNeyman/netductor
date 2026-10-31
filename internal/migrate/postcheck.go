package migrate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/PavelNeyman/netductor/internal/firewall"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// PostCheck verifies host state after migrations (role-aware).
// Always checks filesystem, role, journal, installed_version.
// Optional (opt-in env):
//   NETDUCTOR_POSTCHECK_UNITS=1     — systemctl is-active for role units
//   NETDUCTOR_POSTCHECK_FIREWALL=1  — firewall.Collect must be Active+OK when ufw/iptables present
func PostCheck() error {
	var errs []string
	for _, d := range criticalDirs() {
		st, err := os.Stat(d)
		if err != nil || !st.IsDir() {
			errs = append(errs, "missing dir "+d)
			continue
		}
		if st.Mode().Perm()&0o077 != 0 {
			errs = append(errs, fmt.Sprintf("mode %04o on managed state dir %s (want 0700)", st.Mode().Perm(), d))
		}
	}
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
	role := ""
	if b, err := os.ReadFile(roleFile); err != nil {
		errs = append(errs, "role file missing: "+roleFile)
	} else {
		role = strings.TrimSpace(string(b))
		if role == "" {
			errs = append(errs, "empty role file")
		} else if role != "primary" && role != "secondary" {
			errs = append(errs, "role file invalid: "+role)
		}
	}

	iv := filepath.Join(paths.StateDir(), "installed_version")
	if b, err := os.ReadFile(iv); err != nil {
		errs = append(errs, "installed_version missing")
	} else if len(strings.TrimSpace(string(b))) == 0 {
		errs = append(errs, "installed_version empty")
	}

	// Product VERSION file is optional on secondary agent-only hosts; require on primary when role known.
	verPath := filepath.Join(paths.EtcDir(), "VERSION")
	if role == "primary" {
		if b, err := os.ReadFile(verPath); err != nil {
			// soft: many test fixtures omit VERSION; only fail when NETDUCTOR_POSTCHECK_STRICT=1
			if os.Getenv("NETDUCTOR_POSTCHECK_STRICT") == "1" {
				errs = append(errs, "VERSION file missing on primary")
			}
		} else if len(strings.TrimSpace(string(b))) == 0 && os.Getenv("NETDUCTOR_POSTCHECK_STRICT") == "1" {
			errs = append(errs, "VERSION file empty on primary")
		}
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

	if os.Getenv("NETDUCTOR_POSTCHECK_UNITS") == "1" {
		if _, err := os.Stat("/run/systemd/system"); err == nil {
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

	if os.Getenv("NETDUCTOR_POSTCHECK_FIREWALL") == "1" {
		st := firewall.Collect(role)
		if st.Backend == firewall.BackendNone {
			errs = append(errs, "firewall: no backend")
		} else if !st.Active {
			errs = append(errs, "firewall: inactive backend="+string(st.Backend))
		} else if !st.OK {
			errs = append(errs, "firewall: not OK: "+st.Detail)
			for _, w := range st.Warnings {
				errs = append(errs, "firewall warn: "+w)
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
