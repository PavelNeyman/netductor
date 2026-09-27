**EN** · [RU](ru/DEPLOY-PARITY.md)

# Deploy parity (Mac TUI ↔ Web ↔ CLI)

**Backend:** `operator.FleetDeployWithReport` / `DeployPrimary` / `DeploySecondary` only.  
UIs only fill the same fields.

## Fleet checklist (single list)

| Field | TUI key | Web name | API JSON |
|-------|---------|----------|----------|
| Deploy primary | `do_primary` | `do_primary` | `do_primary` |
| Deploy secondary | `do_secondary` | `do_secondary` | `do_secondary` |
| Primary host | `host` | `primary_host` | `primary_host` |
| Primary password | `password` | `primary_password` | `primary_password` |
| Secondary host | `sec_host` | `secondary_host` | `secondary_host` |
| Secondary password | `sec_password` | `secondary_password` | `secondary_password` |
| SNI | `sni` | `sni` | `sni` |
| Domain base | `domain_base` | `domain_base` | `domain_base` |
| LE email | `le_email` | `le_email` | `le_email` |
| SSH key path | `key_path` | `key` | `key` |
| Key passphrase | `key_pass` | `key_passphrase` | `key_passphrase` |
| Lampac | `with_lampac` | `with_lampac` | `with_lampac` |
| Git/registry | `with_git` | `with_git` | `with_git` |
| Telegram | `with_telegram` + token/admin | same | `with_telegram`, `tg_token`, `tg_admin` |
| CF proxy i. | `cf_proxy` | `cf_proxy` | `cf_proxy` |

CLI: `netductor-op deploy fleet` with the same flags/env as documented in operator help.

Any new deploy field → **all three** surfaces + this table in the same PR.
