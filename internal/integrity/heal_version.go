package integrity

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	ndupdate "github.com/PavelNeyman/netductor/internal/update"
)

// HealStaleVersionAfterSkipUpdate rewrites /etc/netductor/VERSION and the integrity
// manifest when the only drift is the VERSION file and/or node binary after an
// intentional multi-release jump (manual replace or stack apply that skipped tags).
// It does not touch SSH keys. Safe to call on every metrics tick: no-op when clean.
func HealStaleVersionAfterSkipUpdate() (healed bool, detail string) {
	d := Check()
	if !d.HaveManifest || d.OK {
		return false, ""
	}
	if len(d.NewSSH) > 0 || len(d.MissingSSH) > 0 {
		return false, "ssh keys changed — refuse auto-heal"
	}
	onlyMeta := true
	for _, f := range d.ChangedFiles {
		base := f
		if i := strings.Index(f, " "); i > 0 {
			base = f[:i]
		}
		base = strings.TrimSuffix(base, " (now missing)")
		switch filepath.Clean(base) {
		case "/etc/netductor/VERSION", "/usr/local/bin/netductor", "/usr/local/bin/netductor-tg":
			continue
		default:
			onlyMeta = false
		}
	}
	if !onlyMeta || len(d.ChangedFiles) == 0 {
		return false, "non-version files drifted"
	}
	binVer := strings.TrimSpace(binaryVersionField("/usr/local/bin/netductor"))
	if binVer == "" {
		return false, "cannot read binary version"
	}
	ndupdate.WriteVERSION(binVer)
	if err := WriteManifest(); err != nil {
		return false, "write manifest: " + err.Error()
	}
	return true, "VERSION+manifest aligned to binary " + binVer
}

func binaryVersionField(path string) string {
	out, err := exec.Command(path, "version").CombinedOutput()
	if err != nil {
		// fallback: first line may still contain version
		s := string(out)
		return parseVerToken(s)
	}
	return parseVerToken(string(out))
}

func parseVerToken(s string) string {
	for _, f := range strings.Fields(s) {
		f = strings.TrimPrefix(f, "v")
		if len(f) > 0 && f[0] >= '0' && f[0] <= '9' && strings.Contains(f, ".") {
			return f
		}
	}
	// VERSION file as last resort
	if b, err := os.ReadFile("/etc/netductor/VERSION"); err == nil {
		return strings.TrimSpace(string(b))
	}
	return ""
}
