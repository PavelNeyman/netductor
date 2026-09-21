# CLI ↔ UI parity matrix (v0.8.24)

| CLI | TUI (Mac/local) | Telegram | Admin |
|-----|-----------------|----------|-------|
| `deploy` / install | Setup wizard | — | — |
| `doctor` / `status` | ✓ | status | status tab |
| `vpn *` | partial | ✓ | users tab |
| `edge *` | list/pending/register | routers | routers + recovery |
| `secondary *` | sync/status | nodes/secondary | secondary tab |
| `mtls list/rotate` | list | Tools → mTLS | mtls section |
| `mtls rollover` | — | Tools → mTLS (hint) | CLI preferred |
| `nodes *` | list | nodes | nodes tab |
| `sites *` | MT manage | fleet/sites | sites tab |
| `backup` | ✓ | Tools | backup tab |
| `nvr *` | partial | NVR | partial |
| `probe` / `collect` | probe | — | probes tab |
| `ssh-hosts` | — | ✓ | sshhosts tab |
| `redirect-serve` | — | — | ops (default loopback) |
| `addons` | lampac | addons | addons |

Automate: `bash scripts/check-ui-parity.sh`
