
## Checklist (every release)

1. Bump `VERSION` **and** `internal/version.Release` (deploy URL pin).
2. Build: node, tg, agent, op (linux-amd64 + darwin-arm64 as needed).
3. Upload **all** assets to GitHub Release (missing `netductor-tg` breaks primary TG install).
4. Update `Formula/netductor.rb` version + sha256.
5. Smoke: `netductor update check` on primary after apply.

