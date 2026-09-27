# Unified update path (A6)

All components install under **FHS** `/usr/local/bin` (no `/opt/netductor`).

## Node (primary / secondary)

```bash
netductor update                          # pin = internal/version.Release
netductor update 0.9.65                   # explicit
netductor update --component tg --restart
netductor update --component agent        # on secondary host
NETDUCTOR_UPDATE_SHA256=<hex> netductor update --version 0.9.65
```

Downloads:  
`https://github.com/PavelNeyman/netductor/releases/download/vX/netductor-linux-<arch>`  
(+ `netductor-tg-linux-*`, `netductor-agent-linux-*`).

## Operator Mac

```bash
netductor-op update   # when implemented: netductor-op-<goos>-<goarch>
# or brew reinstall netductor
```

## Deploy-time

`internal/deploy` already curls the same Release assets during primary/edge provision.
