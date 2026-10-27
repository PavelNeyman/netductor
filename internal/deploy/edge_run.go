package deploy

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/edge"
	"github.com/PavelNeyman/netductor/internal/mtls"
)

func DeployEdge(o EdgeOpts) error {
	if o.RouterHost == "" || o.DeviceID == "" {
		return fmt.Errorf("router host and device id required")
	}
	if o.RouterUser == "" {
		o.RouterUser = "root"
	}
	if o.PrimaryUser == "" {
		o.PrimaryUser = "root"
	}
	if o.Version == "" {
		o.Version = Release
	}
	if !validReleaseVersion(o.Version) {
		return fmt.Errorf("invalid release version %q", o.Version)
	}
	if !ValidAgentArch(o.AgentArch) {
		return fmt.Errorf("invalid agent arch %q (amd64|arm64|arm|mipsle|riscv64|auto)", o.AgentArch)
	}
	if o.AgentDir == "" {
		home, _ := os.UserHomeDir()
		o.AgentDir = filepath.Join(home, ".cache", "netductor", "agents")
	}
	if o.ServerURL == "" && o.PrimaryHost != "" {
		o.ServerURL = "https://" + o.PrimaryHost + ":" + mtls.AgentTLSPort
	}
	if o.ServerURL == "" {
		if o.DryRun {
			o.ServerURL = "https://dry-run.invalid:8789"
		} else {
			return fmt.Errorf("server URL or primary host required")
		}
	}
	// Never use plain public admin :8787 for edge control.
	if strings.HasPrefix(o.ServerURL, "http://") {
		u := strings.TrimPrefix(o.ServerURL, "http://")
		u = strings.Replace(u, ":8787", ":"+mtls.AgentTLSPort, 1)
		if !strings.Contains(u, ":") {
			u = u + ":" + mtls.AgentTLSPort
		}
		o.ServerURL = "https://" + u
	}
	if !strings.HasPrefix(o.ServerURL, "https://") {
		return fmt.Errorf("server URL must be https:// (agent plane)")
	}
	if serverURLUnsafe(o.ServerURL) {
		return fmt.Errorf("invalid server URL characters")
	}

	// Hybrid plan: probe facts (SSH), build module plan. Dry-run stops here.
	{
		preset := strings.TrimSpace(o.Preset)
		if preset != "" && !edge.ValidPreset(preset) {
			return fmt.Errorf("unknown preset %q (travel-router|sbc-lab|sbc-dual-nic)", preset)
		}
		// Probe is always read-only. USB packages: edge.bootstrap only.
		facts, raw, perr := ProbeDeviceFacts(o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, o.PrimaryKeyPassphrase, false)
		if perr != nil {
			return fmt.Errorf("facts probe failed — refuse deploy without device plan: %w", perr)
		}
		{
			intent := edge.DeployIntent{
				ConfigureNet: o.NetConfigure,
				GuestEnable:  o.GuestEnable,
				VPNEnable:    o.Selection.Enabled != nil && o.Selection.Enabled[edge.ModVPNClient],
				WiFiSSID:     firstNonEmpty(o.WiFiSSID24, o.WiFiSSID, o.WiFiSSID5),
				ExpandFS:     o.ExpandFS,
				OverlayExt:   o.OverlayExt,
			}
			if preset == "" {
				preset = edge.SuggestPreset(facts)
				fmt.Fprintln(os.Stderr, "==> auto-preset:", preset)
			}
			plan := edge.BuildPlan(preset, facts, intent)
			plan = edge.ApplySelection(plan, o.Selection, facts)
			fmt.Fprintln(os.Stderr, "==> deploy plan")
			fmt.Fprint(os.Stderr, FormatPlanHuman(preset, facts, plan))
			for _, s := range plan.Steps {
				fmt.Fprintf(os.Stderr, "==> module %-16s %-5s %s\n", s.Module, s.Action, s.Reason)
			}
			o.Preset = plan.Preset
			if o.DryRun {
				fmt.Fprintln(os.Stderr, "==> dry-run: no provision (read-only; USB disks unknown until real deploy bootstraps kmods)")
				if o.DryRunJSON {
					resp := edge.PlanResponse{
						Preset: plan.Preset, Suggested: edge.SuggestPreset(facts),
						Facts: facts, Plan: plan, Card: edge.CardFromFacts(facts),
					}
					b, _ := json.MarshalIndent(resp, "", "  ")
					fmt.Println(string(b))
				}
				_ = raw
				return nil
			}
			// Gate network modules by plan.
			if o.NetConfigure && !planStepApply(plan, edge.ModWANBaseline) {
				fmt.Fprintln(os.Stderr, "==> plan: skip wan UCI")
				if o.WANProto != "" {
					fmt.Fprintln(os.Stderr, "==> ignoring --wan-proto (not in plan)")
					o.WANProto = ""
				}
			}
			if o.GuestEnable && !planStepApply(plan, edge.ModGuest) {
				fmt.Fprintln(os.Stderr, "==> plan: skip guest (no radios or disabled)")
				o.GuestEnable = false
			}
			o.ApplyLAN = planStepApply(plan, edge.ModLANBaseline)
			o.ApplyWAN = planStepApply(plan, edge.ModWANBaseline)
			o.ApplyWiFi = planStepApply(plan, edge.ModWiFiAP)
			o.ApplyExpand = planStepApply(plan, edge.ModFSExpand)
			o.ApplyOverlay = planStepApply(plan, edge.ModOverlay)
			if o.NetConfigure && !o.ApplyLAN && !o.ApplyWiFi && !o.ApplyWAN {
				fmt.Fprintln(os.Stderr, "==> plan: skip network stage (no lan/wan/wifi apply)")
				o.NetConfigure = false
			}
		}
	}

	token := strings.TrimSpace(o.BootstrapToken)
	var mtlsCA, mtlsCert, mtlsKey []byte
	// Local mTLS material (offline / pre-copied from primary).
	if o.MTLSCAFile != "" {
		if b, err := os.ReadFile(o.MTLSCAFile); err == nil {
			mtlsCA = b
		}
	}
	if o.MTLSCertFile != "" {
		if b, err := os.ReadFile(o.MTLSCertFile); err == nil {
			mtlsCert = b
		}
	}
	if o.MTLSKeyFile != "" {
		if b, err := os.ReadFile(o.MTLSKeyFile); err == nil {
			mtlsKey = b
		}
	}
	// Online: pull bootstrap token from primary over SSH :52222.
	if token == "" && o.PrimaryHost != "" && o.PrimaryKey != "" {
		out, err := runSSHOnPort(day2SSHPort(), "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost,
			"cat /etc/netductor/secrets/edge_bootstrap_token 2>/dev/null", o.PrimaryKeyPassphrase)
		if err != nil {
			return fmt.Errorf("read bootstrap token from primary: %w\n%s\n(hint: offline — pass BootstrapToken from a prior pull)", err, out)
		}
		token = strings.TrimSpace(out)
	}
	if token == "" {
		return fmt.Errorf("edge bootstrap token required (SSH to primary or BootstrapToken / --bootstrap-token)")
	}
	// Online: issue client cert on primary unless local mTLS already loaded.
	if len(mtlsCert) == 0 && o.PrimaryHost != "" && o.PrimaryKey != "" {
		remote := fmt.Sprintf(`set -e
netductor mtls ensure >/dev/null 2>&1 || true
netductor mtls issue-client %s >/dev/null 2>&1
CA=/etc/netductor/secrets/mtls/ca.crt
DIR=/etc/netductor/secrets/mtls/clients/%s
if [ ! -f "$DIR/client.crt" ]; then
  DIR=$(find /etc/netductor/secrets/mtls/clients -maxdepth 1 -type d 2>/dev/null | while read d; do
    [ -f "$d/client.crt" ] || continue
    echo "$d"
  done | head -1)
fi
b64() { base64 -w0 "$1" 2>/dev/null || base64 "$1" | tr -d '\n'; }
echo CA:$(b64 "$CA")
echo CERT:$(b64 "$DIR/client.crt")
echo KEY:$(b64 "$DIR/client.key")
`, shellQuote(o.DeviceID), shellQuote(o.DeviceID))
		mout, err := runSSHOnPort(day2SSHPort(), "", o.PrimaryKey, o.PrimaryUser, o.PrimaryHost, remote, o.PrimaryKeyPassphrase)
		if err != nil {
			fmt.Fprintln(os.Stderr, "warn: mTLS material from primary:", err)
			fmt.Fprintln(os.Stderr, mout)
		} else {
			for _, line := range strings.Split(mout, "\n") {
				line = strings.TrimSpace(line)
				switch {
				case strings.HasPrefix(line, "CA:"):
					mtlsCA, _ = base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "CA:"))
				case strings.HasPrefix(line, "CERT:"):
					mtlsCert, _ = base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "CERT:"))
				case strings.HasPrefix(line, "KEY:"):
					mtlsKey, _ = base64.StdEncoding.DecodeString(strings.TrimPrefix(line, "KEY:"))
				}
			}
		}
	}

	// SSH probe first (unless --arch override); Cudy TR1200 = mipsle, not arm64.
	arch, err := ResolveAgentArch(o.AgentArch, o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, "")
	if err != nil {
		return err
	}
	o.AgentArch = arch

	agent, err := EnsureAgentBinary(o.Version, o.AgentArch, o.AgentDir)
	if err != nil {
		return err
	}
	pub := strings.TrimSpace(o.OperatorPubKey)
	if pub == "" && o.PrimaryKey != "" {
		if b, err := os.ReadFile(o.PrimaryKey + ".pub"); err == nil {
			pub = strings.TrimSpace(string(b))
		}
	}
	target := o.RouterUser + "@" + o.RouterHost
	fmt.Fprintln(os.Stderr, "==> edge provision", target, "→", o.ServerURL)
	// Fail-closed: agent plane is mTLS (:8789). Partial or empty material refuses deploy
	// (except dry-run, which may lack live certs).
	haveCA, haveCert, haveKey := len(mtlsCA) > 0, len(mtlsCert) > 0, len(mtlsKey) > 0
	if haveCA || haveCert || haveKey {
		if !(haveCA && haveCert && haveKey) {
			return fmt.Errorf("mTLS material incomplete (ca=%v cert=%v key=%v) — refuse edge deploy", haveCA, haveCert, haveKey)
		}
	}
	if !(haveCA && haveCert && haveKey) {
		if o.DryRun {
			fmt.Fprintln(os.Stderr, "warn: dry-run without mTLS material")
		} else {
			return fmt.Errorf("mTLS material missing (need ca+cert+key for agent plane %s) — refuse edge deploy", mtls.AgentTLSPort)
		}
	} else {
		fmt.Fprintln(os.Stderr, "==> mTLS client material will be installed on router")
	}

	// 1) Agent + config (+ optional new root pass) WITHOUT harden — password still works for network/guest.
	needNet := o.NetConfigure || o.GuestEnable
	if err := edge.Provision(edge.ProvisionOpts{
		SSHTarget:       target,
		DeviceID:        o.DeviceID,
		ServerURL:       o.ServerURL,
		AgentBin:        agent,
		Token:           token,
		Password:        o.RouterPass,
		NewRootPassword: o.NewRootPassword,
		SSHKey:          o.PrimaryKey,
		OperatorPubKey:  pub,
		MTLSCA:          mtlsCA,
		MTLSCert:        mtlsCert,
		MTLSKey:         mtlsKey,
		SkipHarden:      true, // harden after network/guest
	}); err != nil {
		return err
	}
	// After setPass, subsequent SSH must use the NEW root password.
	sshPass := o.RouterPass
	if np := strings.TrimSpace(o.NewRootPassword); np != "" {
		sshPass = np
	}
	o.RouterPass = sshPass

	// 2) Stage network/guest while password auth still enabled.
	if needNet {
		o.Reboot = true
		fmt.Fprintln(os.Stderr, "==> network/guest changes staged; reboot forced at end")
	}
	if o.NetConfigure {
		if err := applyNetworkOnEdge(o); err != nil {
			return fmt.Errorf("network stage: %w", err)
		}
	}
	if o.GuestEnable {
		if err := applyGuestOnEdge(o); err != nil {
			return fmt.Errorf("guest stage: %w", err)
		}
	}
	if o.ApplyExpand {
		if err := applyFSExpandOnEdge(o); err != nil {
			fmt.Fprintln(os.Stderr, "warn: fs expand:", err)
		}
	}

	// Strict: deploy never installs USB packages. Use edge.bootstrap preset usb-storage first.
	skipVPN := false
	st := EdgeStorageOpts{
		UseUSB:               o.ApplyOverlay || (o.Selection.Enabled != nil && o.Selection.Enabled[edge.ModOverlay]),
		Extroot:              o.ApplyOverlay || (o.Selection.Enabled != nil && o.Selection.Enabled[edge.ModOverlay]),
		PinFallback:          true,
		ExpectUSB:            o.ApplyOverlay || (o.Selection.Enabled != nil && o.Selection.Enabled[edge.ModOverlay]),
		DeviceID:             o.DeviceID,
		PrimaryKey:           o.PrimaryKey,
		PrimaryKeyPassphrase: o.PrimaryKeyPassphrase,
	}
	if o.Selection.Enabled != nil && o.Selection.Enabled["usb_format"] {
		st.Format = true
		st.UseUSB = true
	}
	if st.UseUSB || st.Extroot || st.Format {
		facts2, _, _ := ProbeDeviceFacts(o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, o.PrimaryKeyPassphrase, false)
		if !facts2.Storage.HasUsableUSB() {
			need, _, _ := RouterNeedsUSBStorage(o.RouterPass, o.PrimaryKey, o.RouterUser, o.RouterHost, o.PrimaryKeyPassphrase)
			if need {
				return fmt.Errorf("USB/extroot requested but mass-storage modules missing — run edge.bootstrap preset=usb-storage, then probe again")
			}
			return fmt.Errorf("USB/extroot requested but no USB disk visible — plug stick, bootstrap usb-storage if needed, then probe")
		}
		needRB, err := applyEdgeStorage(o, st)
		if err != nil {
			return fmt.Errorf("storage: %w", err)
		}
		if needRB {
			o.Reboot = true
			skipVPN = true
			fmt.Fprintln(os.Stderr, "==> storage: extroot staged — reboot required; vpn_client after reboot when overlay is large")
		}
	}

	if !skipVPN && o.Selection.Enabled != nil && o.Selection.Enabled[edge.ModVPNClient] {
		if err := applyVPNClientOnEdge(o); err != nil {
			return fmt.Errorf("vpn_client: %w", err)
		}
	}
	// Always: no LuCI ASU nag on Status (Close does not save preference).
	if err := applyLuCIDefaultsOnEdge(o); err != nil {
		fmt.Fprintln(os.Stderr, "warn: luci defaults:", err)
	}

	// 3) Safe SSH transition: keys → verify key login → password-off (no daemon reload) → reboot → poll.
	keyOK := false
	if strings.TrimSpace(pub) != "" && o.PrimaryKey != "" {
		fmt.Fprintln(os.Stderr, "==> install operator SSH key (password still enabled)")
		if err := edge.InstallOperatorKeys(edge.ProvisionOpts{
			SSHTarget: target, Password: sshPass, SSHKey: o.PrimaryKey, OperatorPubKey: pub,
		}); err != nil {
			fmt.Fprintln(os.Stderr, "warn: install keys:", err)
		} else {
			fmt.Fprintln(os.Stderr, "==> verify key login before disabling password")
			if _, err := runSSHOnPort(factorySSHPort(), "", o.PrimaryKey, o.RouterUser, o.RouterHost, "echo nd-key-ok", o.PrimaryKeyPassphrase); err != nil {
				fmt.Fprintln(os.Stderr, "warn: key login failed — password SSH kept on:", err)
				fmt.Fprintln(os.Stderr, "hint: check PrimaryKey matches .pub; do not disable password until key works")
			} else {
				keyOK = true
				fmt.Fprintln(os.Stderr, "==> key login OK")
				fmt.Fprintln(os.Stderr, "==> commit PasswordAuth=off (applies after reboot; password still works now)")
				if err := edge.CommitPasswordOff(edge.ProvisionOpts{
					SSHTarget: target, Password: sshPass, SSHKey: o.PrimaryKey, OperatorPubKey: pub,
				}); err != nil {
					fmt.Fprintln(os.Stderr, "warn: password-off commit:", err)
					keyOK = false
				}
			}
		}
	} else {
		fmt.Fprintln(os.Stderr, "warn: no operator pubkey/key — skip harden (password SSH remains)")
	}

	rebootOK := !o.Reboot
	if o.Reboot {
		fmt.Fprintln(os.Stderr, "==> reboot", o.RouterHost)
		if o.NetConfigure && o.LANIP != "" {
			fmt.Fprintln(os.Stderr, "==> after reboot expect LAN", o.LANIP, "(Mac needs route/DHCP on that subnet to reconnect)")
		}
		rebootCmd := "sync; (sleep 1; reboot) >/dev/null 2>&1 &"
		var out string
		var err error
		// Password still live until reboot (we did not reload dropbear).
		if sshPass != "" {
			out, err = runSSHOnPort(factorySSHPort(), sshPass, o.PrimaryKey, o.RouterUser, o.RouterHost, rebootCmd, "")
		}
		if (err != nil || sshPass == "") && o.PrimaryKey != "" {
			out2, err2 := runSSHOnPort(factorySSHPort(), "", o.PrimaryKey, o.RouterUser, o.RouterHost, rebootCmd, "")
			if err2 == nil {
				out, err = out2, nil
			} else if err == nil {
				out, err = out2, err2
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "RESULT: PARTIAL — stages applied, reboot NOT confirmed:", err, strings.TrimSpace(out))
			fmt.Fprintln(os.Stderr, "hint: power-cycle router or: ssh root@"+o.RouterHost+" reboot")
			rebootOK = false
		} else {
			fmt.Fprintln(os.Stderr, "==> reboot issued — waiting for SSH (old IP + staged LAN)")
			hosts := []string{o.RouterHost}
			if lip := strings.TrimSpace(o.LANIP); lip != "" && lip != o.RouterHost {
				hosts = append(hosts, lip)
			}
			waitFor := 180 * time.Second
			if reachable := waitEdgeSSH(hosts, o.PrimaryKey, o.RouterUser, o.PrimaryKeyPassphrase, waitFor); reachable != "" {
				fmt.Fprintln(os.Stderr, "RESULT: OK — SSH reachable at", reachable, "after reboot (key)")
				rebootOK = true
			} else {
				fmt.Fprintln(os.Stderr, "RESULT: PARTIAL — reboot sent; SSH not seen within", waitFor, "on:", strings.Join(hosts, ", "))
				fmt.Fprintln(os.Stderr, "hint: if Mac uses static IP on factory subnet, move Mac to new LAN or use DHCP client on LAN port")
				fmt.Fprintln(os.Stderr, "hint: then: ssh -i <operator_key> root@<lan_ip>")
				// Reachability after reboot is part of deploy success — do not report OK.
				rebootOK = false
			}
		}
	} else {
		fmt.Fprintln(os.Stderr, "RESULT: OK — provision finished (no reboot requested)")
	}
	_ = keyOK
	if o.Reboot && !rebootOK {
		return fmt.Errorf("stages applied but reboot not confirmed — power-cycle the router or SSH and run reboot")
	}
	if skipVPN && (o.Selection.Enabled != nil && o.Selection.Enabled[edge.ModVPNClient]) {
		fmt.Fprintln(os.Stderr, "RESULT: PARTIAL — extroot/storage staged; vpn_client NOT installed this run")
		fmt.Fprintln(os.Stderr, "hint: after reboot verify overlay, then re-run edge deploy (or enable vpn_client) to install sing-box")
		return fmt.Errorf("PARTIAL: reboot/extroot required before vpn_client; re-run deploy after router is back")
	}
	// Control plane: agent must appear on primary (enrolled), not only SSH to the router.
	if !o.DryRun && o.PrimaryHost != "" && o.PrimaryKey != "" {
		fmt.Fprintln(os.Stderr, "==> wait control-plane sighting for", o.DeviceID)
		if err := waitEdgeOnControlPlane(o, 90*time.Second); err != nil {
			return err
		}
	}
	return nil
}

// waitEdgeOnControlPlane polls primary `netductor edge list` until device_id is present
// with status pending or approved (enroll happened). Online health is best-effort.
func waitEdgeOnControlPlane(o EdgeOpts, waitFor time.Duration) error {
	deadline := time.Now().Add(waitFor)
	user := o.PrimaryUser
	if user == "" {
		user = "root"
	}
	did := strings.TrimSpace(o.DeviceID)
	for time.Now().Before(deadline) {
		out, err := runSSHOnPort("52222", "", o.PrimaryKey, user, o.PrimaryHost,
			"netductor edge list 2>/dev/null || true", o.PrimaryKeyPassphrase)
		if err == nil {
			for _, line := range strings.Split(out, "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				var fields []string
				if strings.Contains(line, "\t") {
					fields = strings.Split(line, "\t")
				} else {
					fields = strings.Fields(line)
				}
				if len(fields) < 2 || fields[0] != did {
					continue
				}
				st := fields[1]
				hl := ""
				if len(fields) > 2 {
					hl = fields[2]
				}
				if st == "pending" || st == "approved" {
					fmt.Fprintf(os.Stderr, "RESULT: OK — control plane sees %s status=%s health=%s\n", did, st, hl)
					return nil
				}
			}
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("control plane: device %s not seen on primary within %s (agent enroll/heartbeat) — check mTLS and SERVER URL", did, waitFor)
}
