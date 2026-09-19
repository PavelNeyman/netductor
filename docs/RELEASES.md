# Releases

Repo: **https://github.com/PavelNeyman/netductor**

## How releases are published

### CI (preferred)

Tag `v*` → `.github/workflows/release-netductor.yml` → assets via `GITHUB_TOKEN`.

### REST API

`POST /repos/PavelNeyman/netductor/releases` + upload assets (`contents: write`).

## Install

```bash
TAG=v0.7.0-dev
curl -fsSL -o /usr/local/bin/netductor \
  "https://github.com/PavelNeyman/netductor/releases/download/${TAG}/netductor-linux-amd64"
chmod 755 /usr/local/bin/netductor
netductor version
```

## Manual release (operator)

GitHub PAT without `workflow` scope cannot update Actions YAML.

1. Tag: `git tag v0.7.35-dev && git push origin v0.7.35-dev` — existing `release-netductor.yml` on tag `v*`.
2. Or local: `./scripts/build-release-local.sh 0.7.35-dev` then upload assets to a Release by hand.
