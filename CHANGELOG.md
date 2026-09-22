## 0.8.48

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
