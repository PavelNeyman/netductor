package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/paths"

	"github.com/PavelNeyman/netductor/internal/install"
	"github.com/PavelNeyman/netductor/internal/stack"
	"github.com/PavelNeyman/netductor/internal/secondary"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	ndver "github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/probes"
)

func lookPath(names ...string) string {
	for _, n := range names {
		if p, err := exec.LookPath(n); err == nil {
			return p
		}
		for _, d := range []string{"/usr/local/bin", "/opt/netductor/bin"} {
			c := filepath.Join(d, n)
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				return c
			}
		}
	}
	return ""
}

func runStatus() {
	for _, u := range []string{"sing-box", "blocky", "netductor-api", "netductor-telegram-bot"} {
		out, _ := exec.Command("systemctl", "is-active", u).Output()
		st := strings.TrimSpace(string(out))
		if st == "" {
			st = "inactive"
		}
		fmt.Printf("  %s: %s\n", u, st)
	}
}

func runBackupCmd(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "peer-set":
			if len(args) < 2 {
				fmt.Fprintln(os.Stderr, "usage: netductor backup peer-set root@host:/var/lib/netductor/backups/peers/core/")
				os.Exit(2)
			}
			opts := ""
			if len(args) > 2 {
				opts = strings.Join(args[2:], " ")
			}
			if err := install.SetBackupPeer(args[1], opts); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(install.BackupPeerStatus())
			return
		case "peer-status", "status":
			fmt.Println(install.BackupPeerStatus())
			return
		case "schedule":
			s := install.LoadBackupSchedule()
			fmt.Println(install.FormatBackupSchedule(s))
			if len(args) >= 4 && (args[1] == "set" || args[1] == "--set") {
				// netductor backup schedule set <hour> <minute> [utc|local]
				h, err1 := strconv.Atoi(args[2])
				m, err2 := strconv.Atoi(args[3])
				if err1 != nil || err2 != nil {
					fmt.Fprintln(os.Stderr, "usage: netductor backup schedule set <hour> <minute> [utc|local]")
					os.Exit(2)
				}
				s.Hour, s.Minute = h, m
				s.UTC = true
				if len(args) >= 5 && (args[4] == "local" || args[4] == "0") {
					s.UTC = false
				}
				if err := install.SaveBackupSchedule(s); err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				fmt.Println("ok", install.FormatBackupSchedule(s))
			}
			return
		case "secondary-local-timer":
			if err := install.InstallSecondaryLocalBackupTimer(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println("secondary local backup timer enabled")
			return
		case "secondary-local", "local-secondary":
			path, err := install.BackupSecondaryLocal()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			fmt.Println(path)
			return
		case "push-recovery":
			url, tok, file := "", "", ""
			for i := 1; i < len(args); i++ {
				switch args[i] {
				case "--url", "-u":
					if i+1 < len(args) {
						i++
						url = args[i]
					}
				case "--token", "-t":
					if i+1 < len(args) {
						i++
						tok = args[i]
					}
				case "--file", "-f":
					if i+1 < len(args) {
						i++
						file = args[i]
					}
				}
			}
			if err := install.PushBackupRecovery(url, tok, file); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		case "push-ssh":
			host, port, key, file := "", "52222", "", ""
			for i := 1; i < len(args); i++ {
				switch args[i] {
				case "--host", "-h":
					if i+1 < len(args) {
						i++
						host = args[i]
					}
				case "--port", "-p":
					if i+1 < len(args) {
						i++
						port = args[i]
					}
				case "--key", "-i":
					if i+1 < len(args) {
						i++
						key = args[i]
					}
				case "--file", "-f":
					if i+1 < len(args) {
						i++
						file = args[i]
					}
				}
			}
			if file == "" {
				// latest local
				dir := filepath.Join(paths.StateDir(), "backups")
				ents, _ := os.ReadDir(dir)
				for _, e := range ents {
					n := e.Name()
					if strings.HasSuffix(n, ".ndenc") {
						file = filepath.Join(dir, n)
					}
				}
			}
			if err := install.PushBackupSSH(host, port, key, file); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			return
		case "now", "run":
			// fallthrough
		case "verify", "list":
			// smoke: latest local archive exists and is non-empty
			dir := filepath.Join(paths.StateDir(), "backups")
			ents, err := os.ReadDir(dir)
			if err != nil {
				fmt.Fprintln(os.Stderr, "no backups dir:", err)
				os.Exit(1)
			}
			var latest string
			var latestSize int64
			for _, e := range ents {
				name := e.Name()
				if !strings.HasSuffix(name, ".ndenc") && !strings.HasSuffix(name, ".tar.gz") {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				if latest == "" || info.ModTime().After(mustStatMod(dir, latest)) {
					latest = name
					latestSize = info.Size()
				}
			}
			if latest == "" || latestSize < 32 {
				fmt.Fprintln(os.Stderr, "FAIL no usable backup archive in", dir)
				os.Exit(1)
			}
			fmt.Printf("OK latest backup %s (%d bytes)\n", filepath.Join(dir, latest), latestSize)
			if args[0] == "list" {
				for _, e := range ents {
					name := e.Name()
					if strings.HasSuffix(name, ".ndenc") || strings.HasSuffix(name, ".tar.gz") {
						info, _ := e.Info()
						if info != nil {
							fmt.Printf("%s\t%d\t%s\n", name, info.Size(), info.ModTime().Format(time.RFC3339))
						}
					}
				}
			}
			return
		}
	}
	path, err := install.Backup()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(path)
}

func runSelfInstall() {
	runUpdate(false)
}



// runUpdate installs from GitHub Releases into /usr/local/bin (FHS).
// usage:
//   netductor update check
//   netductor update list [--limit N]
//   netductor update [apply] [version] [--component node|tg|agent] [--no-restart] [--skip-verify] [--no-backup]
func runUpdate(restart bool) {
	args := os.Args[2:]
	if len(args) > 0 {
		switch args[0] {
		case "check", "status":
			st := ndupdate.CheckStatus(ndver.Release)
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(st)
			if st.Update {
				os.Exit(0)
			}
			return
		case "list", "releases":
			limit := 15
			for i := 1; i < len(args); i++ {
				if (args[i] == "--limit" || args[i] == "-n") && i+1 < len(args) {
					i++
					fmt.Sscanf(args[i], "%d", &limit)
				}
			}
			list, err := ndupdate.ListReleases(limit)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(map[string]any{"local": ndver.Release, "releases": list})
			return
		case "apply":
			args = args[1:]
		case "github-token", "gh-token", "token":
			runUpdateGitHubToken(args[1:])
			return
		case "--help", "-h", "help":
			fmt.Println("netductor update check | list [--limit N]")
			fmt.Println("netductor update [apply] [version] [--component node|tg|agent] [--no-restart] [--skip-verify] [--no-backup]")
			fmt.Println("netductor update github-token status|set <token>|clear")
			fmt.Println("  version empty → latest release; prefer explicit tag if latest is broken")
			return
		}
	}
	comp := "node"
	tag := ""
	doRestart := restart
	noBackup := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--component" && i+1 < len(args):
			i++
			comp = args[i]
		case a == "--no-restart":
			doRestart = false
		case a == "--skip-verify":
			_ = os.Setenv("NETDUCTOR_UPDATE_SKIP_VERIFY", "1")
		case a == "--no-backup":
			noBackup = true
		case a == "--help" || a == "-h":
			fmt.Println("netductor update check | list [--limit N]")
			fmt.Println("netductor update [apply] [version] [--component node|tg|agent] [--no-restart] [--skip-verify] [--no-backup]")
			return
		case strings.HasPrefix(a, "-"):
			// skip unknown flags
		default:
			if tag == "" {
				tag = a
			}
		}
	}
	if !noBackup {
		fmt.Fprintln(os.Stderr, "==> pre-upgrade backup")
		if path, err := install.Backup(); err != nil {
			fmt.Fprintln(os.Stderr, "warn backup:", err)
		} else {
			fmt.Fprintln(os.Stderr, "backup:", path)
			fmt.Fprintln(os.Stderr, "==> wait backup_pull on secondary (up to 15s)")
			acked, pend := install.WaitForBackupPull(15 * time.Second)
			fmt.Fprintf(os.Stderr, "backup_pull: acked=%d pending=%d\n", acked, pend)
		}
	}
	if tag == "" {
		var err error
		tag, err = ndupdate.LatestReleaseTag()
		if err != nil {
			fmt.Fprintln(os.Stderr, "latest:", err)
			os.Exit(1)
		}
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	dest := "/usr/local/bin/netductor"
	unit := "netductor-api"
	switch comp {
	case "tg", "telegram":
		dest = "/usr/local/bin/netductor-tg"
		unit = "netductor-telegram-bot"
	case "agent":
		dest = "/usr/local/bin/netductor-agent"
		unit = "netductor-secondary-agent"
	case "node", "netductor", "":
		comp = "node"
	default:
		fmt.Fprintln(os.Stderr, "unknown component", comp)
		os.Exit(2)
	}
		// Single path for primary node+tg: stack orchestrator (lock, pin, no dual-writer).
	if comp == "node" {
		fmt.Fprintln(os.Stderr, "==> stack apply", tag, "(unified update path)")
		if err := stack.ApplyOpts(tag, noBackup); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	fmt.Fprintln(os.Stderr, "==> update", comp, tag, "→", dest)
	// Download ALL binaries before any restart — never restart TG on old binary.
	if err := ndupdate.DownloadReleaseAsset(tag, comp, dest); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("updated", dest, tag)
	if comp == "node" {
		tgDest := "/usr/local/bin/netductor-tg"
		if err := ndupdate.DownloadReleaseAsset(tag, "tg", tgDest); err != nil {
			fmt.Fprintln(os.Stderr, "tg binary (required with node):", err)
			os.Exit(1)
		}
		fmt.Println("updated", tgDest, tag)
	}
	ndupdate.WriteVERSION(tag)
	if doRestart && unit != "" {
		if unit == "netductor-api" {
			// stop bot first so ETXTBSY is less likely, then install already done
			_ = exec.Command("systemctl", "stop", "netductor-telegram-bot").Run()
		}
		_ = exec.Command("systemctl", "try-restart", unit).Run()
		if unit == "netductor-api" {
			_ = exec.Command("systemctl", "start", "netductor-telegram-bot").Run()
			_ = exec.Command("systemctl", "try-restart", "netductor-redirect").Run()
		} else if unit == "netductor-telegram-bot" {
			_ = exec.Command("systemctl", "restart", "netductor-telegram-bot").Run()
		}
		fmt.Println("restarted", unit)
	}
	// Fan-out binary upgrade to online secondaries (agent cmd "upgrade")
	if comp == "node" {
		n := 0
		for _, d := range secondary.List() {
			if !secondary.Online(d, 2*time.Minute) {
				continue
			}
			if err := secondary.EnqueueCmd(d.ID, "upgrade"); err == nil {
				n++
			}
		}
		if n > 0 {
			fmt.Println("queued secondary upgrade on", n, "device(s)")
		}
	}
}

func runInstall(args []string) {
	comps := []string{}
	for _, a := range args {
		if a == "--help" || a == "-h" {
			fmt.Println("netductor install [--component name ...]   default: core stack")
			fmt.Println("components: dirs hardening singbox blocky vpn-users api metrics telegram backup lampac")
			return
		}
		if a == "--component" || a == "-c" {
			continue
		}
		if strings.HasPrefix(a, "-") {
			continue
		}
		comps = append(comps, a)
	}
	// allow: install singbox blocky
	if err := install.Run(install.Options{Components: comps}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runProbe(args []string) {
	cfg := probes.Load()
	results := probes.Run(cfg)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"probes": results})
	fail := 0
	for _, r := range results {
		if ok, _ := r["ok"].(bool); !ok {
			fail++
		}
	}
	if fail > 0 {
		os.Exit(1)
	}
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func mustStatMod(dir, name string) time.Time {
	fi, err := os.Stat(filepath.Join(dir, name))
	if err != nil {
		return time.Time{}
	}
	return fi.ModTime()
}

func runUpdateGitHubToken(args []string) {
	if len(args) == 0 || args[0] == "status" || args[0] == "check" {
		st := ndupdate.GetTokenStatus()
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(st)
		return
	}
	switch args[0] {
	case "clear", "delete", "rm":
		if err := ndupdate.ClearToken(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("github token cleared")
	case "set":
		tok := ""
		if len(args) > 1 {
			tok = strings.Join(args[1:], " ")
		}
		tok = strings.TrimSpace(tok)
		if tok == "" {
			fmt.Fprintln(os.Stderr, "usage: netductor update github-token set <token>")
			os.Exit(1)
		}
		if err := ndupdate.SetToken(tok); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		st := ndupdate.GetTokenStatus()
		fmt.Println("github token saved:", st.Hint, "source=", st.Source)
	default:
		fmt.Fprintln(os.Stderr, "usage: netductor update github-token status|set <token>|clear")
		os.Exit(1)
	}
}
