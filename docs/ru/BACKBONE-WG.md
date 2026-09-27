**RU** · [EN](../BACKBONE-WG.md)

# Backbone WireGuard / AmneziaWG — design spike

**Статус:** design + **CLI v1** (`netductor backbone …`); soak — вручную  
**Инцидент:** secondary→primary VLESS uplink mux timeout при живом primary OS/API (2026-09)  
**Код:** `secondary_box.go` (uplink VLESS+mux, без vision), `agent.go` (probe + watchdog restart)

---

## 1. Проблема

Клиентский Reality может быть жив, а **control/transit primary↔secondary** сидит на том же **VLESS Reality + multiplex** uplink:

| Симптом | Смысл |
|---------|--------|
| Secondary: `outbound/vless[uplink]: timeout ~60s` | Застрявший mux / middlebox / DPI на **служебном** туннеле |
| TG bot + API primary работают | Хост жив; проблема **путь secondary→primary:443** |
| Watchdog рестартит sing-box | Снимает залипание, **не** даёт независимый control plane |

Клиентский Reality на secondary остаётся. Backbone **не** замена user VLESS.

---

## 2. Цели / не-цели

**Цели:** независимый канал primary↔secondary для agent heartbeat/commands, backup/recovery, лёгкий sync; переживать деградацию Reality uplink; опционально AWG если plain WG режут.

**Не-цели:** WG как клиентский VPN; публичный WG для ноутбуков оператора; mesh OpenWrt edges; CDN/XHTTP entry.

---

## 3. Топология

```
[Clients] --VLESS/Reality--> [Secondary RU] --VLESS mux--> [Primary EU]   // user data
                                 |                            ^
                                 +-------- backbone WG/AWG ---+   // service only
```

Listen на primary (EU); secondary dial-out. Table=off / policy routing — не default route.

---

## 4. Что едет по backbone (v1)

Agent mTLS, recovery pull, опционально backup peer, health. **Не** user bulk.

---

## 5. WG vs AmneziaWG

Обычный WG проще; AWG — если path фильтрует WG. Spike только после доказательства.

---

## 6. Ключи и lifecycle

Генерация на primary; раздача на secondary через provision/agent; ротация осознанная.

---

## 7. Failure modes

| Случай | Поведение |
|--------|-----------|
| Backbone down, Reality up | User VPN OK; agent → public URL |
| Reality stuck, backbone up | Agent/backup через backbone; user страдает до watchdog |
| Оба down | Secondary offline alerts |
| Смена IP primary | Обновить endpoint на secondary |

Fail-open для users: backbone не blackhole клиентский трафик.

---

## 8. Multiplex / h2mux

Uplink: multiplex без vision (`apply.go` / `secondary_box.go`). Stuck mux — боль инцидента.

### Схема multiplex (sing-box 1.14+)

- **Outbound** (secondary uplink): `enabled`, `padding`, `max_connections`, `min_streams`, `max_streams`, опционально `protocol` h2mux.
- **Inbound** (primary `vless-reality`): **только** `enabled` + `padding`. `max_connections` на inbound → `json: unknown field`, **`vpn apply` падает**.

Mux **остаётся включённым** (канон против отвалов). Отключать только для A/B диагностики (`uplink-mux set off`).

---

## 9. Реализация (эскиз)

`internal/backbone`, kernel WG на Debian, CLI `backbone status|init|show`, doctor, status в TG/Web без export ключей.

---

## 10. Spike checklist

wireguard на обеих → /30 → allow UDP → ping → agent URL на backbone IP → stress Reality → tcpdump user path.

---

## 11. Decision log

См. EN таблицу дат; dual SP/PS WSS — реализованный service plane (ниже).

---

## 12. CLI

```bash
netductor backbone status|init|show
netductor uplink-mux set on|off|h2mux
netductor svc-paths status|apply|failover …
```

---

## 13. Dual service paths over WSS (2026-09-27)

| Path | Iface | Подсеть | Dial |
|------|-------|---------|------|
| SP | nd-svc-sp | 10.87.10.0/30 | secondary → primary :8444 |
| PS | nd-svc-ps | 10.87.11.0/30 | primary → secondary :8445 |

Table=off; AllowedIPs peer /32. Health timer → `svc-paths-health.json`.  
Users — VLESS; agent — SP; recovery — PS.  
API :8789 только service CIDR + api-allow; arm через TG.

Failover: policy primary → secondary heartbeat; Users→SP при падении public :443 (opt-in/флаг).

**После recover:** primary conf = secrets + users (relay-uplink); иначе uplink ломается (инцидент 2026-09-27).
