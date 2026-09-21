# Deep code / security / UI review — v0.8.27 (2026-09-21)

## 1. Executive summary

| Area | Verdict |
|------|---------|
| Mac TUI deploy scenario | **Aligned** — workstation wizards call `deploy.DeployPrimary/Secondary/Edge` + MT site flow |
| Public ports without crypto | **Mostly OK** — product VPN/SSH/mTLS; residual risks: optional **:80 redirect**, recovery if WAN-open, footguns `PLAIN_AGENT` / `API_PUBLIC` |
| CLI ↔ UI parity | **Partial** — core ops present; advanced CLI (mtls rollover finish, full NVR, probes, fleet flags) uneven |
| Bilingual EN/RU | **TG/TUI strong**; **Admin incomplete**; CLI doctor/vpn/mtls via cli18n |

---

## 2. Mac TUI deploy path (code match)

```
netductor tui --mode workstation
  → Setup / tab wizard
      → primary   → deploy.DeployPrimary (SSH password→key, release binary, install, SNI, fleet bootstrap, TG secrets)
      → secondary → fleet provision-secondary (from primary or remote) + mTLS client material
      → openwrt   → deploy.DeployEdge (agent + SERVER=https://primary:8789 + mTLS + CONTROL_ONLY path)
      → mikrotik  → site/RSC/TOFU SSH
```

**Verified in tree**

- `tui_wizard.go` / `tui_deploy_wizards.go` — full forms (huh + FormT)
- `internal/deploy/primary.go`, `edge.go`, fleet provision
- Edge URL forced to **:8789** mTLS (not plain :8787)
- SSH harden: PasswordAuthentication no after key

**Residual deploy risks**

- First-login password still required once (by design)
- Operator must approve edge after enroll (by design)
- Version pin must match Releases (`deploy.Release`)

---

## 3. Security review — listening surface

### Intended public (encrypted)

| Port | Role | Protection |
|------|------|------------|
| **443** | VLESS Reality (secondary entry / core as designed) | Reality + UUID + flow |
| **4443** | HY2 / exit-related (config) | protocol crypto |
| **8443** | optional HY2 core | protocol crypto |
| **22** / **52222** | SSH | key-only after harden |
| **8789** | Agent plane (edge+secondary+nvr) | **mTLS client cert** + rate limit + ban; ufw allows from anywhere (NAT edges) |

### Localhost / restricted

| Port | Role |
|------|------|
| **8787** | API + Admin default **127.0.0.1** — public bind needs `API_PUBLIC=1` **and** TLS certs |
| **8788** | Plain agent — **denied** ufw; only `NETDUCTOR_PLAIN_AGENT=1` |
| Lampac etc. | localhost policy in doctor |

### Attention / residual

| Issue | Severity | Notes |
|-------|----------|--------|
| **:80 redirect** (`redirect-serve`) | Medium | HTTP by design for client deep-links; prefer VPN-only or HTTPS when domain exists |
| **:7879 recovery** | Medium if WAN | Code prefers private bind + LAN clients; `ALLOW_ANY` / fallback `:7879` can expose token form |
| **8789 on internet** | Low–Med | Handshake fails without cert; still probeable — ban helps |
| **CLAIM_FIRST** | High if left on | Doctor warns |
| **Bootstrap/recovery codes** | High if leaked | Treat as secrets |
| **Clip URL tokens** | Low if API local | Capability URLs |
| **secondary `upgrade` uses `bash -c`** | Med | Fixed script, not user input — keep allowlisted |
| **uci_set from operator cmds** | Med | Approved devices only; still powerful |

### Crypto posture

- Agent plane: TLS 1.3 + client cert + revoke list + dual-CA support  
- VPN: Reality/HY2 (not plaintext)  
- SSH: key-only post-install  
- API public: requires TLS files  

**Not “everything public is encrypted” only if**: HTTP :80 redirect is enabled, or plain :8788, or recovery bound to WAN.

---

## 4. Code quality review

**Strengths**

- Monolith Go control plane; clear packages (`deploy`, `edge`, `mtls`, `vpn`, `secondary`)
- Edge enroll → pending → approve
- Self-update SHA256; recovery grace rotate
- Doctor surface checks for footguns

**Weaknesses**

- Large CLI surface / duplicated TG `runND` shell-outs
- Residual “relay” identifiers in callbacks (`m:relay:*`) while labels say secondary
- Admin static HTML many labels without `data-i18n`
- `exec.Command` count high — mostly fixed args; secondary upgrade still shell script string

---

## 5. Refactoring plan

| P | Item |
|---|------|
| P1 | Single `AgentPlaneMux()` builder already partly unified — extract register from `StartAgentPlane` fully |
| P1 | Rename TG callbacks `m:relay:*` → `m:secondary:*` (compat alias) |
| P1 | Admin: generate i18n for **all** visible strings (table headers, form labels) |
| P2 | CLI parity matrix auto-test: each `netductor <cmd>` has TG and/or TUI and/or Admin entry |
| P2 | Remove shell from secondary `upgrade`; pure Go download + systemctl |
| P2 | Expand cli18n to remaining cli_*.go usage lines (sites, nvr, fleet, install) |
| P3 | Optional mTLS for redirect HTTPS when domain present |
| P3 | Agent cert auto-push e2e test in CI with fake agent |

---

## 6. Bilingual audit

| Surface | EN/RU |
|---------|--------|
| Telegram | Strong (`i18n.go`) |
| TUI menus / wizards | Strong (`TT`, `FormT`, detectLang) |
| CLI doctor / vpn / mtls / help | Strong (`cli18n`) |
| CLI sites/nvr/fleet/install long texts | Partial |
| Admin UI | **Incomplete** — tabs have data-i18n; many form labels / table headers hardcoded EN |

---

## 7. CLI → UI parity (high level)

| Capability | CLI | TUI | TG | Admin |
|------------|-----|-----|----|-------|
| Deploy primary/secondary/edge | ✓ | ✓ wizard | — | — |
| VPN users add/list/link | ✓ | partial | ✓ | ✓ |
| Edge pending/approve | ✓ | partial | ✓ | ✓ |
| mTLS list/rotate | ✓ | list | list/rotate | list API |
| mTLS rollover | ✓ | — | — | — |
| Doctor / status | ✓ | ✓ | status | status |
| NVR full | ✓ | partial | partial | partial |
| Backup peer | ✓ | ✓ | ✓ | ✓ |
| SSH hosts TOFU | ✓ | — | ✓ | ✓ |
| Probe/collect/YABS | ✓ | probe | — | — |
| Secondary sync/exit | ✓ | ✓ | ✓ | partial |

Deploy from Mac is TUI-centric (correct). Day-2 ops split across TG + Admin + TUI remote.

---

## 8. Recommendations before hardware e2e

1. Confirm ufw: 8788 deny, 8789 allow, 22/52222, 443/4443 as needed; no accidental 8787 WAN.  
2. `doctor` on primary + secondary after Mac deploy.  
3. Do not set `NETDUCTOR_PLAIN_AGENT` / `API_PUBLIC` on prod.  
4. Close Admin i18n + TG callback rename as next polish sprint.  
