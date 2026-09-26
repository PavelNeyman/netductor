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

// runUpdate replaces the local binary and optionally restarts services.
func runUpdate(restart bool) {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dest := "/usr/local/bin/netductor"
	data, err := os.ReadFile(exe)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = os.MkdirAll("/opt/netductor/bin", 0o755)
	if want := os.Getenv("NETDUCTOR_UPDATE_SHA256"); want != "" {
		sum := sha256Hex(data)
		if sum != strings.TrimSpace(want) {
			fmt.Fprintln(os.Stderr, "sha256 mismatch", sum, "!=", want)
			os.Exit(1)
		}
	}
	tmp := dest + ".new"
	if err := os.WriteFile(tmp, data, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = exec.Command("systemctl", "stop", "netductor-api").Run()
	if err := os.Rename(tmp, dest); err != nil {
		fmt.Fprintf(os.Stderr, "wrote %s (rename failed: %v)\n", tmp, err)
	} else {
		fmt.Println("updated", dest)
	}
	// keep parallel copy for unit ExecStart paths
	_ = exec.Command("cp", "-f", dest, "/opt/netductor/bin/netductor").Run()
	if restart {
		_ = exec.Command("systemctl", "start", "netductor-api").Run()
		_ = exec.Command("systemctl", "try-restart", "netductor-telegram-bot").Run()
		fmt.Println("services restarted")
	} else {
		fmt.Println("restart: systemctl start netductor-api")
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
