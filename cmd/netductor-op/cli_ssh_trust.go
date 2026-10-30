package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/secondary"
)

type sshTrustArgs struct {
	host, port, fingerprint string
	showOnly                bool
}

func parseSSHTrustArgs(args []string) (sshTrustArgs, error) {
	out := sshTrustArgs{port: "52222"}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--show":
			out.showOnly = true
		case a == "--port" && i+1 < len(args):
			i++
			out.port = strings.TrimSpace(args[i])
			if out.port == "" {
				return out, fmt.Errorf("empty --port")
			}
		case a == "--fingerprint" && i+1 < len(args):
			i++
			out.fingerprint = strings.TrimSpace(args[i])
		case a == "--port" || a == "--fingerprint":
			return out, fmt.Errorf("%s requires a value", a)
		case strings.HasPrefix(a, "-"):
			return out, fmt.Errorf("unknown flag %s", a)
		default:
			if out.host == "" {
				out.host = a
			} else {
				return out, fmt.Errorf("unexpected argument %q", a)
			}
		}
	}
	if out.host == "" {
		return out, fmt.Errorf("host required")
	}
	if out.showOnly && out.fingerprint != "" {
		return out, fmt.Errorf("use either --show or --fingerprint, not both")
	}
	if !out.showOnly && out.fingerprint == "" {
		return out, fmt.Errorf("--fingerprint required to pin (or --show to print probe fingerprint only)")
	}
	return out, nil
}

// runSSHTrust: explicit host-key bootstrap for operator Mac (Primary/Secondary endpoints).
// Never accepts a key without --fingerprint match (or --show only).
func runSSHTrust(args []string) int {
	ta, err := parseSSHTrustArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "usage: netductor-op ssh-trust <host> [--port 52222] [--show | --fingerprint SHA256:…]")
		fmt.Fprintln(os.Stderr, "error:", err)
		return 2
	}
	if ta.showOnly {
		key, err := secondary.ProbeHostKey(ta.host, ta.port)
		if err != nil {
			fmt.Fprintln(os.Stderr, "probe:", err)
			return 1
		}
		fmt.Printf("host=%s port=%s\nfingerprint=%s\n", ta.host, ta.port, secondary.FingerprintSHA256(key))
		fmt.Fprintln(os.Stderr, "note: probe is not authentication — confirm fingerprint out-of-band, then re-run with --fingerprint")
		return 0
	}
	key, err := secondary.TrustHostKeyIfFingerprintMatch(ta.host, ta.port, ta.fingerprint)
	if err != nil {
		fmt.Fprintln(os.Stderr, "trust:", err)
		return 1
	}
	fmt.Printf("ok: trusted %s port=%s fingerprint=%s\n", ta.host, ta.port, secondary.FingerprintSHA256(key))
	return 0
}
