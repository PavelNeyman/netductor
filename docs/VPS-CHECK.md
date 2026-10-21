# VPS pre-check

Go implementation of the usual VPS scorecard (no `curl | bash` third-party scripts).

## What it runs

| Check | Notes |
|--|--|
| tools | `apt-get install curl iperf3 sysbench` when missing |
| ip_region | ip-api.com (country, ISP, proxy/hosting flags) |
| sysbench_cpu | `--threads=1` |
| disk_write | 256MiB `dd` sample (local path only) |
| dns | resolve example.com |
| latency | TCP to CF/Google; optional primary :443/:52222 |
| http | ya.ru/vk (secondary), CF/GitHub (primary), ipify |
| iperf3 | public/RU servers when reachable (warn if none) |

**Not** a line-by-line clone of YABS/bench.sh/IPQuality/censorcheck — same *intent*, structured JSON, safe from Mac via SSH.

## CLI

```bash
# from Mac (operator)
netductor-op vps-check --host 1.2.3.4 --password '…' --profile secondary --primary 2.27.118.70
netductor-op vps-check --host 1.2.3.4 --key ~/.ssh/netductor_primary --profile primary

# on the node (day-2)
netductor vps-check --profile secondary --primary 2.27.118.70 --json
```

## UI

- Web: **Deploy → VPS check** and **Secondary → VPS pre-check**
- TUI: wizard **VPS pre-check**; Secondary field `precheck=yes|only|no`
- Job: `vps.precheck`

## Day-2 hoster residual

Still separate: `netductor host-audit` (see HOST-AUDIT.md).
