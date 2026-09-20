# Code / Security / Refactoring Review — post-0.8.0 (2026-09-20)

Scope: `cmd/`, `internal/`, TG bot, edge agent recovery, secondary plane, NVR tokens, update path.  
~30k LOC Go. Release baseline: **v0.8.0** (`169a2f9`).

---

## 1. Security review

### 1.1 Strengths (keep)

| Area | Assessment |
|------|------------|
| Operator API | `requireSession` + bearer session on mutating edge/NVR admin routes |
| Edge enroll | Bootstrap/recovery auth, rate limit, **pending → approve**, device token after approve |
| Token compare | `subtle.ConstantTimeCompare` on secondary/edge device tokens |
| Secondary cmds | `restart:` **allowlist** of units |
| NVR clip URL | One-shot token + `PathUnderRoot` on issue **and** redeem |
| Secrets files | Many secrets `0o600`; admin id file `0o600` |
| TG operator | Single admin id; claim-first only with explicit bootstrap |
| VPN secrets | Reality keys under secrets layout |

### 1.2 Issues (priority)

#### P0 — fix soon

1. **Agent recovery HTTP binds `:7879` on all interfaces**  
   Comment says “firewall WAN”, but code does not restrict to LAN.  
   **Risk:** if WAN 7879 open, anyone can push a bootstrap/recovery code they phished or brute (codes are 32 hex, harder) and point `SERVER` at attacker (still need valid code for enroll on *your* primary; attacker can set SERVER to phishing host that captures code).  
   **Fix:** bind default to `br-lan` / first private IP, or `127.0.0.1` + optional LAN; reject non-private RemoteAddr; document UCI firewall rule as install step.

2. **Recovery form accepts arbitrary `SERVER` URL** (partially mitigated in this pass: http/https only)  
   Still allows attacker on LAN to set SERVER to internal IPs (SSRF-ish from agent later).  
   **Fix:** optional allowlist host / pin primary host in config `SERVER_PIN=`.

3. **Secondary agent API `:8788` often plain HTTP** (`NETDUCTOR_PLAIN_AGENT` / no mTLS)  
   Token-bearer over cleartext if not on private link.  
   **Mitigation today:** assume VPN/path; **target:** mTLS default when CA present.

#### P1 — should fix

4. **Clip download `/api/nvr/clip?token=` is unauthenticated** by design (capability URL).  
   Token entropy 128-bit, one-shot, path re-checked — OK if API not on public internet.  
   **Risk:** if primary API is reachable from WAN, clips leak via stolen/logged URL.  
   **Policy:** keep API VPN-only; optional bind loopback + reverse proxy.

5. **Approve response returns `device_token` over session API** — fine for admin; ensure admin UI/TG never logs full token in plain chat history longer than needed (TG shows token on approve — operational risk on shared screens).

6. **`CLAIM_FIRST` / empty admin** — bootstrap footgun if left enabled in prod.

7. **Self-update from GitHub** downloads and replaces binaries as root without signature verify (HTTPS + GitHub only).  
   **Improve:** verify `SHA256SUMS` from same release before install.

8. **Hardcoded `ya.ru` SNI fallbacks** still appear in secondary config paths — operational/DPI concern, not classic vuln.

#### P2 — hardening

9. Recovery page no CSRF token (LAN attacker). Low if LAN trusted.  
10. No global HTTP request body limits on all routes (some have LimitReader).  
11. `exec.Command` for systemctl/journal — mostly fixed args; keep forbidding user-controlled unit names (already allowlisted on secondary).  
12. Install still had legacy version strings in places — align to 0.8.0.

### 1.3 Threat model reminders

- **Compromised primary** = full domain (VPN users, edge, NVR paths). Backup encryption + host hardening remain critical.  
- **Compromised edge** = LAN + camera reachability; device token rotation + revoke.  
- **Stolen TG admin phone** = full operator surface — session revoke + admin id rotation runbook.

---

## 2. Code review

### 2.1 Structure

- Clear packages: `edge`, `secondary`, `vpn`, `nvr`, `update`, `sites`.  
- CLI surface large (`cli_*.go`) — acceptable for monolith binary; **risk** of duplicated strings “relay” comments and dead paths.

### 2.2 Correctness / quality

| Item | Note |
|------|------|
| Dual HTTP servers | `:8787` main + `:8788` secondary agent — document ports in RUNBOOK |
| TG `runND` | Shells out to netductor binary — version skew if bot binary updated without netductor |
| Agent version | Now aligned 0.8.0 in source; deployed agents may lag |
| `Newer()` | Semver-ish major.minor.patch — good enough for tags |
| Error handling | Many `_ =` ignores on best-effort notify/node upsert — OK, but secondary hostname upsert failures silent |

### 2.3 Tests

- Good: edge recovery, vpn, update Newer, nvr tokens, paths.  
- Gaps: API handler integration tests; recovery HTTP handler; secondary Heartbeat auth negative tests; PathUnderRoot edge cases.

---

## 3. Refactoring recommendations

### 3.1 High value

1. **`internal/httpapi` shared middleware** — session, body limit, security headers (dedupe `api_*.go`).  
2. **Rename residual Relay\* comments / `ExportSecondaryBundle` file names** already done; scrub docs “relay” operator language.  
3. **Single `component Version` package** — main/agent/tg ldflags one place.  
4. **Release verify helper** — download SHA256SUMS + asset, verify, then install (TG updates path).  
5. **Recovery listener** — `Listen` on interface with private IP only.

### 3.2 Medium

6. Split `cmd/netductor-tg` handlers by domain files (already partial) — reduce `handlers_cb` size.  
7. Edge + sites coupling (`AttachEdge`) via small `inventory` service.  
8. Config struct for agent instead of ad-hoc KEY=VALUE parse in multiple places.

### 3.3 Low / later

9. OpenTelemetry optional metrics (you deferred Prometheus).  
10. Admin UI TypeScript build (currently plain JS — fine for VPN-only admin).

---

## 4. Bilingual / docs

- UPGRADE, EDGE-REINSTALL, handoff, this review: maintain EN + `docs/ru`.  
- Long PLAN/REVIEW stubs in RU pointing to EN — acceptable if marked.

---

## 5. Actions taken in this pass

- Recovery **SERVER** URL: only `http`/`https`, non-empty host.  
- Review doc recorded.  
- Align install version string remnants toward 0.8.0 where found.

## 6. Suggested next engineering sprint

1. LAN-only bind + firewall snippet for `:7879`.  
2. SHA256 verify on self-update.  
3. mTLS default for `:8788`.  
4. Integration test: enroll → pending → approve → heartbeat.  
5. Audit TG approve messages to show **prefix only** of device_token.

## 7. Implemented follow-up (same day)

- Recovery: bind preferred private IP; forbid non-private clients (unless `NETDUCTOR_RECOVERY_ALLOW_ANY=1`); `SERVER_PIN`
- Self-update: verify `SHA256SUMS` before replace
- TG: `e:appr` / `e:deny` handlers; mask `token=` in approve output
- serve: `mtls.EnsureAll` before agent plane listener
- Secondary config SNI fallback `api.vk.me` (not ya.ru)
- Doctor: WARN if CLAIM_FIRST enabled
- httputil: MaxBodyBytes helpers
