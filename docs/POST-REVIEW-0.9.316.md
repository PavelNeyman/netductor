# Post-review fixes (0.9.316)

## Security / ops
- Backup verify: max archive 4 GiB, max tar members 50k
- Incident collect: no `bash -c` (direct exec)
- `/api/update/status` and `/api/update/releases` require operator session
- Mac build queue: sticky alert `git:mac-build-pending`; clear when empty
- VPN devices: prune >30d + max 500 entries
- Canary: only known non-edge VPN users
- Site room photos: JPEG/PNG/GIF/WebP magic check
- Backup verify timer: install path + primary baseline only

## Canary seed
On primary install/baseline, if `canary-users.json` is missing, it is filled with all human VPN users once. Later users are **not** auto-added — use TG/Web Canary menu.

## Smoke
`netductor smoke dual` — version, local API health, secondary online, channel status, sing-box active (not full client VPN e2e).

## Formula
Homebrew `netductor-op` bumped with release assets.


## Follow-up 0.9.317
- decryptFile: no full ReadFile for archives ≥64 MiB (openssl stream path)
- backup verify: streaming tar member count (no CombinedOutput dump)
- mac queue: AlertRefresh + 2m cooldown for git:mac-build*
- opcatalog.Day2Groups shared by TG Tools
