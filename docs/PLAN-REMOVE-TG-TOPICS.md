# Plan: Remove Telegram private-topics functionality

**Status:** implemented (0.9.269)  
**Goal:** Delete forum/private **topics** support from netductor-tg / notify.  
**Must not break:** alerts **channel** (`telegram_alerts_chat_id`), operator **DM menu**, VPN cards, QR, SR Config, batch alerts.

**Context (operator decision 2026-10-06):** topics are awkward; alerts live in a dedicated channel; menu lives in plain private chat after 0.9.265+. Topics code is dead weight and still causes edge bugs (stale Control thread, hub re-pin, BotFather Threaded Mode).

---

## 1. Target architecture (after removal)

| Surface | Destination | Mechanism |
|--|--|--|
| Operator menu, tools, user cards, Access, SR Config | **Private chat** with admin (`telegram_admin_id`) | `sendMessage` / `sendRichMessage` **without** `message_thread_id` / `direct_messages_topic_id` |
| QR photos / documents | Same private chat | `sendPhoto` / `sendDocument` / rich photo **without** thread ids |
| Alerts, warnings, service, updates (batched) | **Channel** `telegram_alerts_chat_id` | `sendMessage` to channel only; **no** thread id |
| If channel **not** configured | Admin DM (same as menu) | Fallback; warn in `tg-alerts status` / doctor |

**Explicit non-goals:**

- Do **not** remove channel support (`internal/notify/alerts_channel.go`, known_chats, `/alerts_chat`, TG Channel UI).
- Do **not** require BotFather Threaded Mode.
- Do **not** auto-create forum topics on bot start.

---

## 2. Inventory — code to remove or gut

### 2.1 Core package (largest)

| Path | Action |
|--|--|
| `internal/notify/topics.go` | **Delete file** (or reduce to thin stubs that always return 0 / no-op for one release, then delete). |
| Functions | `EnsureTopics`, `ReconcileTopics`, `ForceRecreateTopics`, `SeedAllTopics`, `createForumTopic`, `topicAlive`, `seedTopicMessage`, `AssignTopic`, `ListTopics`, `TopicsStatusHTML`, `ThreadID`, `ThreadForAlertKey`, `ResolveThreadIDForMessage`, `ApplyThread`, `MenuThread`, `MediaThread`, `defaultTopics`, `TopicKeys`, topics.json load/save/backup |
| Keep related but non-topic | `SaveHubMsg` / `LoadHubMsg` / `ClearHubMsg` / hub re-pin live in `batch.go` — **keep hub optional** for menu singleton in DM, **without** thread |

### 2.2 Call sites in notify

| Path | Change |
|--|--|
| `internal/notify/batch.go` `sendTelegramHTML` | Remove all `message_thread_id` / `direct_messages_topic_id`. Channel mode already prefers `sendMessage` to `AlertsChatID()`. |
| `internal/notify/batch.go` `FlushAlerts` | Drop `ResolveThreadIDForMessage` / `ThreadID("alerts")`. Destination = `AlertsChatID()` or admin. |
| `internal/notify/batch.go` `repinHub` | Keep only if still useful in **DM**; never set thread. Prefer skip hub re-pin when channel mode (already partially done). |
| `internal/notify/telegram.go` | Strip thread fields from any remaining send helpers. |
| `internal/notify/alerts_channel.go` | Remove `DisablePrivateTopics` coupling; modes: `channel` \| `admin_only` only (drop `topics`). Status HTML: no topic hints. |
| `TopicsForMenuEnabled` / `DisablePrivateTopics` / `topics_disabled` file | **Delete** — always “flat DM”. |

### 2.3 Bot binary `cmd/netductor-tg`

| Area | Change |
|--|--|
| `main.go` `ensureTopicsOnce` | **Delete** entire function and all calls. |
| `main.go` `sendRich` / `sendDocumentFile` / `sendRichWithPhoto` | Remove `notify.ApplyThread(..., "menu"|"media")` and MediaThread/MenuThread branches. |
| `handlers_cb.go` | Remove cases: `m:topics`, `m:topics:reconcile`, `m:topics:recreate`, `m:topics:flat`. Keep `m:alerts-chat*` (channel UI). |
| `keyboards.go` | Remove Topics button from main/tools; remove `topicsKeyboard()` or reduce to channel-only screen. |
| `i18n.go` | Remove `btn_topics` and topic-only strings; keep channel strings. |
| `handlers_msg.go` | Remove `/topic` command; keep `/alerts_chat`. |
| Access / QR notes | Remove “QR → Media topic” copy entirely (already gated; delete). |

### 2.4 CLI / API / catalog

| Path | Change |
|--|--|
| Any `netductor topic*` CLI if exists | Remove. |
| opcatalog entries for topics reconcile/recreate | Remove. |
| Doctor checks for “topics reconciled” | Remove or replace with “alerts channel configured?”. |

### 2.5 Docs

| Path | Action |
|--|--|
| `docs/TG-ALERTS-TOPIC.md` | **Delete** or replace with one-line “removed; use channel”. |
| `docs/TG-MEDIA-TOPICS.md` | **Delete**. |
| `docs/TG-ALERTS-CHANNEL.md` | **Canonical**; remove “fallback to topics” section; say channel required for ops alerts (or soft-fallback to admin DM). |
| `docs/RECOVER-BOT.md`, `AGENT_HANDOFF.md`, `VERIFICATION-PLAN.md` | Strip topics steps. |
| `docs/ru/*` twins | Same. |

### 2.6 On-disk state (migration, not code)

On first start after upgrade (or one-shot CLI):

```text
/var/lib/netductor/tg/topics.json          → delete or ignore
/var/lib/netductor/tg/topics-backup.json   → delete or ignore
/var/lib/netductor/tg/topics_disabled      → delete (obsolete)
```

**Do not delete:**

```text
/etc/netductor/secrets/telegram_alerts_chat_id
/etc/netductor/secrets/telegram_bot_token
/etc/netductor/secrets/telegram_admin_id
/var/lib/netductor/tg/known_chats.json     → keep (channel pick UX)
/var/lib/netductor/tg/hub.json             → optional; clear on migrate to avoid stale edit targets
```

Optional migrate helper:

```go
// notify.MigrateAwayFromTopics() — remove topics*.json, ClearHubMsg, log once
```

Call from bot `main` once (version gate or empty file marker `tg/topics_removed`).

---

## 3. Behaviour matrix (regression checklist)

| Scenario | Expected |
|--|--|
| Channel set, `/menu` | Full menu in **admin DM**, no thread errors in bot logs |
| Channel set, stack apply / firewall alert | Message appears in **channel** only |
| Channel set, Access + QR + SR Config | Files/cards in **DM** with Menu/back keyboards |
| Channel **cleared** | Alerts go to admin DM; status shows `admin_only`; no panic |
| BotFather Threaded Mode ON or OFF | Both work; no createForumTopic calls |
| Old `topics.json` still on disk | Ignored; no EnsureTopics |
| `tg-alerts test` | Lands in channel if set; reports error body if not |
| Secondary / node alerts via notify | Same batch path → channel |

---

## 4. Implementation order (safe sequence)

1. **Lock channel path**  
   - Confirm `AlertsChatID()` + `sendTelegramHTML` channel branch is the only alert egress.  
   - Add test: mock/postTG not required; unit test that when `telegram_alerts_chat_id` set, threadID forced 0 and chat = channel.

2. **Stop writing threads (behaviour first)**  
   - Make `ApplyThread` / `MenuThread` / `MediaThread` no-ops (return 0).  
   - Make `EnsureTopics` / `ReconcileTopics` / `ForceRecreateTopics` return nil immediately.  
   - Ship as small release if needed (“topics disabled in code”).

3. **Remove UI entry points**  
   - Drop Topics buttons/commands so operators cannot recreate topics.

4. **Delete topics.go and dead imports**  
   - `go test ./internal/notify/ ./cmd/netductor-tg/ ./cmd/netductor/`  
   - Fix compile errors only by deletion/stubs, not by restoring topics.

5. **Migrate on-disk + docs**  
   - One-shot cleanup of topics files.  
   - Docs: channel-only narrative.

6. **Manual smoke (primary)**  
   ```bash
   netductor stack apply vX.Y.Z
   systemctl restart netductor-telegram-bot
   # DM: /menu force
   netductor tg-alerts test
   # Trigger real alert if possible (or doctor warn)
   # Users → Access → SR Config — must have Menu
   ```

---

## 5. Invariants (do not violate)

1. **`telegram_alerts_chat_id` remains the alerts sink** when present.  
2. **Admin DM never receives alert batches when channel is set** (menu-only).  
3. **No `message_thread_id` on outbound API calls** after removal (grep CI or `rg message_thread_id`).  
4. **No BotFather Threaded Mode requirement** in docs or doctor.  
5. **Known-chats / forward-to-bind channel UX stays**.

---

## 6. Grep acceptance (post-change)

Must return **no production code** hits (tests/docs historical OK):

```bash
rg -n 'message_thread_id|direct_messages_topic_id|EnsureTopics|ForceRecreateTopics|createForumTopic|topics\.json|ApplyThread|MenuThread|MediaThread' \
  --type go internal/notify cmd/netductor-tg cmd/netductor
```

Allowed residual: comments in CHANGELOG / this plan.

---

## 7. Risk notes

| Risk | Mitigation |
|--|--|
| Operator still has Threaded Mode DM and expects Control topic | Document: menu is General/DM only; `/menu force` |
| Alerts without channel configured | Fallback admin DM + status `admin_only` |
| Hub edit fails after migrate | `ClearHubMsg` on migrate; force new menu message |
| Partial deploy (old tg binary) | stack apply replaces `netductor-tg`; restart unit |

---

## 8. Out of scope

- Redesign of alert batching / hub re-pin policy (can simplify later).  
- Multiple alert channels.  
- Forum topics **inside the alerts channel** (not used today).  
- Hardware e2e.

---

## 9. Suggested commit series

1. `fix(tg): no-op topics routing (always flat DM)`  
2. `refactor(tg): remove Topics UI and /topic`  
3. `refactor(notify): delete topics.go; channel-only alert routing`  
4. `docs: remove TG topics guides; channel is canonical`  
5. `chore(tg): migrate delete topics.json on start`

Version bump once at the end or per step; always run `scripts/release.sh` (or equivalent) so `SHA256SUMS` exists before `stack apply`.

---

## 10. Owner checklist when executing

- [ ] Channel id present on primary: `netductor tg-alerts status` → `channel_set=true`  
- [ ] Implement steps 1–5  
- [ ] Grep acceptance §6 clean  
- [ ] Smoke §4.6  
- [ ] Mark this plan **done** in OPEN_ITEMS / CHANGELOG  

**Do not** implement partial topic “compatibility mode” long-term — full removal is the goal.
