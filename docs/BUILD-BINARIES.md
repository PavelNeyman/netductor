# Building operator vs node binaries (v0.9+)

Same package `./cmd/netductor`, different link-time role:

```bash
VER=0.9.0
# Node (VPS) — release asset netductor-linux-$ARCH
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath \
  -ldflags "-s -w -X main.version=$VER -X main.binaryRole=node" \
  -o dist/netductor-linux-amd64 ./cmd/netductor

# Operator (Mac) — release asset netductor-op-darwin-$ARCH
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath \
  -ldflags "-s -w -X main.version=$VER -X main.binaryRole=operator" \
  -o dist/netductor-op-darwin-arm64 ./cmd/netductor
```

Deploy always fetches **`netductor-linux-*`** (node). Never install `netductor-op` on a VPS.

Update `.github/workflows/release-netductor.yml` the same way (token needs `workflow` scope to push workflow files).
