# 0.9.341

- Status **Action required** block + TG `m:actions` / API `GET /api/fleet/actions`
- `netductor stack preflight [tag]` — release+SHA256SUMS, pin, secondary, backup age; blocks `stack apply` unless `NETDUCTOR_STACK_SKIP_PREFLIGHT=1`
- VPN profile **kind** user|device: `vpn kind <name> user|device`, API `POST /vpn/users/<name>/kind`, Users list hides device; TG `m:vpn_devices`
- Local release build: `go test ./internal/... ./cmd/...` gate before matrix build
