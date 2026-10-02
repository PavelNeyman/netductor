**EN** · [RU](ru/BREW.md)

**RU:** в Formula только реальные SHA; `:no_check` ломает `brew install`. После релиза: `sha256sum netductor-op-*` → вставить в Formula.

# Homebrew formula (Mac)

**Current Formula version tracks GitHub latest op release** (see `Formula/netductor-op.rb`).

## Hard rule (locked)

**Never** ship `sha256 :no_check` in `Formula/netductor.rb`.

Homebrew rejects it (`Formula reports different checksum: no_check`). Digests **must** match the GitHub Release assets for that version.

## Release checklist

1. Build all `netductor-op-*` targets (darwin-arm64, darwin-amd64, linux-amd64, linux-arm64).
2. Upload assets to the tag release.
3. `sha256sum netductor-op-*` → paste into Formula.
4. Upload `SHA256SUMS` to the same release (enables `netductor update` verify).
5. Bump `version` in Formula to the same tag.

Optional: `netductor-op update` also works without brew (downloads the same assets).

## Same operator key + SSH port (primary and secondary)

After harden, **both** VPS use:

- Port **52222**
- Key **`~/.ssh/netductor_primary`** (Mac private key; pubkey on both hosts)

Bootstrap only: secondary first password login may still be **:22**; after deploy, day-2 is **:52222**.