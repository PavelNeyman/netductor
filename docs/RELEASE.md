
## Checklist (every release)

1. Bump `VERSION` **and** `internal/version.Release` (deploy URL pin).
2. Build: node, tg, agent, op (linux-amd64 + darwin-arm64 as needed).
3. Upload **all** assets to GitHub Release (missing `netductor-tg` breaks primary TG install).
4. Update `Formula/netductor-op.rb` version + sha256.
5. Smoke: `netductor update check` on primary after apply.


## Release gate (0.9.593+)

Before production install from a tag:

1. `VERSION` ≡ `internal/version.Release` — `./scripts/check-versions.sh`
2. Build with `-X …version.Release=$VER` (Makefile, `scripts/release.sh`, GitHub Actions)
3. `./scripts/verify-release-binaries.sh` on `dist/` (runtime `version` or strings)
4. Tag `vX.Y.Z` matches commit and published assets + `SHA256SUMS`
5. SHA256SUMS proves integrity of the *downloaded* set, not cryptographic provenance of the publisher (R-05: signed manifest optional/follow-up)

Local publish: `VERSION=… GITHUB_TOKEN=… ./scripts/release.sh`

### Local release.sh rules (0.9.595+)

- Working tree must be **clean**; VERSION and `internal/version.Release` already match.
- Build is from **HEAD**; release `target_commitish` is HEAD SHA.
- Any asset upload failure (or 422 with SHA mismatch) **fails the script**.
- `verify-release-binaries.sh` requires the full matrix (or `ND_RELEASE_MINIMAL=1` for amd64 smoke).
