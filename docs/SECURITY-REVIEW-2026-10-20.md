# Security review & fix plan — tip v0.9.444 → v0.9.445

**Date:** 2026-10-20 · **Reviewed tip:** `1b1946b` / v0.9.444

Authoritative closure checklist. A finding is closed only when acceptance criteria pass (negative regression + callers), not only when a helper was edited.

## Findings

| ID | Severity | Fix in 0.9.445 |
|--|--|--|
| **H-1** Bootstrap enroll returns existing `device_token` | Critical | **Done** — Enroll returns empty token for approved; token only via Approve |
| **H-3** `IsRevoked` fail-open on I/O/JSON error | Critical | **Done** — `(bool, error)`; TLS rejects on error |
| **H-2** Firewall apply swallows errors; weak Collect OK | High | **Done** — critical ufw must-run; iptables run returns errors; Collect stricter |
| **N-10** OpenWrt USB glob install / weak names | High | **Done** — explicit basenames + `validUSBPkgName` |
| **M-5** Partial CA auto-generates new root | High | **Done** — fail-closed if only cert or only key |
| **M-7** Fixed `/tmp/nd-ak-parse` write | Local | **Done** — removed |
| **N-8** `RenameSubToken` no mutex; late order | Integrity | **Done** — `subMu` + rename after registry |
| **M-3** download no CheckRedirect | Policy | **Done** — redirect re-validates URL policy |
| **L-3** `loadCA` pem.Decode nil | Stability | **Done** — nil checks |
| **M-1** NVR camera_id path | High | **Open (P1)** |
| **M-4** Local release without SHA256SUMS | High | **Open (P1)** |
| **N-9** HTTP timeouts | Medium | **Open (P2)** |
| **M-2** Host/Origin op UI | Hardening | **Open (P2)** |
| **M-6** Ship tag vs HEAD | Release | **Open (P2)** |
| **L-1** Public :80 ACME | Architecture | **Product decision** — not silent-fixed |

## By design (not bugs here)

- iptables fallback without INPUT default DROP (anti-lockout)
- VPN disable revokes sub URL; re-enable = new token
- Bootstrap still required for first enroll (only escalation removed)
- Operator token on loopback Web

## Acceptance (P0 closed in code)

- **H-1:** `TestEnrollApprovedNoTokenLeak` — re-enroll after Approve returns no token
- **H-3:** `TestIsRevokedFailClosedCorruptFile` — corrupt list errors; TLS path uses error
- **H-2:** critical `default deny` / SSH / 443 / enable return errors; Collect not OK without 443/SSH
- **N-10:** install script lists exact uploaded files; invalid package names rejected

## Remaining work

1. M-1 NVR path containment + symlink tests  
2. M-4 local install requires full SHA256SUMS  
3. N-9 listener timeouts (except live streams)  
4. M-2 Host/Origin  
5. M-6 ship checkout of tag SHA  
6. L-1 ACME :80 decision (DNS-01 vs keep)

