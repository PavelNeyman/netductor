**EN** · [RU](ru/REVIEW-0.9.75.md)

# Code & security review — v0.9.75 (2026-09-28)

## Verdict

**Ready for dual-node smoke** under architecture freeze. No critical WAN admin/API exposure. Footgun env knobs are **absent from product paths**, not merely refused.

## Security

| Control | Status |
|---------|--------|
| Node API bind | Loopback only; non-local refused |
| :8789 | mTLS; CIDR + api-allow; WAN via **api-public arm** TTL only |
| Plain :8788 | **Not implemented** |
| XFF / TRUST_PROXY | **Not implemented** (RemoteAddr only) |
| VPS /admin | **Removed** |
| Redirect | HTTPS :8443; no public :80 |
| Recovery :8790 | Arm over SSH; encrypted blob; key offline |
| Edge auth | Per-device token only |
| SSH | 52222 key-only after harden |
| Conf obsolete keys | `ndconfig` **ignores** — cannot enable removed features |
| Backup | No private keys; LE not in archive |

## Recover / VPN

| Item | Status |
|------|--------|
| Two-pass recover | 0.9.71+ |
| `ensure-relay-uplink` + `vpn apply` | 0.9.73+ |
| Inbound mux fields | padding only (sing-box 1.14) |
| Multiplex ON | Canon (outbound full) |
| LE re-issue | 0.9.74+ if DOMAIN+LE_EMAIL |

## Mac deploy parity

TUI + Web → `internal/deploy` (primary / secondary / fleet / edge). TG: day-2 only, no VPS deploy.

## API / OPCATALOG

Day-2 matrix documented; intentional gaps (metrics in Status, deploy Mac-only).

## i18n

42 EN + 42 RU doc pairs.

## Residual (ops, not code bugs)

- Dual-node smoke not yet signed off by owner
- Hardware e2e pending
- LE needs DOMAIN+LE_EMAIL once in conf for fully automatic re-issue
- Doctor may still WARN if obsolete env is set in the shell (ignored by product)

## Next

1. Dual-node smoke  
2. Hardware e2e  
3. Features only  
