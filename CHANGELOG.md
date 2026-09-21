## 0.8.29

- mTLS: revoke list + VerifyPeerCertificate; rotate/list CLI & API
- Doctor: cert expiry WARN (≤30 days); revoked client flags
- Agent plane: temporary IP ban after repeated rate limits
- Docs: MTLS.md, OPEN_ITEMS (hardware e2e remaining)

## 0.8.29 — 2026-09-20

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
