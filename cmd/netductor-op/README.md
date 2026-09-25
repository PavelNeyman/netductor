# netductor-op

Operator workstation binary (Mac/PC).

```bash
go build -o netductor-op ./cmd/netductor-op
```

Commands: `deploy`, `operator serve`, `credentials`, `tui`.

Node control plane is a **separate package**: `./cmd/netductor` → release asset `netductor-linux-*`.
Deploy downloads only that node asset onto VPS hosts.
