package deploy

import (
	"fmt"
	"os"
)

// applyLuCIDefaultsOnEdge disables LuCI "check online for firmware upgrades" on Status page.
// Netductor owns edge image lifecycle; ASU popup is noise and Close does not persist preference.
func applyLuCIDefaultsOnEdge(o EdgeOpts) error {
	script := `set +e
# Prefer attendedsysupgrade.client (current LuCI); migrate legacy luci.main key away.
if uci -q get attendedsysupgrade >/dev/null 2>&1 || [ -f /etc/config/attendedsysupgrade ]; then
  uci -q set attendedsysupgrade.client=client
  uci -q set attendedsysupgrade.client.login_check_for_upgrades='0'
  uci -q commit attendedsysupgrade
fi
if uci -q get luci.main >/dev/null 2>&1; then
  uci -q delete luci.main.check_for_newer_firmwares
  uci -q set luci.main.check_for_newer_firmwares='0'
  uci -q commit luci
fi
true
`
	fmt.Fprintln(os.Stderr, "==> luci: disable online firmware upgrade check on Status")
	out, err := runSSHOnPort(factorySSHPort(), o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, script, o.PrimaryKeyPassphrase)
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}
