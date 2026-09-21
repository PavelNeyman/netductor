# Security

## Defaults
- API bind `127.0.0.1` only; public bind requires `NETDUCTOR_API_PUBLIC=1` **and TLS cert/key**
- Sessions: 256-bit, hash-at-rest, max 72h; cookie helper `HttpOnly` + `SameSite=Strict` (+ `Secure` with TLS)
- Edge: bootstrap ≠ device token; enroll rate-limit; human approve; TG notify on new pending
- Global `edge_token` off unless `NETDUCTOR_EDGE_LEGACY_TOKEN=1`
- Destructive agent cmds need `confirm=yes`: reboot, agent_update, sysupgrade
- Edge command allowlist on enqueue
- API errors truncated via `publicErr`
- Audit: vpn.add, edge.approve/deny/revoke/rotate, nodes.rename, session.revoke
- Update integrity: `NETDUCTOR_UPDATE_SHA256=<hex> netductor update`

## Optional Lampac
- Off by default: `NETDUCTOR_LAMPAC=1` or `netductor install lampac`
- Image: `ghcr.io/lampac-nextgen/lampac:latest` (override `NETDUCTOR_LAMPAC_IMAGE`)
- Port 9118 (Docker). Installs `docker.io` + `docker-cli` on Debian.


## SSH TOFU (MikroTik / secondary provision)
- First connect stores host key under StateDir
- Manage without editing files: `netductor ssh-hosts list|forget|clear`
- Also: TUI, Admin tab **SSH hosts**, Telegram **Nodes → SSH hosts**
- `NETDUCTOR_MT_STRICT=1` rejects unknown MikroTik hosts
- Details: [TOFU.md](TOFU.md)


## Agent plane (0.7.1+)
- Preferred: **mTLS :8789** (client cert + device token)
- Plain :8788 disabled when certs exist (`NETDUCTOR_PLAIN_AGENT=1` to force)
- Firewall: only secondary IP (and localhost)
- Server cert SAN includes `NETDUCTOR_PUBLIC_HOSTNAME` / `public_hostname` (default `netductor.work.gd`)

- Per-node client certs: `netductor mtls issue-client <node-id>`
- Optional HTTPS admin: `netductor tls self-signed` + `NETDUCTOR_API_PUBLIC=1` + TLS env (prefer certbot in prod)
