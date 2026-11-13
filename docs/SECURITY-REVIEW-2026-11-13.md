# Netductor Code Review

**Date:** 2026-11-13 20:25 UTC
**Files:** 528
**regex:** 612

| Severity | Count |
|---|---|
| HIGH | 22 |
| MED | 583 |
| LOW | 7 |

## [HIGH] 22

| File | Line | Pattern | Code |
|---|---|---|---|
| `internal/aptguard/remediate.go` | 678 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=noninteracti` |
| `internal/install/lampac.go` | 75 | EXEC-SHELL | `script := exec.Command("sh", "-c", "curl -fsSL https://get.docker.com ` |
| `internal/install/apt.go` | 49 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=noninteracti` |
| `internal/install/apt.go` | 83 | EXEC-SHELL | `cmd := exec.Command("bash", args...)` |
| `internal/update/build_host.go` | 85 | EXEC-SHELL | `sum := exec.Command("bash", "-c", "sha256sum netductor-* > SHA256SUMS"` |
| `internal/svcpaths/bootstrap.go` | 100 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "apt-get update -qq && DEBIAN_FRONTE` |
| `internal/svcpaths/bootstrap.go` | 138 | EXEC-SHELL | `if out, err := exec.Command("bash", "-c", script).CombinedOutput(); er` |
| `internal/svcpaths/build_paths.go` | 120 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/build_paths.go` | 123 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/build_paths.go` | 182 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `cmd/netductor-agent/build_role.go` | 183 | EXEC-SHELL | `_ = exec.Command("bash", "-lc", "cd '"+workDir+"' && tar -czf dist.tgz` |
| `cmd/netductor-agent/build_role.go` | 224 | EXEC-SHELL | `cmd = exec.Command("bash", "-lc", "cd '"+src+"' && go test ./... 2>&1;` |
| `cmd/netductor-agent/agent_ops.go` | 602 | EXEC-SHELL | `_ = exec.Command("sh", "-c", "mountpoint -q /mnt/netductor // (block i` |
| `cmd/netductor-agent/agent_ops.go` | 660 | EXEC-SHELL | `out, err := exec.Command("sh", "-c", script).CombinedOutput()` |
| `cmd/netductor-agent/facts_local.go` | 25 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "awk -F': ' '/^Serial/{print $` |
| `cmd/netductor-agent/facts_local.go` | 76 | EXEC-SHELL | `if matches, _ := exec.Command("sh", "-c", "ls /dev/sd[a-z] 2>/dev/null` |
| `cmd/netductor-agent/facts_local.go` | 82 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "mount 2>/dev/null / grep ' on` |
| `cmd/netductor-agent/facts_local.go` | 89 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "for d in /sys/block/sd*; do [` |
| `cmd/netductor-agent/facts_local.go` | 150 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", `uci -q show wireless 2>/dev/n` |
| `cmd/netductor-op/tui_wizard_apply.go` | 558 | SECRET | `return mm.runNetductor("nvr", "cameras", "add", "name="+m.fieldVal("ca` |
| `cmd/netductor-op/cli_project.go` | 284 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf("cp -a %q/build/release/* %` |
| `cmd/netductor-op/cli_project.go` | 320 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", script)` |

## [MED] 583

| File | Line | Pattern | Code |
|---|---|---|---|
| `internal/devices/devices.go` | 109 | ERROR-IGNORE | `_ = saveUnlocked(kept)` |
| `internal/tapo/tpap.go` | 28 | ERROR-IGNORE | `p256P, _  = new(big.Int).SetString("FFFFFFFF00000001000000000000000000` |
| `internal/tapo/tpap.go` | 29 | ERROR-IGNORE | `p256N, _  = new(big.Int).SetString("FFFFFFFF00000000FFFFFFFFFFFFFFFFBC` |
| `internal/tapo/tpap.go` | 31 | ERROR-IGNORE | `p256B, _  = new(big.Int).SetString("5AC635D8AA3A93E7B3EBBD55769886BC65` |
| `internal/tapo/tpap.go` | 32 | ERROR-IGNORE | `p256Gx, _ = new(big.Int).SetString("6B17D1F2E12C4247F8BCE6E563A440F277` |
| `internal/tapo/tpap.go` | 33 | ERROR-IGNORE | `p256Gy, _ = new(big.Int).SetString("4FE342E2FE1A7F9B8EE7EB4A7C0F9E162B` |
| `internal/aptguard/store.go` | 68 | NON-ATOMIC | `if err := os.WriteFile(LatestPath(stateRoot, rep.ServerID), b, 0o600);` |
| `internal/aptguard/store.go` | 76 | NON-ATOMIC | `return os.WriteFile(filepath.Join(hd, name), b, 0o600)` |
| `internal/migrate/migrate.go` | 315 | ERROR-IGNORE | `_ = durableWriteJournal(j)` |
| `internal/migrate/migrate.go` | 347 | ERROR-IGNORE | `_ = durableWriteJournal(j)` |
| `internal/migrate/migrate.go` | 357 | ERROR-IGNORE | `_ = durableWriteJournal(j)` |
| `internal/install/services.go` | 79 | ERROR-IGNORE | `_ = run("systemctl", "disable", "--now", "freshvps-api")` |
| `internal/install/services.go` | 81 | ERROR-IGNORE | `_ = installNodeSyncTimer(bin)` |
| `internal/install/services.go` | 111 | ERROR-IGNORE | `_ = run("systemctl", "enable", "--now", "netductor-node-sync.timer")` |
| `internal/install/services.go` | 142 | ERROR-IGNORE | `_ = run("systemctl", "enable", "netductor-metrics.timer")` |
| `internal/install/services.go` | 143 | ERROR-IGNORE | `_ = run("systemctl", "start", "netductor-metrics.timer")` |
| `internal/install/services.go` | 144 | ERROR-IGNORE | `_ = run(bin, "collect")` |
| `internal/install/services.go` | 149 | ERROR-IGNORE | `_ = aptInstall("curl")` |
| `internal/install/services.go` | 171 | NON-ATOMIC | `_ = os.WriteFile(tmp, b, 0o755)` |
| `internal/install/services.go` | 224 | ERROR-IGNORE | `_ = run("systemctl", "restart", "netductor-telegram-bot")` |
| `internal/install/backup_verify.go` | 108 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(paths.StateDir(), "backup-verify", "las` |
| `internal/install/backup_verify.go` | 144 | ERROR-IGNORE | `_ = run("systemctl", "enable", "--now", "netductor-backup-verify.timer` |
| `internal/install/install.go` | 52 | ERROR-IGNORE | `_ = copySelfToLocalBin()` |
| `internal/install/install.go` | 53 | ERROR-IGNORE | `_ = EnsureDomainConfig()` |
| `internal/install/install.go` | 54 | ERROR-IGNORE | `_ = EnsureClientProfiles()` |
| `internal/install/install.go` | 81 | ERROR-IGNORE | `_ = EnsureClientProfiles()` |
| `internal/install/install.go` | 82 | ERROR-IGNORE | `_ = InstallRedirect()` |
| `internal/install/install.go` | 124 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(paths.StateDir(), "installed_version"),` |
| `internal/install/install.go` | 561 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(paths.EtcDir(), "public_ip"), []byte(ip` |
| `internal/install/install.go` | 572 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(paths.EtcDir(), "READY.txt"), []byte(tx` |
| `internal/install/install.go` | 606 | NON-ATOMIC | `if err := os.WriteFile(tmp, in, 0o755); err != nil {` |
| `internal/install/lampac.go` | 52 | NON-ATOMIC | `_ = os.WriteFile("/etc/netductor/addons/lampac.tag", []byte(upstream.L` |
| `internal/install/lampac.go` | 71 | ERROR-IGNORE | `_ = aptInstall("docker.io")` |
| `internal/install/lampac.go` | 72 | ERROR-IGNORE | `_ = aptInstall("docker-cli")` |
| `internal/install/ssh_harden.go` | 47 | NON-ATOMIC | `if err := os.WriteFile(ak, []byte(pubLine+"\n"), 0o600); err != nil {` |
| `internal/install/ssh_harden.go` | 60 | NON-ATOMIC | `if err := os.WriteFile(drop, []byte(hardening.DropInConf()), 0o644); e` |
| `internal/install/ssh_harden.go` | 96 | NON-ATOMIC | `_ = os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644)` |
| `internal/install/blocky.go` | 28 | ERROR-IGNORE | `_ = aptInstall("curl", "tar", "dnsutils")` |
| `internal/install/blocky.go` | 100 | NON-ATOMIC | `_ = os.WriteFile(cfg, []byte(content), 0o644)` |
| `internal/install/blocky.go` | 210 | NON-ATOMIC | `_ = os.WriteFile("/etc/blocky/netductor-dot.snippet.yml", []byte(` |
| `internal/install/blocky.go` | 220 | NON-ATOMIC | `_ = os.WriteFile(cfg+".bak-dot", b, 0o644)` |
| `internal/install/blocky.go` | 224 | NON-ATOMIC | `if err := os.WriteFile(cfg, []byte(s), 0o644); err != nil {` |
| `internal/install/blocky.go` | 227 | ERROR-IGNORE | `_ = run("systemctl", "restart", "blocky")` |
| `internal/install/backup_alt.go` | 163 | NON-ATOMIC | `_ = os.WriteFile("/etc/systemd/system/netductor-secondary-local-backup` |
| `internal/install/backup_alt.go` | 164 | NON-ATOMIC | `_ = os.WriteFile("/etc/systemd/system/netductor-secondary-local-backup` |
| `internal/install/backup_schedule.go` | 53 | NON-ATOMIC | `if err := os.WriteFile(backupSchedulePath(), append(raw, '\n'), 0o600)` |
| `internal/install/backup_schedule.go` | 65 | NON-ATOMIC | `if err := os.WriteFile(filepath.Join(dir, "schedule.conf"), []byte(con` |
| `internal/install/backup_schedule.go` | 68 | ERROR-IGNORE | `_ = run("systemctl", "daemon-reload")` |
| `internal/install/backup_schedule.go` | 69 | ERROR-IGNORE | `_ = run("systemctl", "restart", "netductor-backup.timer")` |
| `internal/install/backup_schedule.go` | 121 | NON-ATOMIC | `return os.WriteFile(backupKeepPath(), append(raw, '\n'), 0o600)` |
| `internal/install/version.go` | 21 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(paths.StateDir(), name+"-version"), []b` |
| `internal/install/host_agents.go` | 212 | NON-ATOMIC | `return os.WriteFile(aptPreferencesHoster, []byte(b.String()), 0o644)` |
| `internal/install/baseline_snapshot.go` | 34 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(dir, file), body, 0o644)` |
| `internal/install/baseline_snapshot.go` | 44 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(dir, "created"),` |
| `internal/install/hardening.go` | 26 | NON-ATOMIC | `_ = os.WriteFile("/etc/sysctl.d/99-netductor.conf", []byte(` |
| `internal/install/hardening.go` | 28 | ERROR-IGNORE | `_ = run("sysctl", "--system")` |
| `internal/install/hardening.go` | 55 | ERROR-IGNORE | `_ = SnapshotHostBaseline()` |
| `internal/install/hardening.go` | 56 | ERROR-IGNORE | `_ = EnsureUnattendedSecurity()` |
| `internal/install/hardening.go` | 57 | ERROR-IGNORE | `_ = EnsureBlockyDoT()` |
| `internal/install/hardening.go` | 61 | ERROR-IGNORE | `_ = InstallBackupVerifyTimer()` |

## [LOW] 7

| File | Line | Pattern | Code |
|---|---|---|---|
| `internal/secondary/provision.go` | 334 | TODO | `"TD=$(mktemp -d /tmp/nd-join.XXXXXX)\n" +` |
| `internal/secondary/provision.go` | 339 | TODO | `"NEW=$(mktemp /usr/local/bin/.netductor.new.XXXXXX)\n" +` |
| `internal/secondary/provision.go` | 373 | TODO | `"  PREV_TMP=$(mktemp /var/lib/netductor/.netductor.bin.prev.XXXXXX)\n"` |
| `internal/secondary/provision.go` | 386 | TODO | `"    RESTORE=$(mktemp /usr/local/bin/.netductor.restore.XXXXXX)\n" +` |
| `internal/secondary/provision.go` | 473 | TODO | `STAGE=$(mktemp -d "$P/.mtls-stage.XXXXXX")` |
| `internal/secondary/provision.go` | 508 | TODO | `BACKUP=$(mktemp -d "$P/.mtls-backup.XXXXXX")` |
| `internal/svcpaths/bootstrap.go` | 125 | TODO | `tmpdir=$(mktemp -d /tmp/netductor-wstunnel.XXXXXX)` |

---
*staticcheck + gosec работают в фоне, результаты в следующем вызове*
