

> **Provision order (0.9.176+):** agent + SSH key → stage network/guest UCI (**no** `network reload`) → single reboot. LAN/Wi‑Fi/WAN from the form apply after reboot so SSH is not dropped mid-run.
## Маршрут: private IP → всегда `direct`

В клиенте sing-box на edge одно из первых правил:

`ip_is_private: true` → outbound **`direct`**

**Что считается private (типично):**
- RFC1918: `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`
- link-local и блоки, которые sing-box относит к private
- Трафик к хостам в **LAN самого роутера** (телефоны, ПК, камеры Tapo, принтеры, LuCI на `192.168.x.1`)

**Зачем:**
1. **LAN должен жить без VPN.** RTSP камер, SMB, SSH на роутер, LuCI, цели guest-изоляции, DHCP — это не должно уезжать в VLESS.
2. **Без hairpin через secondary.** Гнать `192.168.50.20` на RU VPS и обратно бессмысленно и ломает локальные сервисы.
3. **Безопасность soft fallback.** Даже когда дефолтный путь — VLESS, локальный трафик **не зависит** от доступности secondary.

**Что по-прежнему идёт в VLESS (когда proxy жив):**
- Публичные назначения из клиентов **private Wi‑Fi / LAN**, у которых шлюз по умолчанию — роутер (на роутере TUN с `auto_route`).

**Guest Wi‑Fi:** отдельная firewall-зона; политика — **только ISP**, не «затолкнуть всё в TUN». Правило private→direct на стеке роутера всё равно действует.

**Адреса primary / secondary:** тоже **direct**, чтобы mTLS агента на primary и набор VLESS на secondary не зацикливались в туннель.

---

## Шаблон `dns: vpn` (по умолчанию)

Поле шаблона (дефолт для новых):

```json
"vpn": { "enabled": true, "mode": "tun", "fallback": "wan", "dns": "vpn" }
```

**Смысл `dns: vpn`:**
1. Правило маршрута **`protocol: dns` → `hijack-dns`**: DNS, который видит sing-box, обрабатывается его DNS-модулем (не «сырой» уход на случайный upstream только на WAN).
2. DNS-серверы в клиентском конфиге:
   - **`ya`**: `77.88.8.8` — для суффиксов `.ru` / `.рф` / `.su` (**без** detour через VLESS).
   - **`remote` / `remote2`**: `9.9.9.9` / `1.1.1.1` с **`detour: auto|proxy`** — остальное резолвится **по пути VPN** (secondary). Это режим «DNS LAN идёт тем же путём, что и веб»; blocky на primary остаётся на control plane — edge **не** ходит на `127.0.0.1:53` primary.
3. **`dns: wan`**: без упора на hijack + remote detour (резолвер ISP/WAN).
4. **`dns: off`**: без отдельного DNS-блока в JSON клиента (дефолты dnsmasq OpenWrt).

**Клиенты LAN:** по-прежнему DNS = **роутер** (dnsmasq). Upstream dnsmasq следует маршрутизации роутера: при TUN + hijack запросы попадают в политику sing-box выше.

**Это не то же самое, что «каждый ответ от blocky на primary».** У edge нет прямого localhost-пути к blocky primary; фильтрация на primary действует на трафик, который **выходит** через secondary→uplink primary, когда этот путь используется. Отдельные edge-списки blocky можно добавить позже при необходимости.


## LuCI (временный admin UI)

- Только start/stop **uhttpd** (пакет остаётся).
- TTL по умолчанию **1ч**; продление 4/24/72ч.
- **SSH с Mac** (та же LAN, интернет на роутере не нужен) или **очередь агента** на primary.
- Автовыключение по TTL (цикл агента).
- TG: Роутеры → LuCI; Web/TUI/CLI `edge-luci`.

**RU** · [EN](../EDGE-AGENT.md)

# Edge agent (OpenWrt)

## Назначение
Агент на OpenWrt (Cudy и др.): enroll к primary, применение шаблонов, heartbeat, guest Wi‑Fi, NVR helpers, remote commands.

## Жизненный цикл
1. Install с Mac (SSH на чистый роутер) — бинарь agent
2. Agent пишет local config, поднимает сервисы сайта
3. Enroll на `https://PRIMARY:8789` (mTLS после выдачи cert)
4. Пока нет интернета — backoff enroll (не ограничение «5 минут навсегда»)
5. Pending на primary → оператор approve/reject
6. После approve — pull template/config_ver, применение

## Идентичность
Device id, token, client cert. Re-bind: recovery code / страница в LAN без полного re-flash.

## Шаблоны
Серверные templates (не backup другого железа). Agent не должен затирать уникальный site state вслепую.

## Guest Wi‑Fi
Отдельный SSID (не band-split по умолчанию); isolation; WAN direct; grant PIN+QR; TTL (default 10m, max 24h).

## NVR / камеры
Leases list, static DHCP, onboard Tapo — [PLAN-NVR-TAPO](PLAN-NVR-TAPO.md).

## Обновления
UI показывает доступную версию agent; update per-device или everywhere; self-replace binary + restart unit.

## Безопасность
mTLS; rate-limit enroll; revoke/rotate certs — ops. Нет требования VPN для enroll (agent за NAT).

## Команды / API
Полные пути HTTP и поля JSON — в EN; CLI `netductor edge …` на primary.

## Архитектура агента (first-boot)

**Репы OpenWrt / ipk нет.** First-boot — pure-Go binary через SCP с Mac.

1. SSH на роутер.
2. Probe: `uname -m`, openwrt_release, opkg/apk.
3. Карта asset’ов: arm64, arm, amd64, **mipsle** (Cudy TR1200), riscv64.
4. Скачать asset → provision.
5. Поле arch по умолчанию **`auto`**; override только при ошибке probe.

Day-2: stack / agent_update вручную, не opkg.



## Где задаётся `vpn.dns` / soft fallback

| Уровень | Где |
|---------|-----|
| **Дефолтный seed** | Код `edge.EnsureDefaultTemplate()` → файл **`/var/lib/netductor/edge/templates/default.json`** (на primary) |
| **На устройство** | `template_id` + overlay в store устройств; отдача `GET /api/edge/template?device_id=` через `TemplateWithVPN` |
| **На роутере** | Агент пишет `/etc/netductor-agent/sing-box-client.json` при `apply_template` |
| **CLI** | `netductor edge template-get [id]` · `netductor edge template-set-vpn [id] dns=vpn mode=tun fallback=wan` |
| **API** | `POST /api/edge/templates` полным JSON (session); bind — bind-template |

| **Web Control** | Edge → **Template VPN / DNS** (Load / Save) → `GET|POST /api/edge/templates/vpn` |
| **Telegram** | Роутеры → **Шаблон VPN/DNS** → кнопки `dns:` / `mode:` / `fb:` / `soft:` |
| **Web Day-2** | Get/bind шаблонов (отдельной формы полей `dns` пока нет — CLI/API или правка JSON) |
| **TG** | Роутеры → Шаблоны / Bind / Apply; редактора поля `dns` пока нет |

Если в шаблоне нет `vpn.dns`, при `TemplateWithVPN` всё равно подставляется **`dns=vpn`**.

