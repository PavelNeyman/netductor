**EN** · [RU](ru/REVIEW-0.9.73.md)

# Code & security review — v0.9.73 (2026-09-27)

## Verdict

**Production-ready for dual-node smoke** under freeze rules. No critical open RCE / unauthenticated admin on WAN. Residual = ops discipline + hardware e2e.

## Security

| Area | Status |
|------|--------|
| Node API bind | Non-local **refused**; loopback only |
| :8789 | mTLS; UFW service CIDR + api-allow; WAN only `api-public arm` TTL |
| Plain :8788 | **Refuse** if `PLAIN_AGENT=1` |
| TRUST_PROXY / ALLOW_PUBLIC permanent | Disabled / removed |
| Legacy VPS admin | Disabled |
| Redirect | HTTPS :8443; HTTP off; `/` → 404; allowlisted deep links |
| Recovery :8790 | Armed only; encrypted blob; offline key |
| SSH | 52222 key-only after harden |
| Backup | No private keys in archive; pubs only |
| Rate-limit | Agent plane; XFF not trusted |
| InsecureSkipVerify | Recovery / LAN camera only (documented) |

**Ops residual:** leave no footgun env in units; LE certs not in `.ndenc` (re-issue after recover); dual-node smoke still owner next.

## Code / recover VPN path

| Issue | Fix in tree |
|-------|-------------|
| Inbound mux `max_connections` broke `vpn apply` (sing-box 1.14) | **0.9.73** padding-only inbound |
| Empty users / conf ≠ secrets after recover | post-restore `ensure-relay-uplink` + `vpn apply` |
| Multiplex | **ON** canon (outbound full; inbound limited fields) |

## Mac deploy parity

| Path | Primary | Secondary | Fleet | Edge | Creds |
|------|---------|-----------|-------|------|-------|
| TUI | ✅ wizard | ✅ | ✅ | ✅ | ✅ |
| Web (`internal/operator/web`) | ✅ `/v1/primary` | ✅ | ✅ `/v1/fleet` | ✅ | ✅ |
| Backend | `internal/deploy` shared | same | same | same | same |

TG does **not** deploy VPS (intentional).

## API coverage

`docs/OPCATALOG.md`: day-2 matrix Web/TUI/TG/CLI largely ✅. Intentional gaps: metrics absorbed in TG Status; deploy Mac-only.

## i18n docs

42 EN + 42 RU under `docs/` / `docs/ru/`; sense parity pass **ad3db20**. UI: TG+Web+TUI l10n present; CLI long strings residual polish only.

## Next chat

1. Dual-node full smoke (both VPS)  
2. Hardware e2e when ready  
3. Features only after smoke  
