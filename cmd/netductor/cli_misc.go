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
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
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
// usage: netductor update [version] [--component node|tg|agent] [--no-restart] [--skip-verify]
func runUpdate(restart bool) {
	comp, ver := "node", ""
	doRestart := restart
	skipVerify := false
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "--component", "-c":
			if i+1 < len(args) {
				i++
				comp = args[i]
			}
		case "--version", "-v":
			if i+1 < len(args) {
				i++
				ver = args[i]
			}
		case "--no-restart":
			doRestart = false
		case "--restart":
			doRestart = true
		case "--skip-verify":
			skipVerify = true
		case "--help", "-h":
			fmt.Println("netductor update [version] [--component node|tg|agent] [--no-restart] [--skip-verify]")
			fmt.Println("Downloads GitHub Release asset into /usr/local/bin. Verifies SHA256SUMS unless --skip-verify or NETDUCTOR_UPDATE_SKIP_VERIFY=1.")
			return
		default:
			if !strings.HasPrefix(a, "-") && ver == "" {
				ver = a
			}
		}
	}
	if skipVerify {
		_ = os.Setenv("NETDUCTOR_UPDATE_SKIP_VERIFY", "1")
	}
	tag := strings.TrimSpace(ver)
	if tag == "" {
		var err error
		tag, err = ndupdate.LatestReleaseTag()
		if err != nil {
			fmt.Fprintln(os.Stderr, "latest release:", err)
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
	fmt.Fprintln(os.Stderr, "==> update", comp, tag, "→", dest)
	if err := ndupdate.DownloadReleaseAsset(tag, comp, dest); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ndupdate.WriteVERSION(tag)
	fmt.Println("updated", dest, tag)
	if doRestart && unit != "" {
		_ = exec.Command("systemctl", "try-restart", unit).Run()
		if unit == "netductor-api" {
			_ = exec.Command("systemctl", "try-restart", "netductor-telegram-bot").Run()
		}
		fmt.Println("restarted", unit, "(try)")
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
