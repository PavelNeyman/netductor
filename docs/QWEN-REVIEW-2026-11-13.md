# Netductor Deep Review — Automated Analysis

**Date:** 2026-11-13 19:14 UTC
**HEAD:** c75740678a15491cf841b92dfb1200a9d5f755d1
**Files analyzed:** 1056
**Total findings:** 1282
**Tool:** qwen regex analyzer (staticcheck/gosec pending)

---

## Summary

| Severity | Count |
|----------|-------|
| HIGH | 64 |
| MED | 1204 |
| LOW | 14 |

| Pattern | Count | Description |
|---------|-------|-------------|
| ERROR-IGNORE | 612 | Error explicitly ignored with _ |
| NON-ATOMIC | 554 | os.WriteFile without tmp+rename (crash risk) |
| EXEC-SHELL | 42 | exec.Command with bash/sh (injection risk) |
| HTTP-NO-TIMEOUT | 38 | HTTP client without timeout |
| EXEC-SPRINTF | 20 | exec.Command + fmt.Sprintf (injection risk) |
| TODO | 14 | TODO/FIXME/HACK comment |
| SECRET | 2 | Potential hardcoded secret |

---

## HIGH Severity Findings

| File | Line | Pattern | Code |
|---|---|---|---|
| `cmd/netductor-op/tui_wizard_apply.go` | 558 | SECRET | `return mm.runNetductor("nvr", "cameras", "add", "name="+m.fi` |
| `cmd/netductor-op/cli_project.go` | 284 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf("cp -a %q/build/r` |
| `cmd/netductor-op/cli_project.go` | 284 | EXEC-SPRINTF | `_ = exec.Command("bash", "-c", fmt.Sprintf("cp -a %q/build/r` |
| `cmd/netductor-op/cli_project.go` | 320 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", script)` |
| `cmd/netductor-agent/agent_ops.go` | 602 | EXEC-SHELL | `_ = exec.Command("sh", "-c", "mountpoint -q /mnt/netductor \` |
| `cmd/netductor-agent/agent_ops.go` | 660 | EXEC-SHELL | `out, err := exec.Command("sh", "-c", script).CombinedOutput(` |
| `cmd/netductor-agent/facts_local.go` | 25 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "awk -F': ' '/^Seria` |
| `cmd/netductor-agent/facts_local.go` | 76 | EXEC-SHELL | `if matches, _ := exec.Command("sh", "-c", "ls /dev/sd[a-z] 2` |
| `cmd/netductor-agent/facts_local.go` | 82 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "mount 2>/dev/null \` |
| `cmd/netductor-agent/facts_local.go` | 89 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "for d in /sys/block` |
| `cmd/netductor-agent/facts_local.go` | 150 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", `uci -q show wireles` |
| `cmd/netductor-agent/build_role.go` | 183 | EXEC-SHELL | `_ = exec.Command("bash", "-lc", "cd '"+workDir+"' && tar -cz` |
| `cmd/netductor-agent/build_role.go` | 224 | EXEC-SHELL | `cmd = exec.Command("bash", "-lc", "cd '"+src+"' && go test .` |
| `netductor-main/cmd/netductor-op/tui_wizard_apply.go` | 558 | SECRET | `return mm.runNetductor("nvr", "cameras", "add", "name="+m.fi` |
| `netductor-main/cmd/netductor-op/cli_project.go` | 284 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf("cp -a %q/build/r` |
| `netductor-main/cmd/netductor-op/cli_project.go` | 284 | EXEC-SPRINTF | `_ = exec.Command("bash", "-c", fmt.Sprintf("cp -a %q/build/r` |
| `netductor-main/cmd/netductor-op/cli_project.go` | 320 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", script)` |
| `netductor-main/cmd/netductor-agent/agent_ops.go` | 602 | EXEC-SHELL | `_ = exec.Command("sh", "-c", "mountpoint -q /mnt/netductor \` |
| `netductor-main/cmd/netductor-agent/agent_ops.go` | 660 | EXEC-SHELL | `out, err := exec.Command("sh", "-c", script).CombinedOutput(` |
| `netductor-main/cmd/netductor-agent/facts_local.go` | 25 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "awk -F': ' '/^Seria` |
| `netductor-main/cmd/netductor-agent/facts_local.go` | 76 | EXEC-SHELL | `if matches, _ := exec.Command("sh", "-c", "ls /dev/sd[a-z] 2` |
| `netductor-main/cmd/netductor-agent/facts_local.go` | 82 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "mount 2>/dev/null \` |
| `netductor-main/cmd/netductor-agent/facts_local.go` | 89 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "for d in /sys/block` |
| `netductor-main/cmd/netductor-agent/facts_local.go` | 150 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", `uci -q show wireles` |
| `netductor-main/cmd/netductor-agent/build_role.go` | 183 | EXEC-SHELL | `_ = exec.Command("bash", "-lc", "cd '"+workDir+"' && tar -cz` |
| `netductor-main/cmd/netductor-agent/build_role.go` | 224 | EXEC-SHELL | `cmd = exec.Command("bash", "-lc", "cd '"+src+"' && go test .` |
| `netductor-main/internal/install/apt.go` | 49 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=no` |
| `netductor-main/internal/install/apt.go` | 83 | EXEC-SHELL | `cmd := exec.Command("bash", args...)` |
| `netductor-main/internal/install/ssh_harden.go` | 105 | EXEC-SPRINTF | `_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", port)` |
| `netductor-main/internal/install/lampac.go` | 75 | EXEC-SHELL | `script := exec.Command("sh", "-c", "curl -fsSL https://get.d` |
| `netductor-main/internal/probes/run.go` | 19 | EXEC-SPRINTF | `out, err := exec.Command("ss", flag, "sport", "=", fmt.Sprin` |
| `netductor-main/internal/git/git.go` | 189 | EXEC-SPRINTF | `cmd := exec.Command("git", "-C", dir, "log", fmt.Sprintf("-%` |
| `netductor-main/internal/svcpaths/build_paths.go` | 120 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `netductor-main/internal/svcpaths/build_paths.go` | 120 | EXEC-SPRINTF | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `netductor-main/internal/svcpaths/build_paths.go` | 123 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `netductor-main/internal/svcpaths/build_paths.go` | 123 | EXEC-SPRINTF | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `netductor-main/internal/svcpaths/build_paths.go` | 182 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `netductor-main/internal/svcpaths/build_paths.go` | 182 | EXEC-SPRINTF | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `netductor-main/internal/svcpaths/bootstrap.go` | 100 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "apt-get update -qq && DEB` |
| `netductor-main/internal/svcpaths/bootstrap.go` | 138 | EXEC-SHELL | `if out, err := exec.Command("bash", "-c", script).CombinedOu` |
| `netductor-main/internal/update/build_host.go` | 85 | EXEC-SHELL | `sum := exec.Command("bash", "-c", "sha256sum netductor-* > S` |
| `netductor-main/internal/aptguard/remediate.go` | 678 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=no` |
| `netductor-main/internal/vpn/uplink_ab.go` | 711 | EXEC-SPRINTF | `_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", p), "` |
| `netductor-main/internal/vpn/uplink_ab.go` | 714 | EXEC-SPRINTF | `_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/udp", cfg.W` |
| `netductor-main/internal/vpn/uplink_ab.go` | 719 | EXEC-SPRINTF | `_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", p.Por` |
| `internal/install/apt.go` | 49 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=no` |
| `internal/install/apt.go` | 83 | EXEC-SHELL | `cmd := exec.Command("bash", args...)` |
| `internal/install/ssh_harden.go` | 105 | EXEC-SPRINTF | `_ = exec.Command("ufw", "allow", fmt.Sprintf("%d/tcp", port)` |
| `internal/install/lampac.go` | 75 | EXEC-SHELL | `script := exec.Command("sh", "-c", "curl -fsSL https://get.d` |
| `internal/probes/run.go` | 19 | EXEC-SPRINTF | `out, err := exec.Command("ss", flag, "sport", "=", fmt.Sprin` |

... and 14 more HIGH findings

---

## MED Severity Findings (top 30)

| File | Line | Pattern | Code |
|---|---|---|---|
| `cmd/netductor-op/cli_session.go` | 89 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(dir, "node_session"), []byte(` |
| `cmd/netductor-op/job_client.go` | 57 | HTTP-NO-TIMEOUT | `res, err := http.DefaultClient.Do(req)` |
| `cmd/netductor-op/job_client.go` | 83 | HTTP-NO-TIMEOUT | `res, err := http.DefaultClient.Do(req)` |
| `cmd/netductor-op/tui_wizard_apply.go` | 393 | ERROR-IGNORE | `_ = saveTUISettings(s)` |
| `cmd/netductor-op/tui_wizard_apply.go` | 409 | ERROR-IGNORE | `_ = saveTUISettings(s)` |
| `cmd/netductor-op/tui_wizard_apply.go` | 540 | ERROR-IGNORE | `_ = saveTUISettings(s)` |
| `cmd/netductor-op/cli_project.go` | 231 | ERROR-IGNORE | `_ = writeBuildMeta(dist, "go")` |
| `cmd/netductor-op/cli_project.go` | 263 | NON-ATOMIC | `_ = os.WriteFile(dst, b, 0o755)` |
| `cmd/netductor-op/cli_project.go` | 265 | ERROR-IGNORE | `_ = writeBuildMeta(dist, "rust")` |
| `cmd/netductor-op/cli_project.go` | 282 | ERROR-IGNORE | `_ = writeBuildMeta(dist, "swift-package")` |
| `cmd/netductor-op/cli_project.go` | 294 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(dist, "xcodebuild-list.txt"),` |
| `cmd/netductor-op/cli_project.go` | 302 | ERROR-IGNORE | `_ = writeBuildMeta(dist, "swift-xcode-partial")` |
| `cmd/netductor-op/cli_project.go` | 305 | ERROR-IGNORE | `_ = writeBuildMeta(dist, "swift-xcode")` |
| `cmd/netductor-op/cli_project.go` | 329 | NON-ATOMIC | `_ = os.WriteFile(filepath.Join(dist, "shell-ok.txt"), b, 0o6` |
| `cmd/netductor-op/cli_project.go` | 330 | ERROR-IGNORE | `_ = writeBuildMeta(dist, "shell")` |
| `cmd/netductor-op/cli_project.go` | 336 | NON-ATOMIC | `return os.WriteFile(filepath.Join(dist, "build-meta.txt"), [` |
| `cmd/netductor-op/tui_forms.go` | 131 | ERROR-IGNORE | `_ = saveTUISettings(m.snapshotSettings())` |
| `cmd/netductor-op/tui_runtime.go` | 82 | NON-ATOMIC | `_ = os.WriteFile("/etc/netductor/node_id", []byte(name+strin` |
| `cmd/netductor-op/tui_runtime.go` | 83 | NON-ATOMIC | `_ = os.WriteFile("/etc/hostname", []byte(name+string([]byte{` |
| `cmd/netductor-op/tui.go` | 235 | ERROR-IGNORE | `mode, _ = detectSuggestedMode()` |
| `cmd/netductor-op/cli_jobs.go` | 93 | HTTP-NO-TIMEOUT | `res, err := http.DefaultClient.Do(req)` |
| `cmd/netductor-op/cli_jobs.go` | 113 | HTTP-NO-TIMEOUT | `res, err := http.DefaultClient.Do(req)` |
| `cmd/netductor-op/tui_model.go` | 204 | ERROR-IGNORE | `_ = saveTUISettings(s)` |
| `cmd/netductor-op/tui_model.go` | 420 | ERROR-IGNORE | `_ = saveTUISettings(m.snapshotSettings())` |
| `cmd/netductor-tg/main.go` | 36 | NON-ATOMIC | `_ = os.WriteFile(chatFile, []byte(fmt.Sprintf("%d\n", chatID` |
| `cmd/netductor-tg/main.go` | 270 | ERROR-IGNORE | `_, _ = apiPost(token, "deleteMessage", map[string]any{` |
| `cmd/netductor-tg/main.go` | 327 | HTTP-NO-TIMEOUT | `resp, err := http.DefaultClient.Do(req)` |
| `cmd/netductor-tg/main.go` | 396 | HTTP-NO-TIMEOUT | `resp, err := http.DefaultClient.Do(req)` |
| `cmd/netductor-tg/main.go` | 425 | ERROR-IGNORE | `_ = deleteMessage(token, chat, msgID)` |
| `cmd/netductor-tg/main.go` | 443 | ERROR-IGNORE | `_ = deleteMessage(token, chat, msgID)` |

... and 1174 more MED findings

---

## Methodology

- **Regex patterns:** 7 patterns covering injection, atomicity, error handling, secrets, HTTP timeouts
- **Coverage:** 1056 non-test .go files
- **Pending:** staticcheck + gosec deep analysis (downloading in background)

## Next Steps

1. Run staticcheck for real bug detection (nil deref, dead code, API misuse)
2. Run gosec for security analysis (crypto, path traversal, SSRF)
3. Merge all findings into unified report
4. Prioritize fixes by severity and exploitability

---

*Generated by qwen automated review pipeline*
