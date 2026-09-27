**EN** · [RU](ru/OPCATALOG.md)

# OpCatalog matrix (day-2)

Deploy (fleet/primary/secondary/edge) is Mac-only — see [DEPLOY-PARITY.md](DEPLOY-PARITY.md).

| ID | API | CLI hint | Web | TG | CLI | TUI |
|----|-----|----------|-----|----|-----|-----|
| `health` | `GET /health` | `—` | ✅ | ✅ | ✅ | ✅ |
| `doctor` | `GET /api/doctor` | `doctor` | ✅ | ✅ | ✅ | ✅ |
| `domain` | `GET /api/domain` | `domain show` | ✅ | ✅ | ✅ | ✅ |
| `bot` | `GET /api/bot-status` | `—` | ✅ | ✅ | ✅ | ✅ |
| `status` | `GET /api/status` | `status` | ✅ | ✅ | ✅ | ✅ |
| `metrics` | `GET /api/metrics` | `—` | ✅ | — | ✅ | ✅ |
| `metrics-hist` | `GET /api/metrics/history` | `—` | ✅ | — | ✅ | ✅ |
| `addons` | `GET /api/addons` | `addons list` | ✅ | ✅ | ✅ | ✅ |
| `addons-lampac` | `GET /api/addons/lampac` | `—` | ✅ | ✅ | ✅ | ✅ |
| `sni` | `GET /api/sni` | `—` | ✅ | ✅ | ✅ | ✅ |
| `sni-presets` | `GET /api/sni-presets` | `—` | ✅ | ✅ | ✅ | ✅ |
| `latest` | `GET /api/latest` | `—` | ✅ | ✅ | ✅ | ✅ |
| `sessions` | `GET /api/sessions` | `—` | ✅ | ✅ | ✅ | ✅ |
| `vpn-users` | `GET /vpn/users` | `vpn list` | ✅ | ✅ | ✅ | ✅ |
| `vpn-refresh` | `POST /api/vpn/refresh-links` | `vpn refresh-links` | ✅ | ✅ | ✅ | ✅ |
| `nodes` | `GET /api/nodes` | `nodes list` | ✅ | ✅ | ✅ | ✅ |
| `nodes-self` | `GET /api/nodes/self` | `—` | ✅ | ✅ | ✅ | ✅ |
| `secondary` | `GET /api/secondary/status` | `secondary status` | ✅ | ✅ | ✅ | ✅ |
| `secondary-links` | `GET /api/secondary/links` | `—` | ✅ | ✅ | ✅ | ✅ |
| `ssh-hosts` | `GET /api/ssh-hosts` | `ssh-hosts list` | ✅ | ✅ | ✅ | ✅ |
| `ssh-clear` | `POST /api/ssh-hosts/clear` | `ssh-hosts clear` | ✅ | ✅ | ✅ | ✅ |
| `mtls-certs` | `GET /api/mtls/certs` | `mtls list` | ✅ | ✅ | ✅ | ✅ |
| `sites` | `GET /api/sites` | `—` | ✅ | ✅ | ✅ | ✅ |
| `edge-pending` | `GET /api/edge/pending` | `edge pending` | ✅ | ✅ | ✅ | ✅ |
| `edge-devices` | `GET /api/edge/devices` | `edge list` | ✅ | ✅ | ✅ | ✅ |
| `edge-metrics` | `GET /api/edge/metrics` | `—` | ✅ | ✅ | ✅ | ✅ |
| `edge-guest-st` | `GET /api/edge/guest/status` | `—` | ✅ | ✅ | ✅ | ✅ |
| `nvr-cameras` | `GET /api/nvr/cameras` | `nvr cameras` | ✅ | ✅ | ✅ | ✅ |
| `nvr-config` | `GET /api/nvr/config` | `nvr status` | ✅ | ✅ | ✅ | ✅ |
| `nvr-storage` | `GET /api/nvr/storage` | `—` | ✅ | ✅ | ✅ | ✅ |
| `nvr-events` | `GET /api/nvr/events` | `—` | ✅ | ✅ | ✅ | ✅ |
| `nvr-segments` | `GET /api/nvr/segments` | `—` | ✅ | ✅ | ✅ | ✅ |
| `nvr-retention` | `POST /api/nvr/retention/run` | `—` | ✅ | ✅ | ✅ | ✅ |
| `nvr-go2rtc` | `POST /api/nvr/go2rtc` | `nvr go2rtc` | ✅ | ✅ | ✅ | ✅ |
| `git-repos` | `GET /api/git/repos` | `git list` | ✅ | ✅ | ✅ | ✅ |
| `git-pipelines` | `GET /api/git/pipelines` | `git pipelines` | ✅ | ✅ | ✅ | ✅ |
| `reg-status` | `GET /api/registry/status` | `registry status` | ✅ | ✅ | ✅ | ✅ |
| `reg-ensure` | `POST /api/registry/ensure` | `—` | ✅ | ✅ | ✅ | ✅ |
| `dns-lists` | `GET /api/dns/lists` | `dns list` | ✅ | ✅ | ✅ | ✅ |
| `dns-set` | `POST /api/dns/set` | `dns set` | ✅ | ✅ | ✅ | ✅ |
| `dns-reload` | `POST /api/dns/reload` | `dns reload` | ✅ | ✅ | ✅ | ✅ |
| `backup-schedule` | `GET /api/backup/schedule` | `backup schedule` | ✅ | ✅ | ✅ | ✅ |
| `backup-list` | `GET /api/backup/list` | `backup list` | ✅ | ✅ | ✅ | ✅ |
| `backup-peer` | `GET /api/backup/peer` | `—` | ✅ | ✅ | ✅ | ✅ |
| `backup-run` | `POST /api/backup/run` | `backup now` | ✅ | ✅ | ✅ | ✅ |
| `sec-export` | `GET /api/secondary/export` | `—` | ✅ | ✅ | ✅ | ✅ |
| `probes` | `GET /api/probes` | `probe` | ✅ | ✅ | ✅ | ✅ |
| `probes-cfg` | `GET /api/probes/config` | `—` | ✅ | ✅ | ✅ | ✅ |
| `audit` | `GET /api/audit` | `audit tail` | ✅ | ✅ | ✅ | ✅ |

### Intentional gaps (not debt)

| ID | Missing | Why |
|----|---------|-----|
| `metrics` / `metrics-hist` | TG | Covered inside **Status** — no second button |
| Deploy fleet/primary/… | TG | Day-2 only on node; deploy = Mac op |

**Rule:** new day-2 capability → entry in `internal/opcatalog` **before** UI-only code. `—` in matrix without a row above = real parity debt.
