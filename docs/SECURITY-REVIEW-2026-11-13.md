# Netductor Code Review — Automated Analysis

**Date:** 2026-11-13 20:08 UTC
**Files:** 528 .go files
**Findings:** 641
**Method:** Regex pattern analysis (7 patterns)

---

## Summary

| Severity | Count |
|----------|-------|
| HIGH | 32 |
| MED | 602 |
| LOW | 7 |

---

## Findings by Pattern

| Pattern | Count | Severity |
|---|---|---|
| ERROR-IGNORE | 306 | MED |
| NON-ATOMIC | 277 | MED |
| EXEC-SHELL | 21 | HIGH |
| HTTP-NO-TIMEOUT | 19 | MED |
| EXEC-SPRINTF | 10 | HIGH |
| TODO | 7 | LOW |
| SECRET | 1 | HIGH |

---

## [HIGH] 32 findings

| File | Line | Pattern | Code |
|---|---|---|---|
| `cmd/netductor-agent/facts_local.go` | 25 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "awk -F': ' '/^Serial/{print $` |
| `cmd/netductor-agent/facts_local.go` | 76 | EXEC-SHELL | `if matches, _ := exec.Command("sh", "-c", "ls /dev/sd[a-z] 2>/dev/null` |
| `cmd/netductor-agent/facts_local.go` | 82 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "mount 2>/dev/null \| grep ' o` |
| `cmd/netductor-agent/facts_local.go` | 89 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "for d in /sys/block/sd*; do [` |
| `cmd/netductor-agent/facts_local.go` | 150 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", `uci -q show wireless 2>/dev/n` |
| `cmd/netductor-agent/agent_ops.go` | 602 | EXEC-SHELL | `_ = exec.Command("sh", "-c", "mountpoint -q /mnt/netductor \|\| (block` |
| `cmd/netductor-agent/agent_ops.go` | 660 | EXEC-SHELL | `out, err := exec.Command("sh", "-c", script).CombinedOutput()` |
| `cmd/netductor-agent/build_role.go` | 183 | EXEC-SHELL | `_ = exec.Command("bash", "-lc", "cd '"+workDir+"' && tar -czf dist.tgz` |
| `cmd/netductor-agent/build_role.go` | 224 | EXEC-SHELL | `cmd = exec.Command("bash", "-lc", "cd '"+src+"' && go test ./... 2>&1;` |
| `cmd/netductor-op/tui_wizard_apply.go` | 558 | SECRET | `return mm.runNetductor("nvr", "cameras", "add", "name="+m.fieldVal("ca` |
| `cmd/netductor-op/cli_project.go` | 284 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf("cp -a %q/build/release/* %` |
| `cmd/netductor-op/cli_project.go` | 284 | EXEC-SPRINTF | `_ = exec.Command("bash", "-c", fmt.Sprintf("cp -a %q/build/release/* %` |
| `cmd/netductor-op/cli_project.go` | 320 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", script)` |
| `internal/install/ssh_harden.go` | 105 | EXEC-SPRINTF | `_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", port), "comment` |
| `internal/install/apt.go` | 49 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=noninteracti` |
| `internal/install/apt.go` | 83 | EXEC-SHELL | `cmd := exec.Command("bash", args...)` |
| `internal/install/lampac.go` | 75 | EXEC-SHELL | `script := exec.Command("sh", "-c", "curl -fsSL https://get.docker.com ` |
| `internal/probes/run.go` | 19 | EXEC-SPRINTF | `out, err := exec.Command("ss", flag, "sport", "=", fmt.Sprintf(":%d", ` |
| `internal/update/build_host.go` | 85 | EXEC-SHELL | `sum := exec.Command("bash", "-c", "sha256sum netductor-* > SHA256SUMS"` |
| `internal/vpn/uplink_ab.go` | 711 | EXEC-SPRINTF | `_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", p), "comment", ` |
| `internal/vpn/uplink_ab.go` | 714 | EXEC-SPRINTF | `_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/udp", cfg.WGListenPor` |
| `internal/vpn/uplink_ab.go` | 719 | EXEC-SPRINTF | `_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", p.Port), "comme` |
| `internal/svcpaths/build_paths.go` | 120 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/build_paths.go` | 120 | EXEC-SPRINTF | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/build_paths.go` | 123 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/build_paths.go` | 123 | EXEC-SPRINTF | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/build_paths.go` | 182 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/build_paths.go` | 182 | EXEC-SPRINTF | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/bootstrap.go` | 100 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "apt-get update -qq && DEBIAN_FRONTE` |
| `internal/svcpaths/bootstrap.go` | 138 | EXEC-SHELL | `if out, err := exec.Command("bash", "-c", script).CombinedOutput(); er` |
| `internal/aptguard/remediate.go` | 678 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=noninteracti` |
| `internal/git/git.go` | 189 | EXEC-SPRINTF | `cmd := exec.Command("git", "-C", dir, "log", fmt.Sprintf("-%d", n), "-` |

---

## [MED] 602 findings

| File | Line | Pattern | Code |
|---|---|---|---|
| `cmd/netductor/cli_secondary.go` | 72 | NON-ATOMIC | `if err := os.WriteFile(out, append(raw, '\n'), 0o600); err != nil {` |
| `cmd/netductor/cli_secondary.go` | 206 | ERROR-IGNORE | `_ = execLocal("ssh-keygen", "-f", "/root/.ssh/known_hosts", "-R", host` |
| `cmd/netductor/cli_secondary.go` | 483 | ERROR-IGNORE | `_ = execLocal("ssh-keygen", "-f", "/root/.ssh/known_hosts", "-R", host` |
| `cmd/netductor/cli_secondary.go` | 530 | ERROR-IGNORE | `_ = execLocal("systemctl", "try-restart", "sing-box")` |
| `cmd/netductor/cli_secondary.go` | 552 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(paths.EtcDir(), "secrets", "last_second` |
| `cmd/netductor/api_jobs.go` | 36 | ERROR-IGNORE | `_ = primaryJobStore()` |
| `cmd/netductor/collect.go` | 145 | NON-ATOMIC | `_ = os.WriteFile(hist, lines, 0o644)` |
| `cmd/netductor/collect.go` | 149 | NON-ATOMIC | `_ = os.WriteFile(latest, append(line, '\n'), 0o644)` |
| `cmd/netductor/cli_completion.go` | 108 | NON-ATOMIC | `if err := os.WriteFile(path, []byte(bashCompletionScript()), 0o644); e` |
| `cmd/netductor/doctor.go` | 182 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(state, "installed_version"), []byte(ver` |
| `cmd/netductor/cli_edge.go` | 205 | ERROR-IGNORE | `_ = sitesAttach(site, did)` |
| `cmd/netductor/cli_edge.go` | 246 | NON-ATOMIC | `if err := os.WriteFile(out, append(b, 10), 0o600); err != nil {` |
| `cmd/netductor/cli_edge.go` | 282 | ERROR-IGNORE | `_ = sitesAttach(args[2], args[1])` |
| `cmd/netductor/nodes_cli.go` | 89 | NON-ATOMIC | `_ = os.WriteFile(logf, []byte("started\n"), 0o600)` |
| `cmd/netductor/cli_uplink_mux.go` | 28 | NON-ATOMIC | `if err := os.WriteFile(path, []byte(mode+"\n"), 0o644); err != nil {` |
| `cmd/netductor-agent/luci.go` | 79 | NON-ATOMIC | `_ = os.WriteFile(luciUntilPath(), []byte(strconv.FormatInt(until.Unix(` |
| `cmd/netductor-agent/luci.go` | 110 | NON-ATOMIC | `_ = os.WriteFile(luciUntilPath(), []byte(strconv.FormatInt(newUntil.Un` |
| `cmd/netductor-agent/guest_cli.go` | 130 | ERROR-IGNORE | `_ = applyGuestFirewallAllow(st)` |
| `cmd/netductor-agent/guest_cli.go` | 138 | ERROR-IGNORE | `_ = applyGuestFirewallAllow(st)` |
| `cmd/netductor-agent/cmd_done.go` | 67 | NON-ATOMIC | `if err := os.WriteFile(tmp, []byte("1\n"), 0o600); err != nil {` |
| `cmd/netductor-agent/cmd_done.go` | 82 | NON-ATOMIC | `if err := os.WriteFile(tmp, []byte(result), 0o600); err != nil {` |
| `cmd/netductor-agent/tlsclient.go` | 30 | NON-ATOMIC | `_ = os.WriteFile(mtlsRequiredFlagPath(), []byte("1\n"), 0o600)` |
| `cmd/netductor-agent/guest_http.go` | 117 | ERROR-IGNORE | `_ = installGuestNFTHooks()` |
| `cmd/netductor-agent/guest_http.go` | 118 | ERROR-IGNORE | `_ = applyGuestFirewallAllow(store)` |
| `cmd/netductor-agent/guest_http.go` | 341 | ERROR-IGNORE | `_ = applyGuestFirewallAllow(store)` |
| `cmd/netductor-agent/guest_http.go` | 431 | ERROR-IGNORE | `_ = applyGuestFirewallAllow(store)` |
| `cmd/netductor-agent/guest_http.go` | 458 | ERROR-IGNORE | `_ = applyGuestFirewallAllow(store)` |
| `cmd/netductor-agent/guest_apply.go` | 116 | ERROR-IGNORE | `_ = applyGuestVPNBypass(ip)` |
| `cmd/netductor-agent/guest_apply.go` | 130 | ERROR-IGNORE | `_ = applyGuestFirewallAllow(st)` |
| `cmd/netductor-agent/guest_apply.go` | 182 | NON-ATOMIC | `if err := os.WriteFile(alt, []byte(nft), 0o644); err != nil {` |
| `cmd/netductor-agent/guest_apply.go` | 220 | NON-ATOMIC | `if err := os.WriteFile(libexec, []byte(script), 0o755); err != nil {` |
| `cmd/netductor-agent/guest_apply.go` | 222 | NON-ATOMIC | `_ = os.WriteFile(alt, []byte(script), 0o755)` |
| `cmd/netductor-agent/guest_apply.go` | 244 | NON-ATOMIC | `_ = os.WriteFile(path, []byte(body), 0o755)` |
| `cmd/netductor-agent/guest_apply.go` | 248 | NON-ATOMIC | `_ = os.WriteFile("/etc/hotplug.d/iface/48-netductor-guest-vpn", []byte` |
| `cmd/netductor-agent/guest_apply.go` | 270 | NON-ATOMIC | `if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {` |
| `cmd/netductor-agent/guest_apply.go` | 273 | ERROR-IGNORE | `_ = installGuestNFTHooks()` |
| `cmd/netductor-agent/guest_apply.go` | 322 | ERROR-IGNORE | `_ = applyGuestFirewallAllow(store)` |
| `cmd/netductor-agent/nvr_cmds.go` | 141 | ERROR-IGNORE | `_ = nvrTrimTmp(cfg)` |
| `cmd/netductor-agent/nvr_cmds.go` | 410 | ERROR-IGNORE | `_ = nvrTrimTmp(cfg)` |
| `cmd/netductor-agent/recovery_http.go` | 42 | NON-ATOMIC | `_ = os.WriteFile(path, []byte(s+"\n"), 0o600)` |
| `cmd/netductor-agent/recovery_http.go` | 120 | ERROR-IGNORE | `_ = ensureRecoveryUnlock()` |
| `cmd/netductor-agent/recovery_http.go` | 345 | NON-ATOMIC | `return os.WriteFile(path, []byte(b.String()), 0o600)` |
| `cmd/netductor-agent/main.go` | 125 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(agentDir(), "applied_template"), []byte` |
| `cmd/netductor-agent/main.go` | 177 | NON-ATOMIC | `_ = os.WriteFile(marker, []byte(res+"\n"), 0o600)` |
| `cmd/netductor-agent/main.go` | 203 | NON-ATOMIC | `return os.WriteFile(deviceTokenPath(), []byte(tok+"\n"), 0o600)` |
| `cmd/netductor-agent/main.go` | 232 | NON-ATOMIC | `if err := os.WriteFile(tmp, payload, 0o600); err != nil {` |
| `cmd/netductor-agent/main.go` | 624 | NON-ATOMIC | `_ = os.WriteFile(statePath, append([]byte("ok"), 10), 0o644)` |
| `cmd/netductor-agent/main.go` | 630 | NON-ATOMIC | `_ = os.WriteFile(statePath, append([]byte("fail"), 10), 0o644)` |
| `cmd/netductor-agent/main.go` | 670 | ERROR-IGNORE | `_, _ = doJSON(client, http.MethodPost, cfg.Server+"/api/edge/metrics",` |
| `cmd/netductor-agent/main.go` | 1090 | ERROR-IGNORE | `_ = rollback()` |
| `cmd/netductor-agent/main.go` | 1095 | ERROR-IGNORE | `_ = rollback()` |
| `cmd/netductor-agent/main.go` | 1101 | ERROR-IGNORE | `_ = rollback()` |
| `cmd/netductor-agent/main.go` | 1106 | ERROR-IGNORE | `_ = rollback()` |
| `cmd/netductor-agent/main.go` | 1116 | ERROR-IGNORE | `_ = rollback()` |
| `cmd/netductor-agent/main.go` | 1122 | ERROR-IGNORE | `_ = rollback()` |
| `cmd/netductor-agent/agent_ops.go` | 348 | NON-ATOMIC | `_ = os.WriteFile("/etc/netductor-agent/vpn.subscription", []byte(sub+"` |
| `cmd/netductor-agent/agent_ops.go` | 352 | NON-ATOMIC | `_ = os.WriteFile("/etc/netductor-agent/vpn.vless", []byte(vless+"\n"),` |
| `cmd/netductor-agent/agent_ops.go` | 392 | NON-ATOMIC | `_ = os.WriteFile(path, boxJSON, 0o600)` |
| `cmd/netductor-agent/agent_ops.go` | 401 | NON-ATOMIC | `_ = os.WriteFile("/etc/init.d/netductor-vpn", []byte(init), 0o755)` |
| `cmd/netductor-agent/agent_ops.go` | 413 | NON-ATOMIC | `_ = os.WriteFile(path, cfg2, 0o600)` |
| `cmd/netductor-agent/agent_ops.go` | 457 | NON-ATOMIC | `if err := os.WriteFile(tmp, body, 0o600); err != nil {` |
| `cmd/netductor-agent/agent_ops.go` | 596 | NON-ATOMIC | `if err := os.WriteFile(tmp, body, 0o600); err != nil {` |
| `cmd/netductor-agent/agent_ops.go` | 610 | NON-ATOMIC | `_ = os.WriteFile("/etc/netductor-agent/restore_ready", []byte("1\n"), ` |
| `cmd/netductor-agent/build_role.go` | 185 | ERROR-IGNORE | `_ = uploadArtifacts(client, cfg, out.Job.ID, distTGZ)` |
| `cmd/netductor-agent/guest_from_template.go` | 129 | ERROR-IGNORE | `_ = applyGuestFirewallAllow(st)` |
| `cmd/netductor-agent/guest_from_template.go` | 139 | ERROR-IGNORE | `_ = applyGuestFirewallAllow(st)` |
| `cmd/netductor-op/tui_model.go` | 204 | ERROR-IGNORE | `_ = saveTUISettings(s)` |
| `cmd/netductor-op/tui_model.go` | 420 | ERROR-IGNORE | `_ = saveTUISettings(m.snapshotSettings())` |
| `cmd/netductor-op/tui_runtime.go` | 82 | NON-ATOMIC | `_ = os.WriteFile("/etc/netductor/node_id", []byte(name+string([]byte{1` |
| `cmd/netductor-op/tui_runtime.go` | 83 | NON-ATOMIC | `_ = os.WriteFile("/etc/hostname", []byte(name+string([]byte{10})), 0o6` |
| `cmd/netductor-op/tui_wizard_apply.go` | 393 | ERROR-IGNORE | `_ = saveTUISettings(s)` |
| `cmd/netductor-op/tui_wizard_apply.go` | 409 | ERROR-IGNORE | `_ = saveTUISettings(s)` |
| `cmd/netductor-op/tui_wizard_apply.go` | 540 | ERROR-IGNORE | `_ = saveTUISettings(s)` |
| `cmd/netductor-op/cli_session.go` | 89 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(dir, "node_session"), []byte(tok+"\n"),` |
| `cmd/netductor-op/cli_project.go` | 231 | ERROR-IGNORE | `_ = writeBuildMeta(dist, "go")` |
| `cmd/netductor-op/cli_project.go` | 263 | NON-ATOMIC | `_ = os.WriteFile(dst, b, 0o755)` |
| `cmd/netductor-op/cli_project.go` | 265 | ERROR-IGNORE | `_ = writeBuildMeta(dist, "rust")` |
| `cmd/netductor-op/cli_project.go` | 282 | ERROR-IGNORE | `_ = writeBuildMeta(dist, "swift-package")` |
| `cmd/netductor-op/cli_project.go` | 294 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(dist, "xcodebuild-list.txt"), out, 0o64` |
| `cmd/netductor-op/cli_project.go` | 302 | ERROR-IGNORE | `_ = writeBuildMeta(dist, "swift-xcode-partial")` |

*...and 522 more*

---

## [LOW] 7 findings

| File | Line | Pattern | Code |
|---|---|---|---|
| `internal/svcpaths/bootstrap.go` | 125 | TODO | `tmpdir=$(mktemp -d /tmp/netductor-wstunnel.XXXXXX)` |
| `internal/secondary/provision.go` | 334 | TODO | `"TD=$(mktemp -d /tmp/nd-join.XXXXXX)\n" +` |
| `internal/secondary/provision.go` | 339 | TODO | `"NEW=$(mktemp /usr/local/bin/.netductor.new.XXXXXX)\n" +` |
| `internal/secondary/provision.go` | 373 | TODO | `"  PREV_TMP=$(mktemp /var/lib/netductor/.netductor.bin.prev.XXXXXX)\n"` |
| `internal/secondary/provision.go` | 386 | TODO | `"    RESTORE=$(mktemp /usr/local/bin/.netductor.restore.XXXXXX)\n" +` |
| `internal/secondary/provision.go` | 473 | TODO | `STAGE=$(mktemp -d "$P/.mtls-stage.XXXXXX")` |
| `internal/secondary/provision.go` | 508 | TODO | `BACKUP=$(mktemp -d "$P/.mtls-backup.XXXXXX")` |

---
*Generated by qwen-tools in sandbox (no CI)*
