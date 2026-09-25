# WebUI: assets & CSP

## Layout (0.9.12+)

| File | Role |
|------|------|
| `index.html` | Markup only; token in `<meta name="nd-token">` |
| `app.css` | Styles (`/static/app.css`) |
| `app.js` | Logic (`/static/app.js`); reads token from meta |

All embedded via `//go:embed index.html app.css app.js`.

## CSP (no `unsafe-inline`)

```
default-src 'none'
base-uri 'none'
form-action 'self'
frame-ancestors 'none'
img-src 'self' data:
style-src 'self'
script-src 'self'
connect-src 'self'
```

Plus: `X-Frame-Options: DENY`, `COOP: same-origin`, `Permissions-Policy`.

Token is not injected into JS source — only into meta content on HTML response (`Cache-Control: no-store`).
