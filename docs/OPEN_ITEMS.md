# Open items

## Done recently (0.7.3)

- [x] TG Access rich QR + `<pre><code>` URI + body/nav split
- [x] Import redirect on :80 + TG url buttons (SR/Happ/INCY)
- [x] Doctor: redirect unit + `/healthz`
- [x] Alerts: secondary offline / sing-box (collect); redirect unit on primary
- [x] mTLS: `EnsureClientFor` pushed on secondary provision
- [x] `backup verify|list` smoke
- [x] CI: redirect allowlist / encoding tests

## Still open

- [ ] Redirect **HTTPS** with real cert (optional `-tls-cert/-tls-key`; or Reality fallback later). HTTP on :80 works for TG buttons.
- [ ] Admin UI: stays **localhost / session** — do **not** expose publicly (owner decision).
- [ ] Rich **edit in place** for Access (see note in handoff) — careful with photo+buttons.
- [ ] Drop remaining operator-facing `relay` wording in old docs where harmless.
- [ ] Point `NETDUCTOR_REDIRECT_BASE` / advertise hosts at durable domain after reinstall (avoid hard-coded test IP in bot default).

## Notes for agents

**Admin TLS (item 5):** means TLS for the **local** admin API/UI only if ever bound beyond loopback. Owner does not want public admin. No work unless binding changes.

**Rich edit (item 6):** today Access does delete+send so photo/rich stay one logical screen. True `editMessageText` / edit media avoids flicker but must keep **one** message id and not send follow-ups. Risk is low if we only edit; high if we add extra sends. Optional polish.

**Hardcoded base (item 12):** bot default `http://2.27.118.70` is a **test VPS IP**. After reinstall/DNS, set systemd env `NETDUCTOR_REDIRECT_BASE` (and vpn/core advertise hosts) explicitly — do not rely on compiled defaults.
