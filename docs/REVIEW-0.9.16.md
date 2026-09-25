# Full code & security review — v0.9.16 / 0.9.17

Date: 2026-09-25  
Scope: operator (`netductor-op`), node (`netductor`), agent, TG, deploy paths, Web/TUI parity, i18n, API surface.

## Architecture (locked)

```
Web / TUI / CLI  →  internal/operator + internal/deploy
Day-2 Control    →  SSH tunnel → node :8787 session API
Edge / secondary →  mTLS :8789
TG               →  day-2 on node only (not fleet installer)
```

| Binary | Role |
|--------|------|
| netductor-op | Mac: WebUI, TUI, deploy, tunnel, session issue |
| netductor | Linux node API + VPN + CLI |
| netductor-agent | OpenWrt edge |
| netductor-tg | Telegram addon |

## Mac deploy scenario — both UIs

| Step | Backend | Web | TUI |
|------|---------|-----|-----|
| Primary VPS | `DeployPrimary` | Fleet / Primary | Wizard primary / fleet |
| Secondary VPS | `DeploySecondary` Mac-direct | Fleet / Secondary | Wizard secondary / fleet |
| OpenWrt | `DeployEdge` | `/v1/edge` | Wizard openwrt |
| MikroTik site | `DeploySite` | `/v1/site` | Wizard mikrotik |
| Credentials dump | collect secrets | Credentials | Tools / post-deploy |
| Telegram on primary | secrets + install | Fleet/Primary fields | Add-ons / primary fields |
| Lampac / Git | flags | checkboxes | Add-ons checklist |
| Domain + LE | domain set / tls le | domain_base + le_email | same |

**Verdict:** Installer paths are shared-backend parity. Residual: NVR **wizard** is TUI-first (day-2 NVR is on Web Control); technical Control form field names (`device_id`, …) remain English by design (API identifiers).

## API coverage

### Operator (`netductor-op` local)

`/v1/fleet|primary|secondary|edge|site|mikrotik|credentials|tunnel/*|session/*|node/*|health|meta` — complete for product installer + Control proxy.

### Node session API (~100 routes)

- **Web Control:** ~54 explicit GET/POST + Advanced generic for any path + forms for VPN/edge/NVR/git/nodes.
- **TUI Tools:** remote CLI / same APIs.
- **Not in Control (by design):** agent-only (`/api/edge/heartbeat`, enroll material), secondary agent internal, public `/profiles/`, `/r` redirect.
- **Added 0.9.16:** `/api/doctor`, `/api/domain`.

Full path list: generate with `rg 'HandleFunc' cmd/netductor`.

## Security review

### Strengths

| Area | Status |
|------|--------|
| VPS `/admin` | Off unless `NETDUCTOR_LEGACY_ADMIN_UI=1` |
| Node API :8787 | Loopback default; Mac via SSH -L |
| Agent :8789 | mTLS required |
| Session tokens | SHA-256 hashed on disk; plaintext once; sessionStorage on Mac |
| Operator local token | Constant-time compare |
| SSH day-2 | Key-only after harden (52222) |
| Deploy secrets | 0700/0600; operator credentials tar on Mac |
| Recovery :8790 | Off until `recovery arm`; TTL; TLS default |
| CSP / COOP | Operator WebUI |
| Doctor footguns | PLAIN_AGENT / API_PUBLIC / CLAIM_FIRST fail or warn |

### Residual risks (accepted / ops)

1. **Recovery WAN while armed** — short TTL; operator must disarm.
2. **Advanced Control** — full session API power (confirm on POST); needs valid session.
3. **NVR camera LAN** — may use insecure TLS to cams (LAN assumption).
4. **Registry auth=false** default on localhost registry — OK if not published.
5. **Multi-level DNS + CF orange** — free Universal SSL gap; grey + `:8443` recommended for `*.nd.neyman.top`.
6. **Control form labels** — some technical IDs stay EN (not user copy).

### No critical findings in 0.9.16 path

No public password SSH day-2, no default plain agent, no TG fleet deploy, no public VPS admin.

## Bilingual (EN/RU)

| Surface | Status |
|---------|--------|
| Web Installer + Settings + nav | data-i18n + I18N |
| Web Control section tabs + main actions | t() / data-i18n |
| Web Control technical field names | EN identifiers (API) |
| TUI | TT / l10n |
| TG | i18n |
| CLI doctor/messages | cli18n |
| Docs | EN primary; `docs/ru/` partial |

## Refactor backlog (non-blocking)

- Shared action registry Web↔TG for POST forms
- Optional Structured doctor JSON (not text scrape)
- Site rooms / MT API via Pi — PLAN-* docs only

## Verdict

**Ship.** Mac deploy via Web and TUI is the supported path; day-2 Control + Tools cover operator needs; security model holds with documented residuals.
