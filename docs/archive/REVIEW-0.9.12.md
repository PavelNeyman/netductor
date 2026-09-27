# Full code & security review — v0.9.12

Date: 2026-09-25

## Scope

Operator (Mac), node API, agent plane, recovery, WebUI assets/CSP, TUI/TG parity, deploy paths, i18n, API coverage.

---

## Security model (current)

| Surface | Bind | Auth | Notes |
|---------|------|------|--------|
| `netductor-op serve` | **loopback only** | Op token (header) | CSP `script/style-src 'self'`; assets `/static/*` |
| Node API `:8787` | loopback (refuse public without `API_PUBLIC=1`) | Session Bearer | Mac via SSH `-L` or direct if health already up |
| Agent plane `:8789` | WAN | **mTLS** | enroll limited |
| Plain agent `:8788` | only `PLAIN_AGENT=1` | emergency | doctor WARN |
| Recovery `:8790` | off until **arm**; WAN while armed | Bearer + TTL | by design for DR |
| VPS `/admin` | disabled | `LEGACY_ADMIN_UI=1` only | not product UI |
| SSH | port **52222** default | key after first deploy | password only bootstrap |

### Positive

- ProxyNodeAPI: **loopback-only** target, path `..` blocked, body limit 8MiB
- Tunnel: BatchMode + key; VPN host preferred when reachable
- Session token: **sessionStorage** (not localStorage); file via Load
- Token inject: meta tag only (not into JS source)
- Constant-time token compare on op serve (subtle)
- Host/user validation in deploy paths (ValidHost / shell hygiene)

### Residual risks (accepted / ops)

| Risk | Severity | Mitigation / status |
|------|----------|---------------------|
| Recovery WAN while armed | Med | Short TTL; SSH arm only; doctor WARN |
| LAN camera TLS skip-verify | Low | LAN self-signed |
| Advanced WebUI POST any path | Med | Requires op token + node session; confirm dialog |
| `PLAIN_AGENT` / `API_PUBLIC` / `CLAIM_FIRST` | High if enabled | Keep off in prod; doctor |
| Meta token in HTML | Low | loopback + no-store; tab-local |
| CSSOM `.style` from JS | Low | Same-origin script; no inline attributes |

### No new Critical findings in 0.9.12 asset split

---

## Mac deploy parity: TUI vs Web

| Capability | Web Installer | TUI Wizard |
|------------|---------------|------------|
| Fleet (primary+secondary) | ✅ `/v1/fleet` | ✅ `wizFleet` |
| Primary only | ✅ | ✅ |
| Secondary only | ✅ | ✅ |
| Domain / LE / SNI | ✅ forms | ✅ fleet fields |
| Add-ons (lampac/git) | ✅ checkboxes | ✅ addons wizard (richer: TG token etc.) |
| Credentials collect | ✅ | ✅ (fleet end) |
| OpenWrt agent install | ❌ use TUI | ✅ |
| MikroTik site | ❌ use TUI | ✅ |
| NVR bootstrap | ❌ use TUI | ✅ |
| Day-2 remote CLI tools | via Control API | ✅ Tools → `netductor …` SSH |

**Verdict:** Core **VPS fleet deploy** is shared intent (operator/deploy backend). Web covers VPS fleet; **hardware must reach Web/CLI parity via same Deploy* (OpenWrt edge on Web as of 0.9.13)**. Not a bug — document as intentional.

Password: Installer/Web + TUI first login only; day-2 tunnel = key.

---

## API coverage (operator session)

~116 routes registered. WebUI buttons + forms cover **majority of session APIs** (VPN, nodes, edge day-2, NVR, git, registry, backup, probes, sites, mTLS, secondary ops).

**Intentionally not operator UI (agent-plane / device):**  
`/api/edge/enroll`, `heartbeat`, agent mTLS material, secondary agent backup pull endpoints used by agents.

**Available via Advanced form (no dedicated button):**  
e.g. `edge/import`, `git/workflow`, `nvr/ingest`, `registry/auth-set` bodies, `session/revoke`.

**TG / TUI:** day-2 subsets + CLI mirror; not every rare POST. Deploy not from TG.

---

## Bilingual (EN/RU)

| Surface | Status |
|---------|--------|
| TG | Strong (i18n maps) |
| TUI | Mode/wizard/tools via `TT()`; some English-only chips/doctor |
| WebUI | Tabs, settings, section nav, notes; **form field labels** largely EN |
| CLI | cli18n for top commands; long doctor lines partial |

**Gap:** full WebUI form-label dictionary; residual CLI doctor EN.

---

## Code / architecture notes

1. `internal/operator` + `internal/deploy` = shared deploy use-cases — good.
2. WebUI `app.js` large — maintainable enough after split; optional codegen later.
3. TG is node-side binary; correct for day-2 without Mac.
4. Naming `relay*` residual in few strings — cosmetic.

---

## Refactor backlog (non-blocking)

1. Web form i18n dictionary for all labels  
2. Web Installer: optional OpenWrt/MikroTik thin entry linking to TUI docs  
3. Shared action registry TG+Web (codegen)  
4. Purge residual `relay` strings → `secondary`

---

## Verdict

**Production-ready** for Mac-operated fleet + day-2, with ops discipline on recovery arm and feature flags. Continue in new chat from **AGENT_HANDOFF.md** + this review.
