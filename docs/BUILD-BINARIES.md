# Building operator vs node binaries (v0.9+)

**Separate packages** (not only ldflags):

| Package | Binary | Release asset |
|---------|--------|----------------|
| `./cmd/netductor-op` | `netductor-op` | `netductor-op-darwin-*` / `netductor-op-linux-*` |
| `./cmd/netductor` | `netductor` | `netductor-linux-*` (node only) |

```bash
VER=0.9.0
mkdir -p dist
# Node — VPS
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
  -ldflags "-s -w -X main.version=$VER" \
  -o dist/netductor-linux-amd64 ./cmd/netductor
# Operator — Mac
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath \
  -ldflags "-s -w -X main.version=$VER" \
  -o dist/netductor-op-darwin-arm64 ./cmd/netductor-op
```

Deploy always installs **`netductor-linux-*`** on remote hosts — never `netductor-op`.
