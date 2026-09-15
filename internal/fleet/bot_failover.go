package fleet

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/PavelNeyman/netductor/internal/nodes"
	"github.com/PavelNeyman/netductor/internal/paths"
)

// Telegram from RU is often blocked. Model:
//   - Active bot always on control-plane primary (abroad).
//   - Standby on RU secondary: only if primary bot is down.
//   - Standby traffic to api.telegram.org goes via SOCKS5 on 127.0.0.1:1089
//     provided by `ssh -D` to primary (exit IP = core). MTProxy is unnecessary
//     for Bot API (HTTPS); SOCKS over SSH is enough and reuses existing keys.

const (
	botProxyPort = "1089"
	botStandbyUnit = "netductor-telegram-bot-standby"
	sshTunnelUnit  = "netductor-tg-socks-tunnel"
)

// InstallBotStandbyUnits writes systemd units on the local host (intended for secondary).
// Does not enable standby by default — only tunnel helper + standby unit for failover.
func InstallBotStandbyUnits(primarySSH string) error {
	primarySSH = strings.TrimSpace(primarySSH)
	if primarySSH == "" {
		primarySSH = peerHostFromBackupOffsite()
	}
	if primarySSH == "" {
		return fmt.Errorf("primary SSH user@host required")
	}
	if !strings.Contains(primarySSH, "@") {
		primarySSH = "root@" + primarySSH
	}
	bin := filepath.Join(paths.OptDir(), "bin", "netductor-tg")
	if _, err := os.Stat(bin); err != nil {
		bin = "/usr/local/bin/netductor-tg"
	}

	tunnel := fmt.Sprintf(`[Unit]
Description=Netductor SOCKS to primary (TG exit via abroad core)
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/bin/ssh -N -D 127.0.0.1:%s -o BatchMode=yes -o StrictHostKeyChecking=accept-new -o ServerAliveInterval=30 -o ExitOnForwardFailure=yes %s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, botProxyPort, primarySSH)

	standby := fmt.Sprintf(`[Unit]
Description=Netductor Telegram bot (standby via SOCKS to primary)
After=network-online.target %s.service
Requires=%s.service

[Service]
Type=simple
Environment=ALL_PROXY=socks5://127.0.0.1:%s
Environment=HTTPS_PROXY=socks5://127.0.0.1:%s
Environment=HTTP_PROXY=socks5://127.0.0.1:%s
ExecStart=%s
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
`, sshTunnelUnit, sshTunnelUnit, botProxyPort, botProxyPort, botProxyPort, bin)

	_ = os.MkdirAll("/etc/systemd/system", 0o755)
	if err := os.WriteFile("/etc/systemd/system/"+sshTunnelUnit+".service", []byte(tunnel), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile("/etc/systemd/system/"+botStandbyUnit+".service", []byte(standby), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "daemon-reload").Run()
	fmt.Fprintln(os.Stderr, "standby units installed (not started). Use: netductor fleet bot-failover check|promote|demote")
	return nil
}

// BotPrimaryHealthy checks primary bot.
// Prefer HTTP :8788/api/bot-status (works without reverse SSH key).
// Note: if whole primary host is down, SOCKS via core cannot work either —
// standby only covers "host up, bot process down".
func BotPrimaryHealthy() (bool, string) {
	p := LoadPolicy()
	localIP := publicIPGuess()
	host := ""
	if p.PrimarySSH != "" {
		host = p.PrimarySSH
		if i := strings.LastIndex(host, "@"); i >= 0 {
			host = host[i+1:]
		}
	}
	if host == "" && p.PrimaryNodeID != "" {
		if n, ok, err := nodes.Get(p.PrimaryNodeID); err == nil && ok {
			host = n.PublicIP
		}
	}
	if host == "" || host == localIP || host == "127.0.0.1" {
		return localBotActive(), "local"
	}
	url := "http://" + host + ":8788/api/bot-status"
	out, err := exec.Command("curl", "-fsS", "--max-time", "8", url).CombinedOutput()
	s := strings.TrimSpace(string(out))
	if err != nil {
		return false, "http:" + s
	}
	if strings.Contains(s, `"ok":true`) || strings.Contains(s, `"ok": true`) {
		return true, s
	}
	return false, s
}

func localBotActive() bool {
	out, _ := exec.Command("systemctl", "is-active", "netductor-telegram-bot").Output()
	return strings.TrimSpace(string(out)) == "active"
}

// PromoteStandbyBot starts SOCKS tunnel + standby bot (on secondary).
func PromoteStandbyBot() error {
	_ = exec.Command("systemctl", "start", sshTunnelUnit).Run()
	time.Sleep(1 * time.Second)
	return exec.Command("systemctl", "start", botStandbyUnit).Run()
}

// DemoteStandbyBot stops standby bot (keep tunnel optional).
func DemoteStandbyBot() error {
	_ = exec.Command("systemctl", "stop", botStandbyUnit).Run()
	return exec.Command("systemctl", "stop", sshTunnelUnit).Run()
}

// CheckBotFailover: if primary bot process down but host reachable → promote standby via SOCKS.
// If primary host unreachable, demote standby (SOCKS cannot exit via dead core).
func CheckBotFailover() (string, error) {
	ok, detail := BotPrimaryHealthy()
	if ok {
		_ = DemoteStandbyBot()
		return "primary healthy (" + detail + ") — standby down", nil
	}
	if strings.Contains(detail, "http:") || strings.Contains(detail, "Connection") || strings.Contains(detail, "timed out") {
		_ = DemoteStandbyBot()
		return "primary unreachable (" + detail + ") — standby not used (SOCKS needs core up)", nil
	}
	if err := PromoteStandbyBot(); err != nil {
		return "primary bot down (" + detail + ") — promote failed: " + err.Error(), err
	}
	return "primary bot down (" + detail + ") — standby promoted via SOCKS→core", nil
}

// InstallBotFailoverTimer on secondary: periodic check.
func InstallBotFailoverTimer() error {
	svc := `[Unit]
Description=Netductor bot failover check once

[Service]
Type=oneshot
ExecStart=/usr/local/bin/netductor fleet bot-failover check
`
	timer := `[Unit]
Description=Netductor bot failover every 2 min

[Timer]
OnBootSec=2min
OnUnitActiveSec=2min
Persistent=true

[Install]
WantedBy=timers.target
`
	_ = os.WriteFile("/etc/systemd/system/netductor-bot-failover.service", []byte(svc), 0o644)
	_ = os.WriteFile("/etc/systemd/system/netductor-bot-failover.timer", []byte(timer), 0o644)
	_ = exec.Command("systemctl", "daemon-reload").Run()
	_ = exec.Command("systemctl", "enable", "--now", "netductor-bot-failover.timer").Run()
	return nil
}
