package deploy

import (
	"fmt"
	"os"
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

// BootstrapPresetOnRouter installs a fixed preset (packages from catalog only). Fail-closed.
func BootstrapPresetOnRouter(presetID, password, keyPath, user, host, keyPassphrase string) error {
	p, err := BootstrapPresetByID(presetID)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "==> edge bootstrap preset=%s packages=%d\n", p.ID, len(p.Packages))
	switch p.ID {
	case "usb-storage":
		return EnsureUSBStorageOnRouter(password, keyPath, user, host, keyPassphrase)
	default:
		return fmt.Errorf("bootstrap install not implemented for preset %q", p.ID)
	}
}
