package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/PavelNeyman/netductor/internal/svcpaths"
)

func runSvcPaths(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, `usage:
  netductor svc-paths status|health|apply
  netductor svc-paths bootstrap-primary --peer-ip IP
  netductor svc-paths bootstrap-secondary --material /path/or/stdin
  netductor svc-paths failover status|tick
  netductor svc-paths failover enable|disable
  netductor svc-paths failover set-users-sp on|off
`)
		os.Exit(2)
	}
	switch args[0] {
	case "status", "health":
		r := svcpaths.Status()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(r)
		if !r.SP.Up && !r.PS.Up {
			os.Exit(1)
		}
	case "apply":
		msg, err := svcpaths.Apply()
		if msg != "" {
			fmt.Println(msg)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		r := svcpaths.Status()
		fmt.Printf("sp_up=%v ps_up=%v role=%s\n", r.SP.Up, r.PS.Up, r.HostRole)
	case "bootstrap-primary":
		peer := ""
		forceNew := false
		for i := 1; i < len(args); i++ {
			if args[i] == "--peer-ip" && i+1 < len(args) {
				peer = args[i+1]
				i++
			}
			if args[i] == "--force-new-material" {
				forceNew = true
			}
		}
		if peer == "" {
			fmt.Fprintln(os.Stderr, "need --peer-ip secondary-public-ip")
			os.Exit(2)
		}
		prim := strings.TrimSpace(os.Getenv("NETDUCTOR_PUBLIC_IP"))
		if prim == "" {
			prim = "0.0.0.0"
		}
		if b, err := os.ReadFile("/etc/netductor/secrets/public_ip"); err == nil {
			if p := strings.TrimSpace(string(b)); p != "" {
				prim = p
			}
		}
		m, generated, err := svcpaths.LoadOrGenerateMaterial(prim, peer, forceNew)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		m.Primary = prim
		m.Secondary = peer
		if generated {
			fmt.Fprintln(os.Stderr, "svc-paths: generated new WG material for peer", peer)
		} else {
			fmt.Fprintln(os.Stderr, "svc-paths: reusing existing material for peer", peer)
		}
		if err := svcpaths.BootstrapPrimary(m); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		// R2: stdout is what deploy SCPs to secondary — never emit primary private keys.
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(m.ForSecondary())
	case "bootstrap-secondary":
		path := ""
		for i := 1; i < len(args); i++ {
			if args[i] == "--material" && i+1 < len(args) {
				path = args[i+1]
			}
		}
		var m svcpaths.Material
		if path != "" && path != "-" {
			b, err := os.ReadFile(path)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err := json.Unmarshal(b, &m); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		} else {
			if err := json.NewDecoder(os.Stdin).Decode(&m); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		if err := svcpaths.BootstrapSecondary(&m); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("svc-paths secondary OK")
	case "peer":
		runSvcPeer(args[1:])
	case "failover":
		runSvcFailover(args[1:])
	case "bootstrap-primary-build":
		path := flagMaterial(args)
		if path == "" {
			fmt.Fprintln(os.Stderr, "need --material")
			os.Exit(2)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var m svcpaths.BuildMaterial
		if err := json.Unmarshal(b, &m); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := svcpaths.BootstrapPrimaryBuild(&m); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("svc-paths primary-build OK")
	case "bootstrap-build":
		path := flagMaterial(args)
		if path == "" {
			fmt.Fprintln(os.Stderr, "need --material")
			os.Exit(2)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		var m svcpaths.BuildMaterial
		if err := json.Unmarshal(b, &m); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := svcpaths.BootstrapBuildHost(&m); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("svc-paths build-host OK")
	default:
		fmt.Fprintln(os.Stderr, "unknown svc-paths subcommand")
		os.Exit(2)
	}
}

func runSvcFailover(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor svc-paths failover status|tick|enable|disable|set-users-sp on|off")
		os.Exit(2)
	}
	switch strings.ToLower(args[0]) {
	case "status":
		p := svcpaths.LoadPolicy()
		s := svcpaths.LoadState()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(map[string]any{"policy": p, "state": s})
	case "tick":
		s, sum, err := svcpaths.RunSecondaryCycle()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(sum)
		b, _ := json.MarshalIndent(s, "", "  ")
		fmt.Println(string(b))
	case "enable":
		p := svcpaths.LoadPolicy()
		p.Enabled = true
		_ = svcpaths.SavePolicy(p)
		fmt.Println("failover enabled")
	case "disable":
		p := svcpaths.LoadPolicy()
		p.Enabled = false
		_ = svcpaths.SavePolicy(p)
		fmt.Println("failover disabled")
	case "set-users-sp":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "need on|off")
			os.Exit(2)
		}
		on := args[1] == "on" || args[1] == "1" || args[1] == "true"
		p := svcpaths.LoadPolicy()
		p.UsersToSPOnVLESSDown = on
		_ = svcpaths.SavePolicy(p)
		fmt.Printf("users_to_sp_on_vless_down=%v\n", on)
	default:
		if args[0] == "fail-threshold" && len(args) > 1 {
			n, _ := strconv.Atoi(args[1])
			p := svcpaths.LoadPolicy()
			if n > 0 {
				p.FailThreshold = n
			}
			_ = svcpaths.SavePolicy(p)
			fmt.Printf("fail_threshold=%d\n", p.FailThreshold)
			return
		}
		fmt.Fprintln(os.Stderr, "unknown failover subcommand")
		os.Exit(2)
	}
}

func flagMaterial(args []string) string {
	for i := 1; i < len(args); i++ {
		if args[i] == "--material" && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func runSvcPeer(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: netductor svc-paths peer list|add|ready|promote|rollback|remove")
		os.Exit(2)
	}
	switch strings.ToLower(args[0]) {
	case "list":
		idx, err := svcpaths.PeerList()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(idx)
	case "add":
		peerIP, peerID, prim := "", "", ""
		for i := 1; i < len(args); i++ {
			switch args[i] {
			case "--peer-ip":
				if i+1 < len(args) {
					peerIP, i = args[i+1], i+1
				}
			case "--id":
				if i+1 < len(args) {
					peerID, i = args[i+1], i+1
				}
			case "--primary-ip":
				if i+1 < len(args) {
					prim, i = args[i+1], i+1
				}
			}
		}
		if peerIP == "" {
			fmt.Fprintln(os.Stderr, "need --peer-ip")
			os.Exit(2)
		}
		if prim == "" {
			if b, err := os.ReadFile("/etc/netductor/secrets/public_ip"); err == nil {
				prim = strings.TrimSpace(string(b))
			}
		}
		mat, peer, err := svcpaths.PeerAdd(svcpaths.PeerAddOpts{PeerID: peerID, PublicIP: peerIP, PrimaryIP: prim})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "peer %s prepared slot=%d ip=%s (not promoted)\n", peer.ID, peer.Slot.Index, peer.PublicIP)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(mat)
	case "validate":
		id := ""
		if len(args) >= 2 {
			id = args[1]
		}
		vr, err := svcpaths.PeerValidateLocal(id)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(vr)
		if !vr.OK {
			os.Exit(1)
		}
	case "ready":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "need peer id")
			os.Exit(2)
		}
		// Default: require validate. --force-ready skips checks (ops escape hatch).
		req := true
		for _, a := range args[2:] {
			if a == "--force-ready" {
				req = false
			}
		}
		if err := svcpaths.PeerMarkReady(args[1], req); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ready", args[1])
	case "promote":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "need peer id")
			os.Exit(2)
		}
		if err := svcpaths.PeerPromote(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("promoted", args[1])
	case "rollback":
		to := ""
		if len(args) >= 2 {
			to = args[1]
		}
		if err := svcpaths.PeerRollback(to); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("rollback ok")
	case "remove":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "need peer id")
			os.Exit(2)
		}
		if err := svcpaths.PeerRemove(args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("removed", args[1])
	case "recover":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "need peer id")
			os.Exit(2)
		}
		msg, err := svcpaths.RecoverIncompletePeerAdd(args[1])
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println(msg)
	default:
		fmt.Fprintln(os.Stderr, "unknown peer subcommand")
		os.Exit(2)
	}
}
