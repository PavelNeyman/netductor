# Telegram operator bot UI

## Layout rule

| Placement | Purpose | Examples |
|-----------|---------|----------|
| `reply_markup` (under message) | Navigation | Main menu, Users, Nodes, back |
| Message body (`<tg-button>`, tables, rich media) | Actions for current screen | Access, rename, enable/disable, VLESS/Core/HY2, app import, node upgrade |

Never put the same action in both places (e.g. Add user + Main menu duplicated).

## Rich Messages (Bot API 10.x)

- Prefer `sendRichMessage` / `editMessageText` with `rich_message.html`.
- In-body actions: `<tg-button type="callback_data" data="…">` (attribute `data`, not inventing `href` for callbacks).
- URL buttons: `<tg-button type="url" url="https://…">` — attribute is **`url`**, not `href`.
- Telegram **rejects** custom schemes (`shadowrocket://`, `happ://`, `incy://`, browser schemes) in url-buttons (`BUTTON_URL_INVALID`). Only `http`/`https` (plus limited `tg://` / `ton://`).
- QR in the same message: `<img src="tg://photo?id=qr1"/>` + `rich_message.media[]` with `attach://qr1` (multipart).

## Import redirect (:80)

Custom-scheme one-tap requires an **HTTP 302** hop:

| Piece | Detail |
|-------|--------|
| Command | `netductor redirect-serve -listen :80` |
| Unit | `netductor-redirect.service` |
| Path | `GET /r?u=<base64url(deep-link)>` → **302** `Location: <deep-link>` |
| Allowlist | `shadowrocket://`, `happ://`, `incy://`, `vless://`, `hysteria2://`, `hy2://`, `ss://`, `trojan://` |
| Bot env | `NETDUCTOR_REDIRECT_BASE` (e.g. `http://2.27.118.70` or `http://netductor.work.gd`) |

Access body buttons **Shadowrocket / Happ / INCY** point at `{REDIRECT_BASE}/r?u=…` built from `shadowrocket://add/<url-encoded URI>` (same for happ/incy).

Port **80** is already opened for ACME; no extra public port. Do not log the `u` query (contains key material).

## Code

- `cmd/netductor-tg/keyboards.go` — under-message keyboards  
- `cmd/netductor-tg/format_*.go` — HTML + in-body buttons  
- `cmd/netductor-tg/handlers_cb.go` / `handlers_msg.go` — callbacks  
- `cmd/netductor-tg/main.go` — `sendRich` / `sendRichWithPhoto`  
- `cmd/netductor/cli_redirect.go` — redirect-serve  

## Screens

- **Users list:** table/rows + in-body open/access/rename/add; under: Main menu  
- **User hub:** enable/disable/revoke/access in body; under: Users + Main menu  
- **Access + QR:** photo + `<pre><code>URI</code></pre>` + app url-buttons (via redirect) + VLESS/Core/HY2 in body; under: User card + Users + Menu  
- **Node card:** metrics/journal/upgrade/reboot in body; under: Nodes + Menu  


## In-place updates (all screens)

| Source | Behavior |
|--------|----------|
| Callback buttons | `reply()` → `editMessageText` / rich edit; on failure delete + one send |
| Access QR modes | `editRichWithPhoto` → same `message_id`; fallback delete + send |
| User slash commands (`/status`, …) | New reply (user message is not edited) |

Do not send a second bot message on the happy path for menu navigation.


## Gold standard (2026-09-16): DNS lists screen

- **Rich message** table (`<table bordered striped>`) with descriptions.
- **Actions inside table cells** via `<tg-button type="callback_data">` (not only under-message keyboard).
- Under-message keyboard: **navigation only** (Main menu).
- Batch config changes, then explicit **Reload** (no service flap on every toggle).
- Same pattern target: Guest, Backup, Locations, Nodes.
