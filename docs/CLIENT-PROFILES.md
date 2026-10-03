**EN** · [RU](ru/CLIENT-PROFILES.md)

See docs/ru/CLIENT-PROFILES.md


## Service access policy

Per-user internal service ACL (Lampac, git, …) is described in [PLAN-SERVICE-ACCESS-POLICY.md](PLAN-SERVICE-ACCESS-POLICY.md). Operators set checkboxes in TG/Web; enforcement is sing-box `auth_user` routes. Optional **service-net** VIPs (`10.88.0.0/24`, `netductor servicenet apply`) DNAT to loopback listeners on primary.
