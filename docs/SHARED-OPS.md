# Shared ops modules (anti-drift)

## Rule

Any behaviour applied on **more than one device type** (primary VPS, secondary VPS, future hosts) must live in **one** package and be called from deploy/install/provision — not copy-pasted scripts.

## Examples

| Concern | Module | Callers |
|---------|--------|---------|
| SSH harden Port+key-only | `internal/hardening` (`DropInConf`, `RemoteHardenScript`) | `install/ssh_harden.go`, `secondary.DisablePasswordAuth` |
| Default SSH port | `hardening.DefaultSSHPort` / `SSHPort()` | doctor, deploy, tunnel |
| Day-2 API actions | `internal/opcatalog` | Web Control, TG Tools |
| Deploy primary/secondary/edge | `internal/operator` + `internal/deploy` | Web, TUI, CLI |

## Why

Secondary once hardened **password only on :22** while primary used **:52222** — two scripts diverged. Shared `DropInConf` prevents that class of bug.

## Next candidates

- ufw allow lists for node ports
- fail2ban jail snippets
- sing-box base service unit templates
