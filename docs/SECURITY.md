**EN** · [RU](ru/SECURITY.md)

# Security

## Product posture (freeze)

- Node API: **127.0.0.1 only** — non-local bind is refused (no enable flag)
- Agent plane: **mTLS :8789 only** — plain :8788 is not implemented
- Rate-limit uses **RemoteAddr only** (no X-Forwarded-For / proxy trust)
- VPS `/admin` UI removed — operator is **netductor-op** on Mac
- Redirect: **HTTPS :8443** after LE; no public :80
- Recovery `:8790`: arm over SSH; encrypted backup; decryption key **offline only**
- Edge auth: **per-device** tokens after approve (no global edge_token)
- SSH: **52222** key-only after harden

## Sessions / edge

- Sessions: strong token, hash-at-rest, TTL
- Enroll rate-limit; human approve; destructive cmds need confirm
- Edge command allowlist on enqueue

## Conf

`ndconfig.Load` **never loads** removed knobs into env, even if present in an old `netductor.conf`:
`PLAIN_AGENT`, `API_PUBLIC`, `API_ALLOW_PUBLIC`, `TRUST_PROXY`, `LEGACY_ADMIN_UI`, recovery key flags, `REDIRECT_LISTEN`, `EDGE_LEGACY_TOKEN`.

There is **no** supported way to turn those back on in product code.

## Backup / LE

No private operator keys in archive. LE certs not stored — re-issue on recover when `DOMAIN` + `LE_EMAIL` are set.

## Lampac / SSH TOFU

- Lampac optional via install component / op wizard
- SSH host keys: `netductor ssh-hosts list|forget|clear`

See [ARCHITECTURE-FREEZE.md](ARCHITECTURE-FREEZE.md).
