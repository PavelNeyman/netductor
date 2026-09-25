## 0.9.6

- Control: full section coverage (VPN, Edge/guest, NVR, Git/Registry, Backup, Probes, …)
- Result: HTML tables + raw JSON toggle

## 0.9.5

- P4: managed SSH tunnel from WebUI (start/stop/status, auto on Control)
- P4: Issue session from Control → localStorage + ~/.netductor/node_session

## 0.9.4

- Node API-only by default: `/admin` returns 410 unless `NETDUCTOR_LEGACY_ADMIN_UI=1`
- Install no longer ships static admin UI by default

## 0.9.3

- Control P2: VPN users, nodes, edge pending/approve via `/v1/node` proxy (CORS-safe)
- `netductor-op session issue` — create node API session over SSH
- Node session token in WebUI Settings

## 0.9.2

- Locked: Mac client UI only; node = API server ([PLAN-MAC-CLIENT.md](docs/PLAN-MAC-CLIENT.md))
- WebUI: Installer | Control | Settings; tunnel CLI; Control hits node `/health`
- VPS `/admin` documented as legacy

## 0.9.1

- Operator WebUI: Fleet / Primary / Secondary / Credentials tabs, EN/RU, step chips, localStorage
- API: POST /v1/primary, /v1/secondary, /v1/credentials; GET /v1/meta

## 0.9.0

- Physical split: **netductor-op** (Mac) vs **netductor** node (VPS)
- Recovery: default bind **0.0.0.0 while armed** (DR); doctor WARN not FAIL
- Security: path/XFF/token hardening (see prior 0.8.98–0.9.0 commits)

## 0.8.98

- security(operator): token auth for /v1/fleet; host validation; credentials role sanitize
- operator serve: timeouts, deploy mutex, max body 1MiB, CLI --token

## 0.8.97

- `netductor operator serve` — localhost UI + `POST /v1/fleet` (loopback only)
- embed WebUI for FleetDeploy

## 0.8.96

- operator.Step + FleetDeployWithReport structured progress events

## 0.8.95

- internal/operator use-cases; CLI/TUI thin clients; deploy fleet

## Docs (post-0.8.94)

- [ARCHITECTURE-OPERATOR.md](docs/ARCHITECTURE-OPERATOR.md) — operator core + UI clients
- [OPERATOR-PLAN.md](docs/OPERATOR-PLAN.md) — phased checklist
- Handoff compressed; AGENTS.md v2.6 mandatory doc/checklist discipline

## 0.8.94

- Remove legacy huh deploy wizards; framed Setup only
- Wizard **Fleet**: primary+secondary one run (domain/LE/add-ons/TG → credentials on Mac)
- Primary wizard: lampac/git/TG toggles + LE parity with CLI

## 0.8.92

- Default: no CF proxy for multi-level `i.netductor.*` (grey + :8443)
- TUI `cf_proxy` default **no**; DOMAIN.md Universal SSL limit note

## 0.8.91

- --cf-proxy / TUI cf_proxy for Cloudflare orange on i.
- REDIRECT_BASE without :8443; Origin Rule → :8443

## 0.8.90

- Secondary RemoteJoin: no ancient FALLBACK 0.8.45 — download current release only
- docs/UI-PARITY.md rewritten for framed TUI + Mac-direct secondary
- DOMAIN.md: Cloudflare orange for `i.` is optional
- OpenWrt wizard labels FormT; stale huh-path comments fixed

## 0.8.89

- **credentials collect**: full dump of `/etc/netductor/secrets`, conf, LE, secondary devices.json → `~/.netductor/credentials/<role>-<host>-ts/`
- Port fallback 52222/22; CLI `--port`; secondary `--primary-key-passphrase` `--secondary-key`
- DEPLOY-MAC + handoff updated

## 0.8.88

- Redirect after LE: **:8443 only** (no public :80); certbot renew uses free :80
- Deploy automation: key chmod 600, empty .pub repair, REDIRECT_BASE :8443
- TUI≡CLI parity docs [DEPLOY-MAC.md](docs/DEPLOY-MAC.md); FormT le_email
- InstallRedirect prefers LE HTTPS unit

## 0.8.87

- LE redirect: `REDIRECT_BASE=https://i.<base>:8443` (port 443 is Reality, not LE)
- ufw allow 8443/tcp on tls le; docs DOMAIN.md Reality vs LE

## 0.8.86

- **Secondary deploy Mac-direct:** primary only `prepare-pack` (token/mTLS/bundle); **Mac SSHs to secondary** (no primary→secondary SSH)
- CLI: `netductor secondary prepare-pack --sni …`

## 0.8.85

- deploy: SSH_ASKPASS fallback without sshpass; CLI --domain-base/--le-email
- redirect HTTPS **:8443** (443 = Reality)
- secondary deploy defaults NETDUCTOR_SSH_PORT=52222 for primary
- agent logs failed ticks; NETDUCTOR_KEY_PATH for durable key path

## 0.8.84

- **Let's Encrypt:** `netductor tls le` + `domain set --le --email`
- Primary/TUI: optional LE email with domain base → auto certs + HTTPS redirect
- Redirect unit :80 + :443 when certs present

## 0.8.83

- `netductor domain set|show` — primary/vpn/redirect from `--base`
- Primary wizard optional domain base; deploy runs domain set on host
- Docs: DOMAIN.md

## 0.8.82

- Recovery arm: **HTTPS** self-signed by default (`RECOVERY_TLS=0` for HTTP)
- Credentials: keep last 5 files per role
- Handoff/docs: knock history trimmed; baseline 0.8.82

## 0.8.81

- Credentials: no file on SSH/secret failure; `latest-<role>.txt` symlink; iCloud warning
- CLI: `netductor credentials collect`
- Handoff: canonical baseline (no knock)
- TUI log marks credentials step

## 0.8.80

- Remove recovery port-knock; arm over SSH only
- After primary/secondary deploy: write `~/.netductor/credentials/*.txt` (BACKUP_KEY, RECOVERY_TOKEN, SSH)
- Docs: OPERATOR_CREDENTIALS.md

## 0.8.79

- Recovery: remove always-on legacy; 8-port **random** knock sequence per host (secrets file)
- CLI: `recovery knock-show` / `knock-regen`

## 0.8.78

- Recovery HTTP **off by default**; `netductor recovery arm|disarm|status`
- Port-knock sequence (default 41222→41223→41224) arms recovery for TTL
- Recovery token compare: constant-time
- Edge: TUN start failure → automatic socks@127.0.0.1 fallback

## 0.8.77

- Edge VPN default mode: **tun** (was socks/mixed)
- socks/mixed remains opt-in via template `vpn.mode` or explicit mode

## 0.8.76

- Recovery: **no backup key on the wire** by default; `--key` / `NETDUCTOR_BACKUP_KEY` required for recover-from-secondary
- Recovery: real CIDR allowlist, auth lockout (5 fails → 15m), optional `RECOVERY_SERVE_KEY=1`
- TUI: clamp log scroll (no vanishing lines past top)
- Edge mixed inbound: default `127.0.0.1:7890` (override `NETDUCTOR_EDGE_MIXED_LISTEN`)

## 0.8.75

- Telegram **fully optional**: removed from DefaultComponents / core primary install
- `netductor install telegram` installs binary + systemd unit (error if binary missing)
- Doctor: INFO skip when bot unit not present
- Formula + release assets include netductor-tg

## 0.8.74

- Primary deploy: **no** automatic lampac / git / registry (only core install)
- Git+Registry moved to Add-ons checklist (`git_registry` toggle)
- Primary progress checklist: core steps only
- TUI: disable mouse capture so terminal select/copy works (scroll: arrows / PgUp/PgDn)

## 0.8.73

- Primary wizard: removed TG token/admin (use Add-ons)

## 0.8.72

- Add-ons: target **any VPS** (host/user/key), not primary-only
- Telegram: token + admin id fields when installing bot
- Primary still has no Lampac yes/no (use Add-ons)

## 0.8.71

- Primary wizard: **removed** single Lampac yes/no (was confusing)
- Add-ons checklist is the only multi-select path (Мастер → Дополнения)
- Tools → same checklist

## 0.8.70

- TUI Add-ons: **checklist** [✓]/[ ] with Space toggle (not sequential yes/no batch)
- Ctrl+R from checklist installs selected only

## 0.8.69

- **Fix:** publish `netductor-tg-linux-*` on releases; InstallTelegram clearer fallback; primary deploy ensures bot after secrets
- **TUI Add-ons multi-select:** Lampac / Telegram / go2rtc (placeholder) each yes/no independently

- Plan: TUI Add-ons multi-select (per component), not all-or-nothing

## 0.8.67

- TUI deploy: **line-by-line stream** + **checklist/progress bar**
- Toggle views: **p** = progress/checklist, **o** = log (not both at once)

## 0.8.66

- **TUI integrity rule** locked (`docs/TUI-RULES.md`): no external huh/quit for operator flows
- Framed Output (header+border); Tools forms stay in-TUI; site/addons via wizard fields

## 0.8.65

- TUI Wizard: deploy runs **inside** framed UI (async log pane, no bare Output screen)

## 0.8.64

- SSH deploy: clear stale known_hosts before connect (fixes exit 255 after VPS reinstall)
- TUI wizard paste: accept long KeyMsg strings from terminals without bracketed-paste flag

## 0.8.63

- TUI Wizard fields: paste support (bracketed paste + Ctrl+V clipboard), multi-rune input

## 0.8.62

- TUI Wizard: menu-style split layout — short hints + right detail pane for all field steps

## 0.8.61

- TUI Wizard: multi-step fields **inside** framed UI (no external huh window)
- Primary / Secondary / OpenWrt / NVR / Add-ons collect data in-screen; result on Output

## 0.8.60

- go2rtc: WebRTC listen **127.0.0.1:8555** (was `:8555` all interfaces)
- PORTS.md: document go2rtc 1984/8554/8555 localhost-only

## 0.8.59

- TUI: all deploy wizards (Primary/Secondary/OpenWrt/MikroTik/NVR/Add-ons) only outside alt-screen
- Wizard tab: Add-ons entry; Tools site/mt/ssh forms exit to huh
- runWizardApply no longer invokes huh inside Bubble Tea

## 0.8.58

- TUI Wizard tab: exit alt-screen before huh Primary/Secondary/OpenWrt/… forms (fixes blank UI)

## Formula

- Homebrew: pin **0.8.57** with real SHA256 (no `:no_check`)

## Docs

- DEPLOY-WORKSTATION + handoff + OPEN_ITEMS locked to **v0.8.57** (TUI flow, recover e2e)

## 0.8.57

- recover: **peek operator pubkeys from backup tar before harden** (no required env)
- NETDUCTOR_OPERATOR_PUBKEY remains optional extra merge

## 0.8.56

- recover: inject operator pubkey before harden + reload sshd after restore
- PLAN-NVR: go2rtc two-way audio (`tapo://`)

## 0.8.55

- probes: api-health default **http://127.0.0.1:8787/health** (API is HTTP-only); migrate old https defaults

## 0.8.54

- TG: restore in-body rich `<tg-button>` (Access/hub); warn if REDIRECT_BASE unset
- install: EnsureClientProfiles (`nd-oc.conf`), EnsureDomainConfig, InstallRedirect
- Env: `NETDUCTOR_DOMAIN` / `NETDUCTOR_REDIRECT_BASE` / `NETDUCTOR_PUBLIC_HOSTNAME`

## 0.8.53

- TG: Access/User hub use **classic inline_keyboard** (no broken tg-button text glue)
- importRedirectURL returns empty without absolute http(s) base (fixes BUTTON_URL_INVALID)
- replyRichWithPhoto prefers sendPhoto + keyboard

## 0.8.52

- Backup: snapshot live hostname + operator **public** keys (`hostname.backup`, `operator_authorized_keys`)
- Recover: SkipHostname (no invented `nd-core-*`); restore hostname from backup; inject operator pubkeys (+ `NETDUCTOR_OPERATOR_PUBKEY` / `_FILE`)

## 0.8.51

- Remove SCP backup.offsite path (agent backup_pull only)
- TUI: current mode chip in header; clearer Mode tab; persist mode in tui.yaml; confirmation screen on change

## 0.8.50

- Fix: StartRecoveryServer in AgentLoop (was missing on secondary)
- Backup: TG alert when no online secondary for backup_pull
- Recovery: NETDUCTOR_RECOVERY_BIND / UFW / ALLOW_CIDR + rate limit

## 0.8.49

- DeployPrimary: after harden auto `NETDUCTOR_SSH_PORT=52222` for post-steps
- install lampac: Debian `docker-cli` + `docker.io`
- registry crane: correct GitHub asset names (Linux_x86_64, v0.22.1)
- hardening apt: git, wget
- primary deploy: registry+git bootstrap after install
- secondary ProvisionIn: SSH private key auth when password disabled

## 0.8.44

- TUI Setup: **Add-ons** (Lampac) + optional Lampac after primary deploy
- `deploy primary --with-lampac`; `deploy.RunOnPrimary`
- InstallLampac: ensure Docker (apt docker.io / get.docker.com), image fallbacks

## 0.8.43

- Deploy SSH: `NETDUCTOR_SSH_PORT` (default 22; use 52222 after harden)
- Primary binary download: curl **or** wget
- E2E: primary+secondary VPS path validated

## 0.8.42

- CLI `deploy edge`: full parity with TUI network/guest flags (LAN/DHCP/Wi‑Fi 2.4+5/WAN/pppoe)
- `deploy secondary --primary-key-passphrase`; edge/secondary password via NETDUCTOR_SSH_PASSWORD

## 0.8.41

- OpenWrt WAN: dhcp | static | **pppoe** (user/pass/service/AC)
- Wi‑Fi 2.4/5 separate fields; empty band inherits the other (EN/RU hints)
- edgeagent DesiredUCI dual-band + PPPoE

## 0.8.40

- OpenWrt wizard: optional LAN/DHCP/Wi‑Fi/WAN (dhcp|static) EN/RU → UCI on router + edge template
- edgeagent DesiredUCI: wan static, dhcp pool; ShellApply for first-boot

## 0.8.39

- Primary SSH key: optional passphrase in TUI (ask yes/no → phrase + confirm) and CLI `--key-passphrase`
- Key passphrase via SSH_ASKPASS (not argv); secondary/edge wizards can supply primary key phrase
- Form labels EN/RU for passphrase flow

## 0.8.38

- SSH: sshpass uses `SSHPASS` + `-e` (no password in argv)
- fleet provision-secondary: `--password-stdin` / `NETDUCTOR_SSH_PASSWORD`
- Deploy secondary pipes password on primary via stdin
- API `MaxBodyBytes` = 16 MiB, used by `withSecurity` + `readJSON`
- AGENT_HANDOFF baseline cleanup (stale 0.8.30 operator hints)

## 0.8.37

- Security: API systemd restart allowlist (`netductor-*` only)
- Docs/handoff baseline lock after isolated CI + deep review (no large refactor)

## 0.8.36

- Isolated CI: builds/tests in docker/podman by default (`netductor ci status|test|exec`)
- GHA-subset honors `container:`; auto image by go.mod/package.json/…
- Managed pipelines ci-run/go-test/oci-push use isolation; host keeps git+registry data only
- Doctor CI isolation check; seller guest-bot parked under Ideas

## 0.8.35

- Harden git artifact path reads (no `..` escape)
- Docs/handoff/OPEN_ITEMS baseline lock; security review snapshot
- Git+registry+GHA surface marked complete for thin self-host

## 0.8.34

- Git/registry UI parity: Admin+TG workflow, artifacts; catalog tags; registry htpasswd auth
- Pipeline/workflow logs under state/git-artifacts; doctor checks git root + registry
- Sample pipeline `ci-run` (Go/Node/Rust/Python/Make); multi-language docs

## 0.8.33

- TUI: single deploy path (tab Wizard → confirm → huh deploy wizards); drop dead field forms
- NVR in tab wizard targets; GitHub Actions–subset is the only CI YAML target (no GitLab dual-DSL)

# Changelog

## 0.9.0

- security(recovery): default bind 127.0.0.1; non-loopback requires ALLOW_CIDR
- security: SHA-256 constant-time token compares (secondary/edge/recovery)
- doctor: FAIL if recovery :8790 on 0.0.0.0

- security: Content-Disposition filename sanitization; BackupPath rejects control chars
- security: rate-limit / clientIP trust XFF only with NETDUCTOR_TRUST_PROXY=1
- chore: agent version 0.9.0; registerSecondaryAPI rename (drop relay naming)

- **Physical split:** `cmd/netductor-op` (deploy/TUI/operator serve only) vs `cmd/netductor` (node plane only)
- No shared “fat” binary; operator day-2 uses SSH → `netductor` on primary

- **Binary split:** `netductor-op` (operator / Mac) vs `netductor` node (`netductor-linux-*` on VPS)
- Link-time `main.binaryRole=operator|node|all`; command surfaces gated
- Deploy / secondary provision download **node** assets only (never operator binary onto VPS)
- Release workflow emits both asset families; docs + OPERATOR-PLAN Phase 4


## 0.8.99

- security(deploy): release version / agent arch charset; https-only edge ServerURL
- security(deploy): scp uses `--` before paths; token compare via SHA-256

- TUI wizard: all deploy targets via operator (EdgeFromFields, RunRemote); step matcher for Fleet events

- security(operator): stricter ValidHost/ValidUser; DeployEdge use-case
- security(operator): token charset + CSP/XFO on operator serve
- security(deploy): ssh `--` before user@host
- CLI/TUI edge via operator core only


## 0.8.32

- GHA-subset workflow runner: parse `.github/workflows/*.yml`, execute `run:` steps only
- `netductor git workflow <repo> [path]`; post-receive prefers workflow if `NETDUCTOR_GIT_WORKFLOW=1` or path set
- Example workflow in docs; CHANGELOG rebuilt for 0.8.4–0.8.31

## 0.8.31

- Local OCI registry: `registry status|ensure|stop|crane|catalog` + API/Admin/TG
- Pipeline sample `oci-push`; post-receive auto-runs `NETDUCTOR_GIT_PIPELINE`
- Docs SELFHOST-GIT / OPEN_ITEMS

## 0.8.30

- Git: `git delete`, sample `go-test` pipeline, docs/handoff sync
- Thin model: no Forgejo by default

## 0.8.29

- Git pipelines: CLI/API/Admin tab/TG

## 0.8.28

- Thin `netductor git` CLI + SELFHOST-GIT clarification

## 0.8.27

- SSH listen **52222** + fail2ban; SELFHOST-GIT facts

## 0.8.26

- Guest cmd allowlist + TG wait `cmd_result`; certs in TG; docs cleanup

## 0.8.25

- Guest wizard, PNG QR, TG/Admin edge guest control

## 0.8.24

- Guest nft MAC gate, captive DNAT, VPN bypass, DHCP

## 0.8.23

- Guest Wi-Fi allow-list, captive+desk, default grant 10m

## 0.8.22

- Session test, SMTP host, TUI EchoMode, wizard SA4006

## 0.8.21

- SA4000 doctor, dead code removal, HTTP status constants

## 0.8.20

- `go vet` clean; `internal/version` single pin; remove dead TG code & 0.7 install URLs

## 0.8.19

- Full review docs; primary upgrade uses `deploy.Release` (no v0.7 base64)

## 0.8.18

- Admin i18n complete pass, UI parity matrix, mTLS rollover UI

## 0.8.17

- Residual risk mitigations, secondary callbacks, pure-Go upgrade

## 0.8.16

- cli18n full doctor body + VPN CLI strings EN/RU
- Deep code/security/UI review docs

## 0.8.15

- cli18n CLI, `StartAgentPlane`, relay→secondary UI, mTLS in all UIs

## 0.8.14

- mTLS auto-push + grace; CA dual-trust rollover; cli18n

## 0.8.13

- mTLS revoke/rotate; cert expiry doctor; plane IP ban after rate limits

## 0.8.12

- Agent plane rate limit (180/min/IP); FormT deploy labels
- CI version-pin step

## 0.8.11

- Split `netductor-agent` (nvr_cmds / openwrt_net / agent_ops)
- `scripts/check-version-pins.sh`; FormT; docs pin

## 0.8.10

- Mac TUI tab wizard = full deploy wizards; `deploy.Release` pin

## 0.8.9

- TUI deploy/forms/sites RU/EN; TG NVR i18n

## 0.8.8

- Drop IP allowlist; admin EN/RU expanded

## 0.8.7

- Edge NAT-friendly agent plane (mTLS only); admin EN/RU tabs

## 0.8.6

- Agent allowlist experiment (later dropped for edge); UI/docs i18n

## 0.8.5

- Edge mTLS on agent plane `:8789`

## 0.8.4

- mTLS-only secondary agent plane, auto certs, UFW lock

## 0.8.31

- Local OCI registry: `netductor registry status|ensure|stop|crane|catalog` + API/Admin/TG
- Pipeline sample `oci-push`; post-receive auto-runs `NETDUCTOR_GIT_PIPELINE`
- Docs SELFHOST-GIT / OPEN_ITEMS

## 0.8.30

- mTLS: revoke list + VerifyPeerCertificate; rotate/list CLI & API
- Doctor: cert expiry WARN (≤30 days); revoked client flags
- Agent plane: temporary IP ban after repeated rate limits
- Docs: MTLS.md, OPEN_ITEMS (hardware e2e remaining)

## 0.8.30 — 2026-09-20

### Finish remaining refactor items
- Agent plane `:8789` IP rate limit (180 req/min; plain emergency 60)
- `FormT` expanded; deploy wizard labels use dictionary
- CI: `scripts/check-version-pins.sh` step (via GitHub connector)

## 0.8.11 — 2026-09-20

### Refactor
- Split `netductor-agent` into `nvr_cmds.go`, `openwrt_net.go`, `agent_ops.go`
- `scripts/check-version-pins.sh` + CI step (no stale v0.7 in operator paths)
- `FormT` dictionary for shared TUI form labels
- Docs/bootstrap/TG/admin download pins → current release

## 0.8.10 — 2026-09-20

### Mac workstation deploy scenario
- Bubbletea **tab wizard** now delegates to full `wizardPrimary/Secondary/OpenWrt` (same as Setup menu) — no more local-only install on laptop
- `internal/deploy.Release` single pin for binary/agent downloads

## 0.8.9 — 2026-09-20

### TUI / TG i18n
- Deploy wizards, VPN forms, sites, SSH hosts, menu items: full RU/EN via TT()
- TG NVR buttons use i18n dict

## 0.8.8 — 2026-09-20

### Remove agent IP allowlist
- Firewall: deny `:8788`, allow `:8789` (mTLS only) — no IP allowlist
- Edge behind NAT and secondary both use client certificates

### Admin UI i18n
- Tabs and main section labels EN/RU

## 0.8.7 — 2026-09-20

### Agent plane policy (edge + secondary)
- **Edge behind NAT:** no IP allowlist (WAN changes); security = mTLS client cert on `:8789`
- **Secondary:** public IP recorded in `agent_allowlist` (inventory)
- Default ufw: `:8789` open + mTLS; `NETDUCTOR_AGENT_ALLOWLIST_STRICT=1` for secondary-only lock
- Removed edge enroll/heartbeat auto-allowlist

### UI i18n
- Admin tabs + core labels EN/RU

## 0.8.6 — 2026-09-20

### Security default: agent allowlist
- `:8789` **allowlist-only** by default (empty list = closed; no world-open)
- `netductor agent-allowlist list|add|apply`
- Secondary/edge provision and edge enroll/heartbeat add source IPs
- Doctor WARN/FAIL on empty allowlist or world-open 8789

### UI / i18n
- TUI/wizard Primary URL → https://IP:8789
- Admin recovery strings EN/RU; relay one-liner → secondary v0.8.5+
- TG secondary download pin updated

## 0.8.5 — 2026-09-20

### Edge mTLS control plane
- Agent plane `:8789` serves edge + NVR device APIs (alongside secondary)
- `netductor-agent` loads `/etc/netductor-agent/mtls` client certs; forces `https://host:8789`
- DeployEdge / workstation: issue per-device cert on primary, install on router, never plain `:8787`
- Edge control independent of site VPN

## 0.8.4 — 2026-09-20

### Security (secondary agent plane)
- mTLS-only agent plane on **:8789** (plain :8788 off unless `NETDUCTOR_PLAIN_AGENT=1`)
- Certs auto-generated: `EnsureAll` at install/serve; per-node `EnsureClientFor` at secondary provision
- Client material installed in-band over provision SSH session
- UFW: deny 8788; allow 8789; restrict to secondary IP after provision
- Doctor FAIL if 8788 exposed or mTLS missing

### Edge (documented plan)
- Control plane must stay reachable **without** site VPN (mTLS public :8789) so router is not lost when VPN is down

## 0.8.3 — 2026-09-20

### Security / SSH
- Operator Mac pubkey on secondary, OpenWrt, MikroTik after first password bootstrap; password auth disabled where supported
- Primary `install` no longer generates `/root/.ssh/id_ed25519` when `authorized_keys` already has a key
- Secondary: `--operator-pubkey` (workstation deploy passes Mac `.pub`)

### Transport (documented)
- Admin API remains localhost `:8787` (SSH tunnel from Mac)
- Secondary agent: prefer mTLS `:8789`; plain `:8788` legacy
- Edge: prefer HTTPS or VPN path; plain public HTTP is token-only, not confidential

### Docs
- ARCHITECTURE, EDGE-AGENT, DEPLOY-WORKSTATION, AGENT_HANDOFF updated

## 0.8.1-docs — 2026-09-20

### Docs
- Align operator guides to **v0.8.1**: bootstrap default, INSTALL/DEPLOY/RUNBOOK/RELEASES/BOOTSTRAP (+ ru)
- CLI naming: `relay` → `secondary` / `fleet provision-secondary` in FLEET, RUNBOOK, RELAY, VPN-USERS, README
- EDGE-REINSTALL: recovery codes documented as implemented (LAN :7879)
- FLEET: secondary sync; drop lab IP placeholder in remote example

## 0.8.1 — 2026-09-20

### Security hardening
- Edge recovery: prefer private IP bind, LAN-only clients, optional SERVER_PIN
- Self-update verifies release SHA256SUMS
- TG: edge approve/deny callbacks; mask device_token in chat
- serve: EnsureAll mTLS before agent plane; SNI fallback api.vk.me
- Doctor warns on CLAIM_FIRST

## 0.8.0 — 2026-09-20

### Highlights
- **Secondary-only** RU entry (legacy relay naming removed)
- Edge **LAN recovery** (`:7879/netductor-recovery`), recovery codes, register/set-site
- NVR/Tapo Go port foundations; RU-direct for gosuslugi stack
- Primary self-update from **GitHub Releases** (TG Tools → Updates); agents **manual** `agent_update`
- Docs EN+RU parity pass; review `docs/REVIEW-2026-09-20.md`

### Breaking / cleanup
- CLI `relay` alias removed → `secondary`
- State path only `secondary/`
- API only `/api/secondary/*`

## 0.7.39-dev

- UI: edge recovery/register/set-site/export in TG, Admin, TUI + API

## 0.7.38-dev

- Edge recovery codes + LAN page :7879; register/export/import/set-site; CONTROL_ONLY

## 0.7.37-dev

- Rename remaining Relay* identifiers to Secondary*; docs EDGE-REINSTALL

## 0.7.36-dev

- Remove legacy relay naming: paths, CLI, API, roles — secondary only

See git history for earlier 0.7.x / 0.5.x notes.
## 0.8.93

- LE: strip :port from redirect hostname for certbot
- docs example p.nd / i.nd / s.nd

