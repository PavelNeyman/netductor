# Releases

Published tags (see GitHub Releases): **`v0.8.12`** (current), `v0.8.0`, older `v0.7.x` / `v0.5.x`.

Artifacts typically:

- `netductor-linux-amd64` / `arm64`
- `netductor-tg-linux-amd64`
- `netductor-agent-linux-*` (edge arches)
- `SHA256SUMS`

## Install a specific version

```bash
export NETDUCTOR_VERSION=0.8.12
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/bootstrap.sh | bash
```

Or:

```bash
TAG=v0.8.12
wget -qO /usr/local/bin/netductor \
  https://github.com/PavelNeyman/netductor/releases/download/${TAG}/netductor-linux-amd64
chmod 755 /usr/local/bin/netductor
netductor version
```

## Publishing (maintainers)

1. Bump `VERSION`, CHANGELOG, docs pins if needed.
2. Tag: `git tag v0.8.12 && git push origin v0.8.2` — workflow on tag `v*`.
3. Confirm Release assets + `SHA256SUMS` (primary self-update verifies checksums).

Primary update path for operators: TG Tools → Updates (GitHub Release + SHA256). Agents: **manual** `agent_update` only — [UPGRADE.md](UPGRADE.md) · [OPEN_ITEMS.md](OPEN_ITEMS.md).
