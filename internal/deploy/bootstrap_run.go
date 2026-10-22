package deploy

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ProbeInstalledPackages returns installed package base names and mod:NAME from lsmod.
func ProbeInstalledPackages(password, keyPath, user, host, keyPassphrase string) (map[string]bool, error) {
	script := `
set +e
if command -v opkg >/dev/null 2>&1; then
  opkg list-installed 2>/dev/null | awk '{print "PKG="$1}'
elif command -v apk >/dev/null 2>&1; then
  apk info 2>/dev/null | awk '{print "PKG="$1}'
fi
lsmod 2>/dev/null | awk 'NR>1 {print "MOD="$1}'
true
`
	out, err := runSSHOnPort(factorySSHPort(), password, keyPath, user, host, script, keyPassphrase)
	installed := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "PKG="):
			name := strings.TrimPrefix(line, "PKG=")
			if i := strings.IndexByte(name, ' '); i > 0 {
				name = name[:i]
			}
			if name != "" {
				installed[name] = true
			}
		case strings.HasPrefix(line, "MOD="):
			mod := strings.TrimPrefix(line, "MOD=")
			if mod != "" {
				installed["mod:"+mod] = true
			}
		}
	}
	if len(installed) == 0 && err != nil {
		return installed, fmt.Errorf("probe installed packages: %w (%s)", err, strings.TrimSpace(out))
	}
	return installed, nil
}

// probeOverlayFreeKB reads free KiB on /overlay or / (0 = unknown/failed).
func probeOverlayFreeKB(password, keyPath, user, host, keyPassphrase string) int64 {
	script := `
set +e
if command -v df >/dev/null; then
  line=$(df -k /overlay 2>/dev/null | tail -1)
  if [ -n "$line" ]; then
    echo "$line" | awk '{print $4}'
    exit 0
  fi
  line=$(df -k / 2>/dev/null | tail -1)
  if [ -n "$line" ]; then
    echo "$line" | awk '{print $4}'
  fi
fi
`
	out, err := runSSHOnPort(factorySSHPort(), password, keyPath, user, host, script, keyPassphrase)
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(out)
	// take last line if noise
	if i := strings.LastIndex(s, "\n"); i >= 0 {
		s = strings.TrimSpace(s[i+1:])
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// BootstrapPresetOnRouter installs a fixed preset (packages from catalog only). Fail-closed.
// Re-checks overlay free space on the router before install (not only UI Selectable).
func BootstrapPresetOnRouter(presetID, password, keyPath, user, host, keyPassphrase string) error {
	p, err := BootstrapPresetByID(presetID)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "==> edge bootstrap preset=%s packages=%d\n", p.ID, len(p.Packages))

	// Server-side capacity gate (CLI/job/API cannot bypass UI Selectable).
	if p.MinOverlayFreeKB > 0 {
		free := probeOverlayFreeKB(password, keyPath, user, host, keyPassphrase)
		if free <= 0 {
			return fmt.Errorf("bootstrap %s: cannot determine overlay free space — refuse install (need ≥%d KB)", p.ID, p.MinOverlayFreeKB)
		}
		if free < p.MinOverlayFreeKB {
			return fmt.Errorf("bootstrap %s: overlay free %d KB < need %d KB — refuse install", p.ID, free, p.MinOverlayFreeKB)
		}
		fmt.Fprintf(os.Stderr, "==> overlay free %d KB (min %d)\n", free, p.MinOverlayFreeKB)
	}

	switch p.ID {
	case "usb-storage":
		return EnsureUSBStorageOnRouter(password, keyPath, user, host, keyPassphrase)
	default:
		return fmt.Errorf("bootstrap install not implemented for preset %q", p.ID)
	}
}
