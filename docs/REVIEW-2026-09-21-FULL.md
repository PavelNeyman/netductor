# Full review cycle — code · security · refactor · UI (v0.8.20)

Date: 2026-09-21. Baseline after tests + parity scripts green.

## 1. Code review

### Strengths
- Single Go control plane (`netductor`, `netductor-tg`, `netductor-agent`)
- Clear packages: `deploy`, `edge`, `mtls`, `vpn`, `secondary`, `nvr`, `sites`
- Mac TUI workstation wizards → `DeployPrimary` / secondary provision / `DeployEdge`
- Edge enroll → pending → operator Approve; CONTROL_ONLY recovery path
- Self-update with version pin (`internal/deploy.Release`) + SHA256SUMS on releases
- `go test ./...` green (all packages with tests)

### Issues found & fixed this cycle
| Issue | Severity | Action |
|-------|----------|--------|
| Primary `nodes local-cmd upgrade` used **base64 bash** pinned to **v0.7.0-dev** | **High** | Replaced with pure Go upgrade using `deploy.Release` (0.8.20) |
| Secondary upgrade was shell (earlier) | Med | Pure Go since 0.8.17 |

### Remaining soft debt (non-blocking)
- `internal/install` still uses `bash -c` for apt/ufw (acceptable install-time)
- TG bot invokes `status.sh` via bash (read-only status helper)
- Callback aliases `m:relay:*` retained for old clients
- Some Admin option labels (psk2/sae) intentionally protocol tokens, not translated

## 2. Security review

### Public surface (encrypted / product)
| Port | Role | Controls |
|------|------|----------|
| 443 / 4443 / 8443 | VPN Reality / HY2 | Protocol crypto + UUID |
| 22 / 52222 | SSH | Key-only after harden |
| 8789 | Agent plane | **mTLS** + rate limit + ban |

### Restricted
| Port | Role | Controls |
|------|------|----------|
| 8787 | API/Admin | Default **127.0.0.1**; public needs `API_PUBLIC=1` **and** TLS |
| 8788 | Plain agent | **ufw deny**; only `PLAIN_AGENT=1` |
| 7879 recovery | Edge re-attach | Private/LAN bind; **no** public fallback (0.8.17+) |
| redirect | TG deep-link | Default **127.0.0.1:80** (0.8.17+) |

### Residual (documented `docs/RESIDUAL_RISKS.md`)
- `:8789` probeable from internet (required for NAT edge)
- Long-lived bootstrap token if mis-handled
- `CLAIM_FIRST` if left enabled
- Operator cmds on approved edge still powerful (uci/reboot)

**Verdict:** No unexpected plaintext public listeners in default deploy path.

## 3. Architecture / Mac deploy

```
Mac TUI workstation
  → primary  → SSH password→key → release binary → install → SNI → fleet → TG
  → secondary → provision from primary + mTLS material
  → openwrt  → edge provision SERVER=https://primary:8789
  → mikrotik → site/RSC/TOFU
```

Matches product intent. Control plane stays on VPS after install.

## 4. UI / i18n / parity

| Surface | EN/RU | Notes |
|---------|-------|-------|
| TG | Strong | |
| TUI | Strong | TT/FormT + locale |
| Admin | Strong (0.8.20) | data-i18n pass |
| CLI | doctor/vpn/mtls/help | cli18n |

`scripts/check-ui-parity.sh` — all groups **OK**  
`docs/UI-PARITY.md` — matrix  

## 5. Refactor plan (post-0.8.20)

| P | Item | Status |
|---|------|--------|
| P0 | Fix primary upgrade pin | **done 0.8.20** |
| P1 | Hardware e2e | operator |
| P2 | Domain HTTPS redirect | when domain exists |
| P2 | Drop `m:relay` alias after TG clients refreshed | later |
| P3 | More unit tests on deploy SSH path | optional |
| P3 | Replace install-time bash apt with pure exec (partial already) | low |

## 6. Test evidence
- `go test ./...` — pass
- `check-version-pins.sh` — pass
- `check-ui-parity.sh` — pass
