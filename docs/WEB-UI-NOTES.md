# WebUI notes: index.html & CSP

## Why one big `index.html`

Operator UI is **embedded** in `netductor-op` (`//go:embed`). One file keeps:

- offline install (no asset server)
- single token inject (`/*__ND_TOKEN__*/`)
- no separate static host

**Downside:** hard to edit; growth. **Later:** split into `app.js` + `style.css` still embedded as multiple files, or generate from templates. Not required for security.

## CSP today (0.9.10+)

```
default-src 'none'
base-uri 'none'
form-action 'self'
frame-ancestors 'none'
img-src 'self' data:
style-src 'unsafe-inline'
script-src 'unsafe-inline'
connect-src 'self'
```

Plus: `X-Frame-Options: DENY`, `COOP: same-origin`, `Permissions-Policy` (no cam/mic/geo).

### Why `'unsafe-inline'` still

Scripts and styles live **inside** the HTML string. Without a **nonce** or **hash** CSP, browsers block inline JS if `script-src` has no `'unsafe-inline'`.

**Path to strict CSP (no unsafe-inline):**

1. Embed separate `app.js` / `app.css`
2. Serve them as `/static/...` with hashes or nonces in CSP
3. HTML only references external scripts — no inline handlers ideally

Until then: loopback bind + op token + session in **sessionStorage** limit impact of XSS.

## Session storage

- Connection settings → `localStorage` (no secrets)
- Node session token → **`sessionStorage` only** (+ optional file via Load session)
