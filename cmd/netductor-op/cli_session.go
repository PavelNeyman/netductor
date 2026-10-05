package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func runSession(args []string) {
	if len(args) < 1 || args[0] == "help" {
		fmt.Fprint(os.Stderr, `usage:
  netductor-op session issue --host IP [--key PATH] [--hours 72]

SSH to node and POST /api/session/issue on loopback (R7b).
Prints token (store in WebUI Settings → Node session).
`)
		os.Exit(2)
	}
	if args[0] != "issue" {
		fmt.Fprintln(os.Stderr, "unknown:", args[0])
		os.Exit(2)
	}
	host, key, hours := "", "", "72"
	user := "root"
	sshPort := strings.TrimSpace(os.Getenv("NETDUCTOR_SSH_PORT"))
	if sshPort == "" {
		sshPort = "52222"
	}
	for i := 1; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--host" && i+1 < len(args):
			i++
			host = args[i]
		case a == "--key" && i+1 < len(args):
			i++
			key = args[i]
		case a == "--user" && i+1 < len(args):
			i++
			user = args[i]
		case a == "--hours" && i+1 < len(args):
			i++
			hours = args[i]
		}
	}
	if host == "" {
		fmt.Fprintln(os.Stderr, "required: --host")
		os.Exit(2)
	}
	// F19/R7b: hours digits only, clamp 1–168
	h := 72
	if n, err := strconv.Atoi(strings.TrimSpace(hours)); err == nil && n >= 1 && n <= 168 {
		h = n
	}
	if key == "" {
		home, _ := os.UserHomeDir()
		key = filepath.Join(home, ".ssh", "netductor_primary")
	}
	if strings.HasPrefix(key, "~/") {
		home, _ := os.UserHomeDir()
		key = filepath.Join(home, key[2:])
	}
	// Fixed remote command; hours only as JSON number (no shell interpolation of free text)
	remote := fmt.Sprintf(
		`curl -fsS -X POST http://127.0.0.1:8787/api/session/issue -H 'Content-Type: application/json' -d '{"hours":%d,"label":"op-issue"}'`,
		h,
	)
	cmd := exec.Command("ssh", "-p", sshPort, "-i", key, "-o", "BatchMode=yes", "--", user+"@"+host, remote)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			fmt.Fprintln(os.Stderr, string(ee.Stderr))
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	tok := extractTokenJSON(string(out))
	if tok == "" {
		fmt.Fprintln(os.Stderr, "no token in response:", strings.TrimSpace(string(out)))
		os.Exit(1)
	}
	fmt.Println(tok)
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".netductor")
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "node_session"), []byte(tok+"\n"), 0o600)
	fmt.Fprintln(os.Stderr, "saved ~/.netductor/node_session — paste into WebUI Settings or use as Bearer")
}

func extractTokenJSON(s string) string {
	s = strings.TrimSpace(s)
	// minimal parse: "token":"..."
	const key = `"token"`
	i := strings.Index(s, key)
	if i < 0 {
		// plain token fallback (legacy CLI)
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
		if len(s) >= 32 && !strings.Contains(s, " ") {
			return s
		}
		return ""
	}
	rest := s[i+len(key):]
	j := strings.Index(rest, `"`)
	if j < 0 {
		return ""
	}
	rest = rest[j+1:]
	k := strings.Index(rest, `"`)
	if k < 0 {
		return ""
	}
	return rest[:k]
}
