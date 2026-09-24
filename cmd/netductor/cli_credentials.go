package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/PavelNeyman/netductor/internal/deploy"
)

func runCredentials(args []string) {
	if len(args) < 1 || args[0] == "help" || args[0] == "-h" {
		fmt.Fprint(os.Stderr, `usage:
  netductor credentials collect --host IP [--user root] [--key PATH] [--role primary|secondary]

Pull BACKUP_KEY / RECOVERY_TOKEN over SSH and write ~/.netductor/credentials/*.txt
`)
		os.Exit(2)
	}
	if args[0] != "collect" {
		fmt.Fprintln(os.Stderr, "unknown credentials subcommand")
		os.Exit(2)
	}
	host, user, key, role, pass := "", "root", "", "primary", ""
	for i := 1; i < len(args); i++ {
		a := args[i]
		next := func() string {
			if i+1 < len(args) {
				i++
				return args[i]
			}
			return ""
		}
		switch a {
		case "--host", "-h":
			host = next()
		case "--user", "-u":
			user = next()
		case "--key", "-i":
			key = next()
		case "--role":
			role = next()
		case "--passphrase":
			pass = next()
		}
	}
	if host == "" {
		fmt.Fprintln(os.Stderr, "--host required")
		os.Exit(2)
	}
	if key == "" {
		key = os.Getenv("NETDUCTOR_SSH_KEY")
	}
	if key == "" {
		home, _ := os.UserHomeDir()
		for _, c := range []string{
			home + "/.ssh/netductor_primary",
			home + "/.ssh/id_ed25519",
		} {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				key = c
				break
			}
		}
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "--key or NETDUCTOR_SSH_KEY required")
		os.Exit(2)
	}
	if os.Getenv("NETDUCTOR_SSH_PORT") == "" {
		_ = os.Setenv("NETDUCTOR_SSH_PORT", "52222")
	}
	role = strings.TrimSpace(role)
	path, err := deploy.CollectOperatorSecrets(role, user, host, key, pass)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(path)
}
