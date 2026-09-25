# netductor-op

The **operator** workstation binary is built from `./cmd/netductor` with:

```bash
go build -ldflags "-X main.binaryRole=operator -X main.version=0.9.0" \
  -o netductor-op ./cmd/netductor
```

Release assets: `netductor-op-darwin-*`, `netductor-op-linux-*`.

Node binary (VPS) uses `-X main.binaryRole=node` → asset `netductor-linux-*`.
Deploy always installs the **node** asset on remote hosts.
