package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

// runSSHTrust: explicit host-key bootstrap for operator Mac (Primary/Secondary endpoints).
// Never accepts a key without --fingerprint match (or --show only).
func runSSHTrust(args []string) int {
	host, port := "", "52222"
	fp, showOnly := "", false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--show":
			showOnly = true
		case a == "--port" && i+1 < len(args):
			i++
			port = strings.TrimSpace(args[i])
		case a == "--fingerprint" && i+1 < len(args):
			i++
			fp = strings.TrimSpace(args[i])
		case strings.HasPrefix(a, "-"):
			fmt.Fprintln(os.Stderr, "unknown flag", a)
			return 2
		default:
			if host == "" {
				host = a
			} else {
				fmt.Fprintln(os.Stderr, "usage: netductor-op ssh-trust <host> [--port 52222] [--show | --fingerprint SHA256:…]")
				return 2
			}
		}
	}
	if host == "" {
		fmt.Fprintln(os.Stderr, "usage: netductor-op ssh-trust <host> [--port 52222] [--show | --fingerprint SHA256:…]")
		return 2
	}
	if showOnly {
		key, err := secondary.ProbeHostKey(host, port)
		if err != nil {
			fmt.Fprintln(os.Stderr, "probe:", err)
			return 1
		}
		fmt.Printf("host=%s port=%s\nfingerprint=%s\n", host, port, secondary.FingerprintSHA256(key))
		fmt.Fprintln(os.Stderr, "note: probe is not authentication — confirm fingerprint out-of-band, then re-run with --fingerprint")
		return 0
	}
	if fp == "" {
		fmt.Fprintln(os.Stderr, "refuse: --fingerprint required to pin (or --show to print probe fingerprint only)")
		return 2
	}
	key, err := secondary.TrustHostKeyIfFingerprintMatch(host, port, fp)
	if err != nil {
		fmt.Fprintln(os.Stderr, "trust:", err)
		return 1
	}
	fmt.Printf("ok: trusted %s port=%s fingerprint=%s\n", host, port, secondary.FingerprintSHA256(key))
	return 0
}
