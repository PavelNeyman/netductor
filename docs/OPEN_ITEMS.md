# Open items

Baseline: **v0.8.20**.

## Software quality gates (green)
- `go test ./...`
- `go vet ./...`
- `scripts/check-version-pins.sh`
- `scripts/check-ui-parity.sh`

## Operator-only
- Hardware e2e
- Domain HTTPS redirect when domain exists
