# Security & code review — v0.9.9

Date: 2026-09-25

## Model

| Surface | Exposure | Auth |
|---------|----------|------|
| Operator WebUI | 127.0.0.1 + op token | Local |
| Node API :8787 | Loopback; Mac via SSH -L | Session |
| Agent :8789 | WAN | mTLS |
| Recovery :8790 | Armed only | Bearer + TTL |
| VPS /admin | Off default | LEGACY flag |

## Findings

**Critical/High:** none new in 0.9.9.

**Medium:** session in localStorage (loopback XSS residual); Advanced form is full API power (requires session); recovery WAN while armed (ops TTL); LAN InsecureSkipVerify for cameras.

**Low:** form labels EN-only; TG/TUI not full POST mirror.

## Refactor (non-blocking)

Split web/index.html; shared action registry for TG+WebUI; optional CSP on op serve.

## Verdict

Ship 0.9.9. Keep PLAIN_AGENT / API_PUBLIC / CLAIM_FIRST off.


## Fixes in 0.9.10

- [x] Session token → **sessionStorage only** (not localStorage)
- [x] Stronger CSP + Permissions-Policy + COOP on operator serve
- [x] Confirm dialog before Advanced POST
- [x] Control section labels EN/RU
