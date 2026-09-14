# Netductor Admin Shortcut (iOS / iPadOS / macOS)

**RU:** [ru/SHORTCUT-IOS.md](ru/SHORTCUT-IOS.md)

**Documentation only.** Build the Shortcut once on your device. The VPS does **not** generate or redistribute `.shortcut` files.

Goal: operator Shortcut → **VPN/tunnel API** after **Face ID** + **local session token file**. End users never use this.

## Security model

1. API only on **your VPN** or SSH tunnel to `127.0.0.1:8787`.
2. **Session token** via `netductor vpn session` or Telegram `/session` — not master token inside the Shortcut.
3. **Face ID / passcode** at the start of the Shortcut.
4. Token in a **local file** (On My iPhone / On My Mac; avoid iCloud for the token file).

## Mint a session

```bash
sudo netductor vpn session 72
```

Or Telegram: `/session 72`

On the phone: Files → **On My iPhone** → `Netductor/session.txt` (token only).

## Build on iPhone

### A. Create

1. **Shortcuts** → **+** → name `Netductor Admin`.
2. **Authenticate** (Face ID / passcode). Stop on failure.

### B. Token

3. **Get File** → `On My iPhone/Netductor/session.txt` (or Ask Each Time).
4. **Get Text from Input** → **Set Variable** `Token`.

### C. Menu

5. **Choose from Menu**: List users · Add user · Disable user · Link/QR for user.

### D. API base

Default install: **`127.0.0.1:8787`**.

From iPhone typically:

- **Telegram** for day-to-day admin (no tunnel needed), or
- **SSH local forward** then `http://127.0.0.1:8787`, or
- later bind API to VPN interface (optional hardening task).

### E. HTTP

**List:** `GET http://BASE/vpn/users`  
Header: `Authorization: Bearer <Token>`

**Add:** `POST http://BASE/vpn/users`  
Body: `{"name":"alice","note":"phone"}`  
Then optional `GET .../vpn/users/alice/qr` → Quick Look.

**Disable:** `POST http://BASE/vpn/users/alice/disable`

### F. QR

PNG comes from the server; Shortcut only displays it.

## Other devices

Same Apple ID → Shortcut can sync via iCloud. **Re-mint or copy session file per device** if needed; keep session out of iCloud Drive when possible.

## Checklist

- [ ] Face ID at start
- [ ] Token from local file / Ask only
- [ ] No long-lived master in Shortcut text
- [ ] API only over VPN or tunnel
- [ ] Refresh session with `/session` when expired
