**EN** · [RU](ru/CLIENT-PROFILES.md)

See docs/ru/CLIENT-PROFILES.md


## Service access policy

Per-user internal service ACL (Lampac, git, …) is described in [PLAN-SERVICE-ACCESS-POLICY.md](PLAN-SERVICE-ACCESS-POLICY.md). Operators set checkboxes in TG/Web; enforcement is sing-box `auth_user` routes. Optional **service-net** VIPs (`198.18.88.0/24`, `netductor servicenet apply`) DNAT to loopback listeners on primary.

## SR Config profiles (0.9.209+)

| ID | Audience | Notes |
|--|--|--|
| `operator-mobile` | Operator phone/Mac on LTE or foreign Wi‑Fi | RU/banks DIRECT, home `10.9.8.0/24` DIRECT, work CIDRs DIRECT, ads REJECT, service-net + shop PROXY, FINAL PROXY |
| `operator-fullproxy` | Operator on foreign Wi‑Fi “all via VLESS” | No client RU DIRECT (secondary split), work+home DIRECT, service-net PROXY, FINAL PROXY |
| `family` | Family phones | RU/banks DIRECT, ads REJECT, FINAL PROXY; no work/shop/service-net |

**CLI:** `netductor vpn sr-config list` · `netductor vpn sr-config family` · `-o /path.conf`

**Work CIDRs:** `/etc/netductor/work-direct-cidrs` or `NETDUCTOR_WORK_DIRECT_CIDRS` (one CIDR per line).

**TG:** Users → Access → **SR Config** (operator: picker; others: Family).

**Apple TV:** no SR Config — VLESS only; DNS/ads/split on secondary.

**Scene / On Demand:** configure on device (SSID → disconnect at home router-VPN; cellular → Connect + operator-mobile).
