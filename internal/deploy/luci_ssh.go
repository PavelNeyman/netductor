package deploy

import (
	"fmt"
	"strings"
	"time"
)

// LuciSSH runs enable|disable|extend|status on OpenWrt over SSH (LAN; no agent required).
func LuciSSH(password, keyPath, user, host, action string, hours float64, keyPassphrase string) (string, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	if hours <= 0 {
		hours = 1
	}
	if hours > 168 {
		hours = 168
	}
	script := luciSSHScript(action, hours)
	out, err := runSSH(password, keyPath, user, host, script, keyPassphrase)
	if err != nil {
		return out, fmt.Errorf("luci ssh: %w: %s", err, out)
	}
	return strings.TrimSpace(out), nil
}

func luciSSHScript(action string, hours float64) string {
	dir := "/etc/netductor-agent"
	until := time.Now().Add(time.Duration(hours * float64(time.Hour))).Unix()
	switch action {
	case "disable":
		return fmt.Sprintf(`set -e
rm -f %s/luci_until
/etc/init.d/uhttpd stop 2>/dev/null || true
echo luci_disabled
`, dir)
	case "status":
		return fmt.Sprintf(`
until=none
[ -f %s/luci_until ] && until=$(cat %s/luci_until)
st=$(/etc/init.d/uhttpd status 2>/dev/null || echo unknown)
echo "status=$st until=$until"
`, dir, dir)
	case "extend":
		sec := int(hours * 3600)
		return fmt.Sprintf(`set -e
mkdir -p %s
now=$(date +%%s)
cur=0
[ -f %s/luci_until ] && cur=$(cat %s/luci_until)
[ "$cur" -gt "$now" ] || cur=$now
echo $((cur + %d)) > %s/luci_until
/etc/init.d/uhttpd start 2>/dev/null || /etc/init.d/uhttpd restart 2>/dev/null || true
echo luci_extended_until=$(cat %s/luci_until)
`, dir, dir, dir, sec, dir, dir)
	default: // enable
		return fmt.Sprintf(`set -e
mkdir -p %s
echo %d > %s/luci_until
/etc/init.d/uhttpd enable 2>/dev/null || true
/etc/init.d/uhttpd start 2>/dev/null || /etc/init.d/uhttpd restart 2>/dev/null || true
echo luci_enabled_until=%d
`, dir, until, dir, until)
	}
}
