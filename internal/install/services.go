package install

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/PavelNeyman/netductor/internal/paths"
	ndupdate "github.com/PavelNeyman/netductor/internal/update"
	"github.com/PavelNeyman/netductor/internal/version"
	"github.com/PavelNeyman/netductor/internal/vpn"
)

func InstallVPNUsers() error {
	if err := vpn.EnsureDirs(); err != nil {
		return err
	}
	// seed operator
	users, _ := vpn.ListNative()
	has := false
	for _, u := range users {
		if u.Name == "operator" {
			has = true
			break
		}
	}
	if !has {
		if _, err := vpn.AddNative("operator", "auto-created at install"); err != nil {
			return err
		}
	} else {
		_ = vpn.ApplyConfig()
	}
	return nil
}

func InstallAPI() error {
	if readSecret("edge_bootstrap_token") == "" {
		_ = writeSecret("edge_bootstrap_token", randomHex(32))
	}
	if readSecret("edge_token") == "" {
		_ = writeSecret("edge_token", randomHex(32))
	}

	if out, _ := runOut("systemctl", "is-active", "netductor-api"); strings.TrimSpace(out) == "active" {
		fmt.Fprintln(os.Stderr, "netductor-api already active — reconfigure unit only")
	}
	// self binary already expected at /usr/local/bin/netductor
	bin := "/usr/local/bin/netductor"
	if _, err := os.Stat(bin); err != nil {
		return fmt.Errorf("netductor binary not at %s — install release asset first", bin)
	}
	// VPS admin UI removed — use netductor-op on Mac.
	unit := fmt.Sprintf(`[Unit]
Description=Netductor API
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=%s serve --bind 127.0.0.1 --port 8787 --no-proxy
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
`, bin)
	if err := writeUnit("netductor-api.service", unit); err != nil {
		return err
	}
	// stop legacy python if any
	if out, _ := runOut("systemctl", "cat", "freshvps-api.service"); strings.Contains(out, "[Unit]") {
		_ = run("systemctl", "disable", "--now", "freshvps-api")
	}
	_ = installNodeSyncTimer(bin)
	return enableStart("netductor-api")
}

func installNodeSyncTimer(bin string) error {
	svc := fmt.Sprintf(`[Unit]
Description=Netductor node registry sync (hostname desired)
After=network-online.target

[Service]
Type=oneshot
ExecStart=%s nodes sync-local
`, bin)
	timer := `[Unit]
Description=Netductor node sync timer

[Timer]
OnBootSec=2min
OnUnitActiveSec=5min
Persistent=true

[Install]
WantedBy=timers.target
`
	if err := writeUnit("netductor-node-sync.service", svc); err != nil {
		return err
	}
	if err := writeUnit("netductor-node-sync.timer", timer); err != nil {
		return err
	}
	_ = run("systemctl", "enable", "--now", "netductor-node-sync.timer")
	return nil
}

func InstallMetrics() error {
	bin := "/usr/local/bin/netductor"
	unit := fmt.Sprintf(`[Unit]
Description=Netductor metrics sample
After=network-online.target

[Service]
Type=oneshot
ExecStart=%s collect
`, bin)
	timer := `[Unit]
Description=Netductor metrics every minute

[Timer]
OnBootSec=30
OnUnitActiveSec=60
AccuracySec=10

[Install]
WantedBy=timers.target
`
	if err := writeUnit("netductor-metrics.service", unit); err != nil {
		return err
	}
	if err := writeUnit("netductor-metrics.timer", timer); err != nil {
		return err
	}
	_ = run("systemctl", "enable", "netductor-metrics.timer")
	_ = run("systemctl", "start", "netductor-metrics.timer")
	_ = run(bin, "collect")
	return nil
}

func InstallTelegram() error {
	_ = aptInstall("curl")
	a := runtime.GOARCH
	if a == "arm" {
		a = "arm64" // best effort
	}
	ver := version.Release
	// Always install a real binary at /usr/local/bin/netductor-tg (never symlink to self).
	dest := "/usr/local/bin/netductor-tg"
	_ = os.MkdirAll(filepath.Dir(dest), 0o755)
	haveBin := false
	if err := ndupdate.DownloadReleaseAsset("v"+ver, "tg", dest); err != nil {
		fmt.Fprintf(os.Stderr, "telegram binary download: %v — trying local/fallback\n", err)
		tmp := dest + ".new"
		for _, src := range []string{
			"/tmp/netductor-tg.bin",
			filepath.Join(paths.BinDir(), "netductor-tg.real"),
		} {
			st, e := os.Lstat(src)
			if e != nil || st.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if b, e := os.ReadFile(src); e == nil && len(b) > 1000 {
				_ = os.WriteFile(tmp, b, 0o755)
				haveBin = true
				break
			}
		}
		if !haveBin {
			return fmt.Errorf("telegram: netductor-tg binary missing for %s — publish netductor-tg-linux-%s on release v%s, then: netductor install telegram", a, a, ver)
		}
		_ = os.Chmod(tmp, 0o755)
		_ = os.Remove(dest)
		if err := os.Rename(tmp, dest); err != nil {
			return fmt.Errorf("telegram install binary: %w", err)
		}
	} else {
		haveBin = true
	}
	if !haveBin {
		return nil
	}
	unit := fmt.Sprintf(`[Unit]
Description=Netductor Telegram bot
After=network-online.target

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=5
# Cap RSS so a backup/update leak cannot OOM the whole VPS
MemoryMax=512M
MemoryHigh=384M
OOMPolicy=stop

[Install]
WantedBy=multi-user.target
`, dest)
	if err := writeUnit("netductor-telegram-bot.service", unit); err != nil {
		return err
	}
	tok := filepath.Join(paths.EtcDir(), "secrets", "telegram_bot_token")
	if _, e1 := os.Stat(tok); e1 != nil {
		fmt.Fprintln(os.Stderr, "telegram_bot_token missing — unit installed, not started")
		return nil
	}
	chat := filepath.Join(paths.EtcDir(), "secrets", "telegram_admin_id")
	if _, e2 := os.Stat(chat); e2 != nil {
		fmt.Fprintln(os.Stderr, "telegram_admin_id missing — set secrets/telegram_admin_id (or NETDUCTOR_TG_ADMIN)")
	}
	if err := enableStart("netductor-telegram-bot"); err != nil {
		return err
	}
	// ensure running after install
	_ = run("systemctl", "restart", "netductor-telegram-bot")
	return nil
}
