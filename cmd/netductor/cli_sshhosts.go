package main

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/netductor/internal/mikrotik"
	"github.com/PavelNeyman/netductor/internal/secondary"
)

func runSSHHosts(args []string) {
	if len(args) < 1 {
		args = []string{"list"}
	}
	kind := "all"
	rest := []string{}
	for i := 0; i < len(args); i++ {
		if args[i] == "--kind" && i+1 < len(args) {
			kind = args[i+1]
			i++
			continue
		}
		rest = append(rest, args[i])
	}
	if len(rest) < 1 {
		rest = []string{"list"}
	}
	switch rest[0] {
	case "list":
		if kind == "all" || kind == "mt" || kind == "mikrotik" {
			fmt.Println("=== mikrotik ===")
			for _, e := range mikrotik.ListKnownHosts() {
				fmt.Printf("%s\t%s\n", e.ID, e.KeyPrefix)
			}
		}
		if kind == "all" || kind == "secondary" {
			fmt.Println("=== secondary ===")
			for _, e := range secondary.ListSSHHosts() {
				fmt.Printf("%s\t%s\n", e.ID, e.KeyPrefix)
			}
		}
	case "forget", "delete", "rm":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor ssh-hosts forget [--kind mt|secondary] <id>")
			os.Exit(2)
		}
		id := rest[1]
		var err error
		switch kind {
		case "secondary":
			err = secondary.ForgetSSHHost(id)
		case "mt", "mikrotik":
			err = mikrotik.ForgetKnownHost(id)
		default:
			e1 := mikrotik.ForgetKnownHost(id)
			e2 := secondary.ForgetSSHHost(id)
			if e1 != nil && e2 != nil {
				err = fmt.Errorf("mt: %v; secondary: %v", e1, e2)
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("ok: forgot", id)
	case "clear":
		switch kind {
		case "secondary":
			_ = secondary.ClearSSHHosts()
		case "mt", "mikrotik":
			_ = mikrotik.ClearKnownHosts()
		default:
			_ = mikrotik.ClearKnownHosts()
			_ = secondary.ClearSSHHosts()
		}
		fmt.Println("ok: cleared", kind)
	case "show", "probe":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor ssh-hosts show <host> [--port 52222]")
			os.Exit(2)
		}
		host, port := rest[1], "52222"
		for i := 2; i < len(rest); i++ {
			if rest[i] == "--port" && i+1 < len(rest) {
				port = rest[i+1]
				i++
			}
		}
		key, err := secondary.ProbeHostKey(host, port)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("host=%s port=%s fingerprint=%s\n", host, port, secondary.FingerprintSHA256(key))
	case "trust":
		if len(rest) < 2 {
			fmt.Fprintln(os.Stderr, "usage: netductor ssh-hosts trust <host> [--port 52222] --fingerprint SHA256:…")
			os.Exit(2)
		}
		host, port, fp := rest[1], "52222", ""
		for i := 2; i < len(rest); i++ {
			if rest[i] == "--port" && i+1 < len(rest) {
				port = rest[i+1]
				i++
				continue
			}
			if rest[i] == "--fingerprint" && i+1 < len(rest) {
				fp = rest[i+1]
				i++
			}
		}
		if fp == "" {
			fmt.Fprintln(os.Stderr, "refuse: --fingerprint required")
			os.Exit(2)
		}
		key, err := secondary.TrustHostKeyIfFingerprintMatch(host, port, fp)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("ok: trusted %s fingerprint=%s\n", host, secondary.FingerprintSHA256(key))
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor ssh-hosts list|show|trust|forget|clear [--kind mt|secondary]")
		os.Exit(2)
	}
}
