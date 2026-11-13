# Netductor Code Review

**Date:** 2026-11-13 20:29 UTC
**Files:** 528
**Go:** 1.22.5
**regex:** 612

| Severity | Count |
|---|---|
| HIGH | 22 |
| MED | 583 |
| LOW | 7 |

## [HIGH] 22

| File | Line | Pattern | Code |
|---|---|---|---|
| `cmd/netductor-op/cli_project.go` | 284 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf("cp -a %q/build/release/* %` |
| `cmd/netductor-op/cli_project.go` | 320 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", script)` |
| `cmd/netductor-op/tui_wizard_apply.go` | 558 | SECRET | `return mm.runNetductor("nvr", "cameras", "add", "name="+m.fieldVal("ca` |
| `cmd/netductor-agent/build_role.go` | 183 | EXEC-SHELL | `_ = exec.Command("bash", "-lc", "cd '"+workDir+"' && tar -czf dist.tgz` |
| `cmd/netductor-agent/build_role.go` | 224 | EXEC-SHELL | `cmd = exec.Command("bash", "-lc", "cd '"+src+"' && go test ./... 2>&1;` |
| `cmd/netductor-agent/facts_local.go` | 25 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "awk -F': ' '/^Serial/{print $` |
| `cmd/netductor-agent/facts_local.go` | 76 | EXEC-SHELL | `if matches, _ := exec.Command("sh", "-c", "ls /dev/sd[a-z] 2>/dev/null` |
| `cmd/netductor-agent/facts_local.go` | 82 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "mount 2>/dev/null / grep ' on` |
| `cmd/netductor-agent/facts_local.go` | 89 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", "for d in /sys/block/sd*; do [` |
| `cmd/netductor-agent/facts_local.go` | 150 | EXEC-SHELL | `if out, err := exec.Command("sh", "-c", `uci -q show wireless 2>/dev/n` |
| `cmd/netductor-agent/agent_ops.go` | 602 | EXEC-SHELL | `_ = exec.Command("sh", "-c", "mountpoint -q /mnt/netductor // (block i` |
| `cmd/netductor-agent/agent_ops.go` | 660 | EXEC-SHELL | `out, err := exec.Command("sh", "-c", script).CombinedOutput()` |
| `internal/svcpaths/build_paths.go` | 120 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/build_paths.go` | 123 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/build_paths.go` | 182 | EXEC-SHELL | `_ = exec.Command("bash", "-c", fmt.Sprintf(` |
| `internal/svcpaths/bootstrap.go` | 100 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "apt-get update -qq && DEBIAN_FRONTE` |
| `internal/svcpaths/bootstrap.go` | 138 | EXEC-SHELL | `if out, err := exec.Command("bash", "-c", script).CombinedOutput(); er` |
| `internal/install/apt.go` | 49 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=noninteracti` |
| `internal/install/apt.go` | 83 | EXEC-SHELL | `cmd := exec.Command("bash", args...)` |
| `internal/install/lampac.go` | 75 | EXEC-SHELL | `script := exec.Command("sh", "-c", "curl -fsSL https://get.docker.com ` |
| `internal/aptguard/remediate.go` | 678 | EXEC-SHELL | `cmd := exec.Command("bash", "-c", "export DEBIAN_FRONTEND=noninteracti` |
| `internal/update/build_host.go` | 85 | EXEC-SHELL | `sum := exec.Command("bash", "-c", "sha256sum netductor-* > SHA256SUMS"` |

## [MED] 583

| File | Line | Pattern | Code |
|---|---|---|---|
| `cmd/netductor-tg/handlers_edge_guest.go` | 142 | ERROR-IGNORE | `_ = deleteMessage(token, chat, promptID)` |
| `cmd/netductor-tg/handlers_edge_guest.go` | 145 | ERROR-IGNORE | `_ = deleteMessage(token, chat, userMsgID)` |
| `cmd/netductor-tg/handlers_extra.go` | 949 | ERROR-IGNORE | `_ = sendPhotoBytes(token, chat, data, fmt.Sprintf("%s/%s #%d", siteID,` |
| `cmd/netductor-tg/handlers_cb.go` | 31 | ERROR-IGNORE | `_, _ = apiPost(token, "answerCallbackQuery", map[string]any{` |
| `cmd/netductor-tg/handlers_cb.go` | 465 | ERROR-IGNORE | `_ = runND("vpn", "session", "revoke-all")` |
| `cmd/netductor-tg/handlers_cb.go` | 519 | ERROR-IGNORE | `_ = runND("ssh-hosts", "clear", "--kind", "mt")` |
| `cmd/netductor-tg/handlers_cb.go` | 522 | ERROR-IGNORE | `_ = runND("ssh-hosts", "clear", "--kind", "secondary")` |
| `cmd/netductor-tg/handlers_cb.go` | 531 | ERROR-IGNORE | `_ = runND("api-public", "disarm")` |
| `cmd/netductor-tg/handlers_cb.go` | 533 | ERROR-IGNORE | `_ = runND("api-public", "arm", "--ttl", "15m")` |
| `cmd/netductor-tg/handlers_cb.go` | 540 | ERROR-IGNORE | `_ = runND("svc-paths", "failover", "set-users-sp", "off")` |
| `cmd/netductor-tg/handlers_cb.go` | 542 | ERROR-IGNORE | `_ = runND("svc-paths", "failover", "set-users-sp", "on")` |
| `cmd/netductor-tg/widget_dual.go` | 67 | ERROR-IGNORE | `_ = deleteMessage(token, chat, p.InfoID)` |
| `cmd/netductor-tg/widget_dual.go` | 70 | ERROR-IGNORE | `_ = deleteMessage(token, chat, p.NavID)` |
| `cmd/netductor-tg/widget_dual.go` | 86 | ERROR-IGNORE | `_ = deleteMessage(token, chat, infoID)` |
| `cmd/netductor-tg/widget_dual.go` | 160 | ERROR-IGNORE | `_ = deleteMessage(token, chat, id)` |
| `cmd/netductor-tg/widget_dual.go` | 191 | ERROR-IGNORE | `_ = deleteMessage(token, chat, old)` |
| `cmd/netductor-tg/widget_dual.go` | 232 | ERROR-IGNORE | `p, _ = getDual(chat)` |
| `cmd/netductor-tg/widget_dual.go` | 236 | ERROR-IGNORE | `p, _ = getDual(chat)` |
| `cmd/netductor-tg/format_sites.go` | 11 | ERROR-IGNORE | `_ = runND("secondary", "export", "-o", "/tmp/nd-secondary-bundle.json"` |
| `cmd/netductor-tg/widget_core.go` | 298 | ERROR-IGNORE | `navCD, _ = appMarshal(appState{S: screenID, T: screenID, Acts: "r"})` |
| `cmd/netductor-tg/widget_core.go` | 410 | ERROR-IGNORE | `_ = editWidgetScreenData(token, q.Chat.ID, q.MessageID, screenID)` |
| `cmd/netductor-tg/i18n.go` | 833 | NON-ATOMIC | `_ = os.WriteFile(langFile, []byte(l+"\n"), 0644)` |
| `cmd/netductor-tg/widget_persist.go` | 92 | NON-ATOMIC | `if err := os.WriteFile(tmp, b, 0o600); err != nil {` |
| `cmd/netductor-tg/widget_lists.go` | 363 | ERROR-IGNORE | `_ = editWidgetScreenData(token, q.Chat.ID, q.MessageID, screen)` |
| `cmd/netductor-tg/format_nodes.go` | 14 | ERROR-IGNORE | `_ = runND("secondary", "status")` |
| `cmd/netductor-tg/widget_app_shell.go` | 58 | ERROR-IGNORE | `_ = deleteMessage(token, chat, mid)` |
| `cmd/netductor-tg/main.go` | 36 | NON-ATOMIC | `_ = os.WriteFile(chatFile, []byte(fmt.Sprintf("%d\n", chatID)), 0o600)` |
| `cmd/netductor-tg/main.go` | 270 | ERROR-IGNORE | `_, _ = apiPost(token, "deleteMessage", map[string]any{` |
| `cmd/netductor-tg/main.go` | 425 | ERROR-IGNORE | `_ = deleteMessage(token, chat, msgID)` |
| `cmd/netductor-tg/main.go` | 443 | ERROR-IGNORE | `_ = deleteMessage(token, chat, msgID)` |
| `cmd/netductor-tg/main.go` | 449 | ERROR-IGNORE | `_ = deleteMessage(token, chat, msgID)` |
| `cmd/netductor-tg/main.go` | 493 | ERROR-IGNORE | `_ = deleteMessage(token, chat, target)` |
| `cmd/netductor-tg/main.go` | 496 | ERROR-IGNORE | `_ = deleteMessage(token, chat, msgID)` |
| `cmd/netductor-tg/main.go` | 510 | ERROR-IGNORE | `_ = deleteMessage(token, chat, userMsgID)` |
| `cmd/netductor-tg/main.go` | 513 | ERROR-IGNORE | `_ = deleteMessage(token, chat, pid)` |
| `cmd/netductor-tg/main.go` | 567 | ERROR-IGNORE | `_ = deleteMessage(token, chat, h.MessageID)` |
| `cmd/netductor-tg/main.go` | 788 | ERROR-IGNORE | `_, _ = apiPost(token, "answerCallbackQuery", map[string]any{"callback_` |
| `cmd/netductor-tg/main.go` | 792 | ERROR-IGNORE | `_, _ = apiPost(token, "answerCallbackQuery", map[string]any{` |
| `cmd/netductor-tg/main.go` | 934 | ERROR-IGNORE | `_, _ = apiPost(token, "setMyCommands", map[string]any{"commands": cmds` |
| `cmd/netductor-tg/main.go` | 1056 | ERROR-IGNORE | `_ = handleWidgetQuery(token, u.WidgetQuery, admin)` |
| `cmd/netductor-tg/handlers_reboot.go` | 94 | ERROR-IGNORE | `_ = enqueueNodeCmd(id, "reboot")` |
| `cmd/netductor-tg/handlers_msg.go` | 232 | ERROR-IGNORE | `_ = runND("nodes", "sync-local")` |
| `cmd/netductor-tg/handlers_msg.go` | 517 | ERROR-IGNORE | `_ = deleteMessage(token, chat, uid)` |
| `cmd/netductor-tg/handlers_msg.go` | 522 | ERROR-IGNORE | `_ = deleteMessage(token, chat, hub.MessageID)` |
| `cmd/netductor-tg/handlers_msg.go` | 549 | ERROR-IGNORE | `_ = deleteMessage(token, chat, uid)` |
| `cmd/netductor-tg/handlers_msg.go` | 559 | ERROR-IGNORE | `_ = deleteMessage(token, chat, uid)` |
| `cmd/netductor-tg/handlers_msg.go` | 567 | ERROR-IGNORE | `_ = deleteMessage(token, chat, uid)` |
| `cmd/netductor-tg/handlers_cb_nodes.go` | 30 | ERROR-IGNORE | `_ = enqueueNodeCmd(id, "upgrade")` |
| `cmd/netductor-tg/widget_cards.go` | 310 | ERROR-IGNORE | `_ = enqueueNodeCmd(id, "upgrade")` |
| `cmd/netductor-tg/widget_app.go` | 1678 | ERROR-IGNORE | `_ = runND("api-public", "disarm")` |
| `cmd/netductor-tg/widget_app.go` | 1680 | ERROR-IGNORE | `_ = runND("api-public", "arm", "--ttl", "15m")` |
| `cmd/netductor-tg/widget_app.go` | 1689 | ERROR-IGNORE | `_ = runND("svc-paths", "failover", "set-users-sp", "off")` |
| `cmd/netductor-tg/widget_app.go` | 1691 | ERROR-IGNORE | `_ = runND("svc-paths", "failover", "set-users-sp", "on")` |
| `cmd/netductor-tg/widget_app.go` | 1978 | ERROR-IGNORE | `_ = runVPN("enable", focus)` |
| `cmd/netductor-tg/widget_app.go` | 1987 | ERROR-IGNORE | `_ = runVPN("disable", focus)` |
| `cmd/netductor-tg/widget_app.go` | 2183 | ERROR-IGNORE | `_ = runVPN("rename", focus, draft)` |
| `cmd/netductor-tg/widget_app.go` | 2191 | ERROR-IGNORE | `_ = runVPN("add", draft)` |
| `cmd/netductor/api_jobs.go` | 36 | ERROR-IGNORE | `_ = primaryJobStore()` |
| `cmd/netductor/cli_edge.go` | 205 | ERROR-IGNORE | `_ = sitesAttach(site, did)` |
| `cmd/netductor/cli_edge.go` | 246 | NON-ATOMIC | `if err := os.WriteFile(out, append(b, 10), 0o600); err != nil {` |

## [LOW] 7

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
*staticcheck + gosec: цепочка в фоне (mod→sc→gosec), результаты в следующем вызове*
