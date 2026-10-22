# VPS pre-check

Go scorecard (no `curl | bash` third-party scripts). Intent aligned with common VPS chat suites (region, IP quality, geoblock reachability, DPI hints, sysbench, iperf).

## Checks (0.9.466+)

| Group | IDs | Notes |
|--|--|--|
| Tools | `tools` | apt: curl, iperf3, sysbench, dnsutils; optional fio |
| Region | `ip_region` | ip-api.com country/ISP/AS |
| IP quality | `ipq_dnsbl`, `ipq_proxy`, `ipq_org`, `ipq_egress` | Spamhaus ZEN/SBL/XBL, CBL, Barracuda; proxy flag; ASN; egress IP match |
| Bench | `sysbench_cpu`, `sysbench_mem`, `cpu_info`, `disk_fio`, `disk_write` | CPU+memory sysbench; fio if present; 256MiB dd |
| Network | `dns`, `latency_*`, `tcp_bulk`, `http_*` | DNS, TCP RTT, HTTP |
| Geoblock | `geo_*` | YouTube, X, Discord, Netflix CDN, OpenAI, Instagram, Yandex, VK, Gosuslugi, Rutube |
| DPI (RU) | `dpi_tls_*`, `dpi_dns_*` | TLS SNI handshake; DNS pollution (loopback/private) |
| iperf | `iperf_ru` / `iperf_pub` | RU or public servers; warn if none respond |

**Profiles:** `secondary` (RU entry), `primary` (abroad), `generic`.

## CLI

```bash
# Mac → fresh VPS
netductor-op vps-check --host IP --password '…' --profile secondary --primary PRIMARY_IP

# Day-2 on node (full Go path)
netductor vps-check --profile secondary --primary PRIMARY_IP --json
```

Remote SSH prefers `netductor vps-check` when the binary exists; otherwise an embedded Python probe (same categories, slightly thinner).

## UI

- Web: **Deploy → VPS check**, **Secondary → VPS pre-check**
- TUI: **VPS pre-check** wizard; Secondary `precheck=yes|only|no`
- Job: `vps.precheck`

## Day-2 hoster residual

`netductor host-audit` — separate from channel scorecard (HOST-AUDIT.md).


## Security (0.9.467+)

Disk/fio tests use private `MkdirTemp` dirs (no fixed `/tmp` names). Password SSH requires `sshpass` or fails explicitly. DNSBL treats DNS errors as WARN, not clean.
