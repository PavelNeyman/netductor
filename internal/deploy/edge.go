package deploy

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/edgeagent"
)

type EdgeOpts struct {
	PrimaryHost          string
	PrimaryUser          string
	PrimaryKey           string
	PrimaryKeyPassphrase string
	// Offline: skip SSH to primary when BootstrapToken is set (optional local mTLS files).
	BootstrapToken  string
	MTLSCAFile      string
	MTLSCertFile    string
	MTLSKeyFile     string
	RouterHost      string
	RouterUser      string
	RouterPass      string
	NewRootPassword string // LuCI/root pass; required unless SkipRootPass
	SkipRootPass    bool
	DeviceID        string
	ServerURL       string
	AgentArch       string
	Version         string
	AgentDir        string
	OperatorPubKey  string
	GuestEnable     bool
	GuestSSID       string
	GuestPIN        string
	GuestPSK        string
	GuestVisible    bool // true = broadcast SSID (default for forms); false = hidden
	// Optional first-boot network (applied via UCI on router + edge template)
	NetConfigure bool
	LANIP        string
	LANMask      string
	DHCPStart    string // e.g. 100
	DHCPLimit    string // e.g. 150
	WiFiSSID     string // legacy both bands
	WiFiKey      string
	WiFiSSID24   string
	WiFiKey24    string
	WiFiSSID5    string
	WiFiKey5     string
	WANProto     string // dhcp | static | pppoe
	WANIP        string
	WANMask      string
	WANGateway   string
	WANDNS       string
	PPPoEUser    string
	PPPoEPass    string
	PPPoEService string
	PPPoEAC      string
	// Reboot router after provision (agent init + optional network).
	Reboot bool
	// Hybrid deploy plan (P1+).
	Preset     string               // travel-router | sbc-lab | sbc-dual-nic | empty=suggest
	DryRun     bool                 // probe facts + print plan, no provision
	Selection  edge.ModuleSelection // optional UI/operator module toggles
	DryRunJSON bool                 // with DryRun: print PlanResponse JSON (thin UI contract)
	// Set by plan (module gates); empty = legacy NetConfigure behavior.
	ApplyLAN, ApplyWAN, ApplyWiFi, ApplyExpand, ApplyOverlay bool
	ExpandFS, OverlayExt                                     bool // operator intent for optional storage modules
}

func networkTemplateMap(o EdgeOpts) map[string]any {
	// Non-destructive: only emit keys the operator set. WAN defaults to dhcp only when WAN module applies.
	m := map[string]any{}
	net := map[string]any{}
	allowLAN, allowWAN, allowWiFi := o.ApplyLAN, o.ApplyWAN, o.ApplyWiFi
	if !o.ApplyLAN && !o.ApplyWAN && !o.ApplyWiFi && o.NetConfigure {
		// Legacy path before plan flags: configure all requested via flags.
		allowLAN, allowWAN, allowWiFi = true, true, true
	}

	if allowLAN {
		if o.LANIP != "" {
			net["lan_ip"] = o.LANIP
		}
		if o.LANMask != "" {
			net["lan_mask"] = o.LANMask
		} else if o.LANIP != "" {
			net["lan_mask"] = "255.255.255.0"
		}
	}
	if allowWAN {
		proto := strings.ToLower(strings.TrimSpace(o.WANProto))
		if proto == "" {
			proto = "dhcp"
		}
		net["wan_proto"] = proto
		if proto == "static" {
			if o.WANIP != "" {
				net["wan_ip"] = o.WANIP
			}
			if o.WANMask != "" {
				net["wan_mask"] = o.WANMask
			} else if o.WANIP != "" {
				net["wan_mask"] = "255.255.255.0"
			}
			if o.WANGateway != "" {
				net["wan_gateway"] = o.WANGateway
			}
			if o.WANDNS != "" {
				net["wan_dns"] = o.WANDNS
			}
		}
		if proto == "pppoe" {
			if o.PPPoEUser != "" {
				net["pppoe_user"] = o.PPPoEUser
			}
			if o.PPPoEPass != "" {
				net["pppoe_pass"] = o.PPPoEPass
			}
			if o.PPPoEService != "" {
				net["pppoe_service"] = o.PPPoEService
			}
			if o.PPPoEAC != "" {
				net["pppoe_ac"] = o.PPPoEAC
			}
			if o.WANDNS != "" {
				net["wan_dns"] = o.WANDNS
			}
		}
	}
	if len(net) > 0 {
		m["network"] = net
	}
	if allowLAN && (o.DHCPStart != "" || o.DHCPLimit != "") {
		m["dhcp"] = map[string]any{"start": o.DHCPStart, "limit": o.DHCPLimit}
	}
	if allowWiFi {
		wifi := map[string]any{"encryption": "psk2"}
		if o.WiFiSSID24 != "" || o.WiFiKey24 != "" || o.WiFiSSID5 != "" || o.WiFiKey5 != "" {
			wifi["ssid_24"] = o.WiFiSSID24
			wifi["key_24"] = o.WiFiKey24
			wifi["ssid_5"] = o.WiFiSSID5
			wifi["key_5"] = o.WiFiKey5
		} else if o.WiFiSSID != "" {
			wifi["ssid"] = o.WiFiSSID
			wifi["key"] = o.WiFiKey
		}
		if o.WiFiSSID24 != "" || o.WiFiSSID5 != "" || o.WiFiSSID != "" {
			m["wifi"] = wifi
		}
	}
	return m
}

func applyFSExpandOnEdge(o EdgeOpts) error {
	script := `set +e
if ! ls /dev/mmcblk0 >/dev/null 2>&1; then echo "no mmc"; exit 0; fi
if command -v resize2fs >/dev/null 2>&1; then
  ROOT=$(findmnt -n -o SOURCE / 2>/dev/null)
  echo "expand attempt root=$ROOT"
  resize2fs "$ROOT" 2>/dev/null && echo "resize2fs ok" || echo "resize2fs skipped"
else
  echo "resize2fs not installed"
fi
true
`
	out, err := runSSHOnPort(factorySSHPort(), o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, script, "")
	fmt.Fprintln(os.Stderr, "==> fs expand:", strings.TrimSpace(out))
	return err
}

func applyNetworkOnEdge(o EdgeOpts) error {
	tmpl := networkTemplateMap(o)
	if len(tmpl) == 0 {
		return nil
	}
	if o.PrimaryHost != "" && o.DeviceID != "" {
		_ = edge.BindTemplate(o.DeviceID, "default", tmpl)
		_ = edge.EnqueueCmd(o.DeviceID, "apply_template", "")
	}
	desired := edgeagent.DesiredUCI(tmpl)
	script := edgeagent.ShellApplyStaged(desired)
	out, err := runSSHOnPort(factorySSHPort(), o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, script, "")
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	fmt.Fprintln(os.Stderr, "==> network UCI staged (commit, no reload) on", o.RouterHost)
	return nil
}

func applyGuestOnEdge(o EdgeOpts) error {
	ssid := o.GuestSSID
	if ssid == "" {
		ssid = "Guest"
	}
	pin := o.GuestPIN
	psk := o.GuestPSK
	// Persist guest into edge template only when we have a device id (day-2 apply_template).
	// First-boot source of truth is SSH guest enable below.
	if o.PrimaryHost != "" && strings.TrimSpace(o.DeviceID) != "" {
		_ = edge.BindTemplate(o.DeviceID, "default", map[string]any{
			"guest": map[string]any{
				"enabled":  true,
				"ssid":     ssid,
				"desk_pin": pin,
				"psk":      psk,
				"hidden":   !o.GuestVisible,
			},
		})
		_ = edge.EnqueueCmd(o.DeviceID, "apply_template", "")
	}
	// First-boot: stage guest UCI without network/wifi reload (keeps SSH until harden+reboot).
	cmd := "/usr/sbin/netductor-agent guest enable --stage --ssid=" + shellQuote(ssid) + " --pin=" + shellQuote(pin)
	if psk != "" {
		cmd += " --psk=" + shellQuote(psk)
	}
	if o.GuestVisible {
		cmd += " --hidden=0"
	} else {
		cmd += " --hidden=1"
	}
	out, err := runSSHOnPort(factorySSHPort(), o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, cmd, "")
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	fmt.Fprintln(os.Stderr, "==> guest Wi-Fi staged on", o.RouterHost, "(credentials not logged)")
	// Legacy agents wrote a full nft table into fw4 include dir and broke firewall.
	_, _ = runSSHOnPort(factorySSHPort(), o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost,
		"rm -f /etc/nftables.d/90-netductor-guest.nft; true", "")

	_ = out

	// PLAN-EDGE-USB-STORAGE: auto A+B backup when device known
	if strings.TrimSpace(o.DeviceID) != "" {
		_ = edge.EnqueueCmd(o.DeviceID, "config_backup", "")
		_ = edge.EnqueueCmd(o.DeviceID, "storage_backup", "")
		fmt.Fprintln(os.Stderr, "==> queued storage backups for", o.DeviceID)
	}
	return nil
}

func serverURLUnsafe(u string) bool {
	for _, r := range u {
		switch r {
		case ' ', '\t', '\n', '\r', ';', '|', '&', '$', '`', '"', '\'', '\\', '<', '>', '(', ')', '{', '}':
			return true
		}
	}
	return false
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func planStepApply(plan edge.DeployPlan, mod string) bool {
	for _, s := range plan.Steps {
		if s.Module == mod && s.Action == "apply" {
			return true
		}
	}
	return false
}
