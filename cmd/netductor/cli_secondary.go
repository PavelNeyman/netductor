package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/cli18n"

	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/mtls"
	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/paths"
	"github.com/PavelNeyman/netductor/internal/secondary"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func runSecondary(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, cli18n.T("secondary.usage"))
		os.Exit(2)
	}
	switch args[0] {
	case "export":
		out := "bundle.json"
		sni := vpn.DefaultRealitySNI
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "-o", "--output":
				if i+1 < len(args) {
					i++
					out = args[i]
				}
			case "--sni":
				if i+1 < len(args) {
					i++
					sni = args[i]
				}
			}
		}
		b, err := vpn.ExportSecondaryBundle(sni)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if id, tok, err := secondary.IssueToken("secondary"); err == nil {
			b.AgentID = id
			b.AgentToken = tok
			b.CoreAgentURL = "https://" + b.CoreIP + ":" + mtls.AgentTLSPort
		}
		raw, _ := json.MarshalIndent(b, "", "  ")
		if err := os.WriteFile(out, append(raw, '\n'), 0o600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("wrote", out)
		fmt.Println("Copy to RU VPS and run: netductor secondary join", out)
		fmt.Println("uplink user:", vpn.RelayUplinkName, "uuid="+b.UplinkUUID)
	case "join":
		path := "bundle.json"
		if len(args) > 1 {
			path = args[1]
		}
		if err := install.InstallRelay(path); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "links":
		dir := filepath.Join(paths.SecondaryDir(), "clients")
		ents, err := os.ReadDir(dir)
		if err != nil {
			fmt.Fprintln(os.Stderr, "no secondary client links — run join on RU VPS first")
			os.Exit(1)
		}
		for _, e := range ents {
			if e.IsDir() {
				continue
			}
			b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
			fmt.Printf("## %s\n%s\n", e.Name(), string(b))
		}
	case "prune":
		n := secondary.PruneDuplicates()
		s := secondary.PruneStale(10 * time.Minute)
		fmt.Printf("duplicates_removed=%d stale_removed=%d\n", n, s)
		// drop offline secondary nodes from registry
		for _, d := range secondary.List() {
			_ = d
		}
		nodes.MarkStaleRelays(180)
		if list, err := nodes.List(); err == nil {
			for _, n := range list {
				if n.Role == "secondary" && n.Status == "offline" && n.PublicIP == "" {
					_ = nodes.Delete(n.ID)
				}
			}
		}
	case "remove":
		if len(args) < 2 {
			os.Exit(2)
		}
		if err := secondary.RemoveDevice(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = nodes.Delete(args[1])
		fmt.Println("removed", args[1])
	case "agent":
		tok := paths.ReadSecret("secondary_agent_token")
		url := paths.ReadSecret("secondary_core_url")
		if len(args) > 1 && args[1] != "" {
			// optional overrides
		}
		if tok == "" || url == "" {
			fmt.Fprintln(os.Stderr, "missing secondary_agent_token or secondary_core_url")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "secondary agent →", url)
		secondary.AgentLoop(url, tok, 30*time.Second)
	case "provision":
		host, user, pass, sni, opPub, sshKey := "", "root", "", "", "", ""
		port := 22
		for i := 1; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--host" && i+1 < len(args):
				i++
				host = args[i]
			case a == "--user" && i+1 < len(args):
				i++
				user = args[i]
			case a == "--password" && i+1 < len(args):
				i++
				pass = args[i]
			case a == "--port" && i+1 < len(args):
				i++
				fmt.Sscanf(args[i], "%d", &port)
			case a == "--sni" && i+1 < len(args):
				i++
				sni = args[i]
			case a == "--operator-pubkey" && i+1 < len(args):
				i++
				opPub = args[i]
			case a == "--ssh-key" && i+1 < len(args):
				i++
				sshKey = args[i]
			}
		}
		if pass == "" {
			pass = os.Getenv("NETDUCTOR_SSH_PASSWORD")
		}
		if sshKey == "" {
			sshKey = os.Getenv("NETDUCTOR_SSH_KEY")
		}
		if host == "" || (pass == "" && sshKey == "") {
			fmt.Fprintln(os.Stderr, "required: --host and (--password or --ssh-key / NETDUCTOR_SSH_*)")
			os.Exit(2)
		}
		// Reinstall always changes SSH host key — clear TOFU + OpenSSH known_hosts before dial.
		_ = secondary.ForgetSSHHost(host)
		_ = execLocal("ssh-keygen", "-f", "/root/.ssh/known_hosts", "-R", host)
		sni = vpn.ResolveSecondarySNI(sni, host)
		fmt.Fprintln(os.Stderr, "provision SNI:", sni)
		// Pre-clean: same IP must not keep dead agents from previous install.
		_ = secondary.RemoveByPublicIP(host, "")
		b, err := vpn.ExportSecondaryBundle(sni)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = mtls.EnsureAll(os.Getenv("NETDUCTOR_PUBLIC_IP"))
		var mtlsCA, mtlsCert, mtlsKey []byte
		if id, tok, err := secondary.IssueToken("secondary"); err == nil {
			b.AgentID = id
			b.AgentToken = tok
			b.CoreAgentURL = "https://" + b.CoreIP + ":" + mtls.AgentTLSPort
			if ca, cert, key, err := mtls.EnsureClientFor(id); err != nil {
				fmt.Fprintln(os.Stderr, "mtls EnsureClientFor:", err)
			} else {
				mtlsCA, mtlsCert, mtlsKey = ca, cert, key
			}
		}
		raw, _ := json.MarshalIndent(b, "", "  ")
		res, err := secondary.ProvisionFromCore(secondary.ProvisionIn{
			Host: host, Port: port, User: user, Password: pass, SSHPrivateKey: sshKey, SNI: sni, OperatorPubKey: opPub,
			MTLSCA: mtlsCA, MTLSCert: mtlsCert, MTLSKey: mtlsKey,
		}, string(raw))
		if res != nil {
			fmt.Println(res.Log)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		postProvisionSecondary(host, sni)
		fmt.Println("provisioned", host)
	case "prepare-pack":
		// Control-plane only: issue token + mTLS + VPN bundle. No SSH to secondary.
		// Operator machine applies the pack via `deploy secondary` (Mac → secondary SSH).
		sni := ""
		name := "secondary"
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--sni":
				if i+1 < len(args) {
					i++
					sni = args[i]
				}
			case "--name":
				if i+1 < len(args) {
					i++
					name = args[i]
				}
			}
		}
		sni = vpn.ResolveSecondarySNI(sni, "")
		_ = mtls.EnsureAll(os.Getenv("NETDUCTOR_PUBLIC_IP"))
		b, err := vpn.ExportSecondaryBundle(sni)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		id, tok, err := secondary.IssueToken(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, "IssueToken:", err)
			os.Exit(1)
		}
		b.AgentID = id
		b.AgentToken = tok
		b.CoreAgentURL = "https://" + b.CoreIP + ":" + mtls.AgentTLSPort
		var mtlsCA, mtlsCert, mtlsKey []byte
		if ca, cert, key, err := mtls.EnsureClientFor(id); err != nil {
			fmt.Fprintln(os.Stderr, "mtls EnsureClientFor:", err)
			os.Exit(1)
		} else {
			mtlsCA, mtlsCert, mtlsKey = ca, cert, key
		}
		pack := map[string]any{
			"bundle":       b,
			"agent_id":     id,
			"agent_token":  tok,
			"core_url":     b.CoreAgentURL,
			"mtls_ca_b64":  base64.StdEncoding.EncodeToString(mtlsCA),
			"mtls_cert_b64": base64.StdEncoding.EncodeToString(mtlsCert),
			"mtls_key_b64": base64.StdEncoding.EncodeToString(mtlsKey),
			"sni":          sni,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(pack)

	case "device":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, cli18n.T("secondary.usage"))
			os.Exit(2)
		}
		for _, d := range secondary.List() {
			if d.ID != args[1] {
				continue
			}
			on := "offline"
			if secondary.Online(d, 2*time.Minute) {
				on = "online"
			}
			fmt.Println("status:", on)
			fmt.Println("sb=", d.SingBoxOK, "cpu=", d.CPUPercent, "mem=", d.MemUsedMB, "/", d.MemTotalMB, "load=", d.Load1)
			if len(d.PendingCmds) > 0 {
				fmt.Println("pending:", d.PendingCmds)
			}
			if d.LastCmd != "" {
				ok := "fail"
				if d.LastCmdOK {
					ok = "ok"
				}
				fmt.Println("last_cmd:", d.LastCmd, ok, d.LastCmdAt.Format(time.RFC3339))
				if d.LastCmdLog != "" {
					log := d.LastCmdLog
					if len(log) > 1500 {
						log = log[len(log)-1500:]
					}
					fmt.Println(log)
				}
			}
			return
		}
		fmt.Println("not found")
	case "cmd":
		if len(args) < 3 {
			fmt.Fprintln(os.Stderr, cli18n.T("secondary.usage"))
			os.Exit(2)
		}
		if err := secondary.EnqueueCmd(args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("queued", args[2], "for", args[1])
	case "sync":
		ver := secondary.BumpConfigVer()
		fmt.Println("config_ver", ver)
		fmt.Println(cli18n.T("secondary.pull_hint"))
	case "exit":
		if len(args) < 2 || args[1] == "status" {
			fmt.Println("exit_enabled", secondary.ExitEnabled())
			return
		}
		on := args[1] == "on" || args[1] == "1" || args[1] == "true"
		if args[1] == "off" || args[1] == "0" || args[1] == "false" {
			on = false
		} else if args[1] != "on" && args[1] != "1" && args[1] != "true" {
			fmt.Fprintln(os.Stderr, cli18n.T("secondary.usage"))
			os.Exit(2)
		}
		if err := secondary.SetExitEnabled(on); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		_ = vpn.ApplyConfig()
		fmt.Println("exit_enabled", on)
		fmt.Println("sing-box re-applied on core; relays will sync on next agent poll")
	case "status":
		_ = secondary.PruneDuplicates()
		devs := secondary.List()
		if len(devs) == 0 {
			b, err := os.ReadFile(filepath.Join(paths.SecondaryDir(), "bundle.json"))
			if err != nil {
				fmt.Println("no relays registered")
				return
			}
			fmt.Println(string(b))
			return
		}
		for _, d := range devs {
			on := "offline"
			if secondary.Online(d, 2*time.Minute) {
				on = "online"
			}
			fmt.Printf("%s	%s	%s	ip=%s	sb=%v	ver=%d\n", d.ID, d.Name, on, d.PublicIP, d.SingBoxOK, d.ConfigVer)
		}
		fmt.Println("config_ver", secondary.ConfigVer())
	default:
		fmt.Fprintln(os.Stderr, "unknown secondary subcommand")
		os.Exit(2)
	}
}


// sshIdentityArgs: prefer operator_reprovision (deploy left key for post-steps), else default identity.
func sshIdentityArgs() []string {
	// Prefer core's own key for rare primary→secondary post-steps — never Mac private key.
	for _, p := range []string{"/root/.ssh/id_ed25519", "/root/.ssh/id_rsa"} {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return []string{"-i", p}
		}
	}
	return nil
}

func scpToSecondary(local, remoteHostPath string) error {
	args := []string{"-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes"}
	args = append(args, sshIdentityArgs()...)
	args = append(args, local, remoteHostPath)
	return execLocal("scp", args...)
}

func sshOnSecondary(host, cmd string) error {
	args := []string{"-o", "StrictHostKeyChecking=accept-new", "-o", "BatchMode=yes"}
	args = append(args, sshIdentityArgs()...)
	args = append(args, "root@"+host, cmd)
	c := exec.Command("ssh", args...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}

// postProvisionSecondary restores operator-facing state after a clean relay reinstall.
// Reality keys are always new on the secondary; everything else is rebuilt on core + remote.
func postProvisionSecondary(host, sni string) {
	fmt.Println("==> post-provision: forget old SSH host key (reinstall changes fingerprint)")
	_ = execLocal("ssh-keygen", "-f", "/root/.ssh/known_hosts", "-R", host)

	fmt.Println("==> post-provision: mTLS already installed in provision SSH session (no primary→secondary SSH)")
	// Optional later refresh uses agent plane: EnqueueCmd(id, "mtls_refresh") after heartbeat.

	fmt.Println("==> post-provision: keep issued agent tokens (do not wipe before heartbeat)")
	// Do not RemoveByPublicIP(host,"") here — that deleted the token IssueToken just wrote.
	_ = secondary.PruneStale(24 * time.Hour)
	if list, err := nodes.List(); err == nil {
		for _, n := range list {
			if n.Role != "secondary" {
				continue
			}
			if n.PublicIP == host || n.PublicIP == "" {
				_ = nodes.Delete(n.ID)
				fmt.Println("  nodes: removed", n.ID)
			}
		}
	}

	fmt.Println("==> post-provision: wait for agent heartbeat (pbk/sid)…")
	dev := secondary.WaitOnlinePBK(host, 90*time.Second)
	if dev == nil {
		fmt.Println("  warn: no heartbeat with PBK within 90s — links may lag until agent checks in")
	} else {
		fmt.Printf("  online id=%s sni=%s pbk=%s…\n", dev.ID, dev.SNI, trimPBK(dev.PBK))
		// Agent-plane path (HTTPS mTLS): optional cert re-pull; not required if provision wrote material.
		if err := secondary.EnqueueCmd(dev.ID, "mtls_refresh"); err != nil {
			fmt.Println("  mtls_refresh enqueue:", err)
		} else {
			fmt.Println("  queued mtls_refresh via agent (HTTPS), not SSH")
		}
		// keep only this device for the IP
		_ = secondary.RemoveByPublicIP(host, dev.ID)
		_, _ = nodes.SetDesiredHostname(dev.ID, "nd-secondary")
		_ = secondary.Rename(dev.ID, "nd-secondary")
	}

	fmt.Println("==> post-provision: re-apply core sing-box (uplink / routing)")
	if err := vpn.ApplyConfig(); err != nil {
		fmt.Println("  apply:", err)
	} else {
		fmt.Println("  apply: ok")
		_ = execLocal("systemctl", "try-restart", "sing-box")
	}

	fmt.Println("==> post-provision: refresh all client links (new secondary keys)")
	if n, err := vpn.RefreshLinks(""); err != nil {
		fmt.Println("  refresh-links:", err)
	} else {
		fmt.Println("  refresh-links: refreshed", n)
	}

	fmt.Println("==> post-provision: local backup only (no primary→secondary SSH/scp)")
	// Offsite mirror to secondary is not done over SSH. Model: agent pulls from primary
	// over mTLS HTTPS later (backup_pull cmd) or operator seeds from Mac once.
	// SSH offsite peer disabled — primary keeps local backups only until agent pull exists
	if path, err := install.Backup(); err != nil {
		fmt.Println("  backup now:", err)
	} else {
		fmt.Println("  backup (primary local):", path)
	}

	fmt.Println("==> post-provision: remember SNI", sni)
	_ = vpn.RememberSecondarySNI(sni)
	_ = os.WriteFile(filepath.Join(paths.EtcDir(), "secrets", "last_secondary_host"), []byte(host+"\n"), 0o600)

	// final prune pass
	_ = secondary.PruneDuplicates()
	fmt.Println("==> post-provision: done — secondary VPN plane ready (new Reality keys only); run: netductor fleet provision-secondary extras or fleet bootstrap")
}

func trimPBK(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:8] + "…"
}

func pushLatestBackup(target string) {
	dir := filepath.Join(paths.StateDir(), "backups")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	var latest string
	var latestMod time.Time
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".ndenc") && !strings.HasSuffix(name, ".tar.gz") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if latest == "" || info.ModTime().After(latestMod) {
			latest = filepath.Join(dir, name)
			latestMod = info.ModTime()
		}
	}
	if latest != "" {
		_ = scpToSecondary(latest, target)
	}
}

func execSSHHost(host, cmd string) error {
	return sshOnSecondary(host, cmd)
}

func execLocal(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}
