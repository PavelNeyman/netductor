> Полный Blue-Green: [BLUE-GREEN-SECONDARY.md](BLUE-GREEN-SECONDARY.md)

## Blue-Green (v0.9.496+)

```
netductor svc-paths peer add --peer-ip B_IP [--id peer-b]
netductor svc-paths peer validate peer-b  # real checks
netductor svc-paths peer ready peer-b
netductor svc-paths peer promote peer-b
# DNS
netductor svc-paths peer rollback peer-a
netductor svc-paths peer remove peer-b
```

Deploy secondary: Promote=false по умолчанию. Peers `prepared`/`ready` не попадают в клиентские подписки; публикуется только явно promoted peer со `state=active`. Слоты 0–3: `10.87.10/11` … `24/25`.

**EN** · [RU](SECONDARY-DEPLOY.md)

# Deploy Secondary — доверие, конечное состояние, failover

Актуально с **v0.9.490+** (этап A) и **v0.9.491+** (этап B). Путь: оркестрация с Mac; **без SSH primary→secondary**.

Связанные документы: [DEPLOY-MAC.md](DEPLOY-MAC.md) · [SECURITY.md](SECURITY.md) · [MTLS.md](MTLS.md) · [FLEET.md](FLEET.md)

---

## 1. Доверие к SSH host key (R6 / R16)

Два независимых хранилища:

| Хранилище | Кто использует | Где |
|-----------|----------------|-----|
| **Go TOFU** | Первый hop по паролю (`ProvisionFromCore`, Go SSH) | JSON relay known_hosts (`netductor ssh-hosts`) |
| **OpenSSH known_hosts** | Day-2 `ssh`/`scp` через `runSSHOnPort` | `~/.ssh/known_hosts` на Mac оператора |

### Политика (с 0.9.490)

| Ситуация | Поведение |
|----------|-----------|
| Первый контакт (ключа ещё нет) | Принять и сохранить (TOFU / `accept-new`) |
| Ключ совпадает с сохранённым | Продолжить |
| Ключ **изменился** без предупреждения | **Отказ** — без auto-forget и без password retry |
| Оператор подтвердил переустановку / ротацию | CLI forget в обоих store, затем повтор deploy (`force_reinstall` по желанию) |

**Одного пароля недостаточно. Deploy не делает auto-retry при mismatch (v0.9.493+).**

- Go TOFU: `ForgetSSHHost` только через `ForceReinstall` / `force_reinstall` или CLI `netductor ssh-hosts forget --kind relay <host>`.
- OpenSSH: оператор выполняет `ssh-keygen -R` после проверки fingerprint вне канала.

При mismatch сверьте fingerprint; только потом forget и повтор.

```bash
netductor ssh-hosts list
netductor ssh-hosts forget --kind relay SECONDARY_IP
# затем: deploy secondary … (пароль или ключ)
```

---

## 2. Последовательность deploy (кратко)

1. Mac → primary `:52222`: `secondary prepare-pack` (token, mTLS, VPN bundle).
2. JSON pack: `schema_version`, `min_agent_version`, `agent_id`, материал — строгий decode (R12/R15).
3. Mac → secondary `:22` (пароль один раз) или ключ: pubkey, mTLS, `secondary join`.
4. Join отклоняет неподдерживаемую **схему bundle** или agent **старше** `min_agent_version` (R15).
5. Harden → SSH **только ключ** на **52222**; пароль выключен.
6. До SP/PS bootstrap фиксируется deployment IPv4 в `/etc/netductor/secrets/public_ip`; material передаётся в `bootstrap-secondary` через SSH stdin, без временного файла на сервере.
7. Ping только peer (S→`10.87.10.1`/`10.87.11.1`, P→`10.87.10.2`/`10.87.11.2`); units active; **health timer** active.
8. Ожидание именно этого `AgentID`: статус online и `seen=` не старше **120 с**. Окно ожидания heartbeat — до **10 минут** для холодного старта mTLS. Agent использует закреплённый deployment IP, а не результат внешнего IP-check сервиса.
9. Проверяется готовность профиля (`devices.json`: точный deployment IP, PBK, SID, `singbox_ok`). Deploy оставляет peer в `prepared`: не выполняет promote и не публикует B в `/sub`. В подписке доступны только peers со `state=active`; после `peer promote`. До promote обязательна проверка клиентского Reality-трафика и затем DNS cutover.

Smoke ≠ полное доказательство HA — см. §4.

---

## 3. Проверка оператором (day-2)

```bash
export NETDUCTOR_SSH_PORT=52222
KEY=~/.ssh/netductor_primary

ssh -i "$KEY" -p 52222 root@PRIMARY 'netductor secondary status; netductor svc-paths status'
ssh -i "$KEY" -p 52222 root@SECONDARY 'systemctl is-active nd-wss-sp-client nd-wss-ps-server wg-quick@nd-svc-sp wg-quick@nd-svc-ps nd-svc-paths-health.timer; cat /etc/netductor/secrets/secondary_core_url'

ssh -i "$KEY" -p 22 root@SECONDARY true && echo 'FAIL: :22 accepts key' || echo 'ok: :22 not usable with key'
```

Ожидается `secondary_core_url` → `https://10.87.10.1:8789` (SP), не public WAN для agent plane.

---

## Несколько Secondary и service plane

Текущая реализация поддерживает несколько service-plane peers на одном Primary через peer registry и отдельные слоты `SlotTable`.

- Реестр: `/etc/netductor/svc-paths/peers.json`; material каждого peer хранится отдельно в `peers/<peer-id>/material.json`.
- `svc-paths peer add` выделяет свободный слот, генерирует отдельные WG-ключи и применяет конфигурацию только для добавляемого peer.
- Слоты 0–3 имеют отдельные WG-интерфейсы, адреса, порты и WSS-порты. Максимум одновременно доступных peer в текущей таблице — четыре.
- На Primary WSS systemd units для слотов, отличных от legacy slot 0, получают суффикс интерфейса; конфигурация ранее добавленного peer не должна перезаписываться.
- На каждом Secondary устанавливается собственный `material.json` и используются его slot-specific WG-интерфейсы. WSS client/server units на отдельном Secondary имеют локальные стандартные имена — это не конфликтует между разными хостами.
- Первое добавление при наличии legacy `material.json` мигрирует прежний peer в slot 0. Нельзя удалять legacy material или вручную освобождать slot 0 до проверки этой миграции.
- При повторном добавлении того же `peer-id` и IP операция проверяет существующий material и должна быть идемпотентной. Другой IP для уже занятого ID и тот же IP под другим ID отклоняются.
- `--force-new-material` для legacy single-peer workflow не является способом добавить второй peer: это может заменить ключи существующего туннеля. Для blue-green использовать только `svc-paths peer add` и штатный deploy.

Перед новым deploy проверить:
1. `svc-paths peer list` — активные/подготовленные peers и их слоты.
2. `inventory list` — нет ли уже записи для целевого IP.
3. Свободен ли слот в таблице 0–3.
4. Текущая версия operator и Primary поддерживает blue-green workflow.
5. В deploy journal целевого IP нет незавершённой операции либо её можно безопасно продолжить.

**Документация старого single-peer режима ниже этого раздела не должна использоваться как описание текущей blue-green реализации.**
## 4. Семантика failover (v0.9.494+)

| `user_path` | Цель uplink |
|-------------|-------------|
| `vless` | Публичный primary |
| `sp` | `10.87.10.1` (SP) |
| `ps` | `10.87.11.1` (PS) |
| `degraded` | Без перезаписи uplink |

Health VLESS = локальный listener secondary `:443`. URL агента SP/PS отдельно. Циклы под `flock`. Failback — окно `StableSeconds`.

## 4. Чеклист failover (R22) — вручную

Deploy подтверждает units, ICMP peer, файл policy и **active** timer. Реальное переключение — на стенде или в окно обслуживания.

| Шаг | Действие | Ожидание |
|-----|----------|----------|
| 1 | SP и PS живы | Свежий heartbeat; предпочтение SP |
| 2 | Остановить только транспорт SP (не весь secondary) | Переход агента на **PS** по порогам политики |
| 3 | Heartbeat продолжается | `secondary status` online для **этого** id; путь PS |
| 4 | Вернуть SP | Failback по окну политики |
| 5 | Остановить PS при живом SP | Основной путь работает |
| 6 | Оба пути down | Degraded/offline, не «healthy» |
| 7 | До `peer ready/promote` | Проверить реальный клиент напрямую через B с PBK/SID и endpoint B; подтвердить Reality handshake и устойчивый трафик, пока B скрыт из `/sub` |\n| 8 | После promote и DNS cutover | Обновить `/sub/<token>`, переподключить клиент и проверить активный endpoint и реальный трафик |

Фиксируйте время отказа, обнаружения, переключения и возврата. Не выключайте всю машину Secondary для теста «только SP».

Позже: `svc-paths selftest` — только когда ручная матрица однозначна.

---

## 5. Совместимость bundle (R15)

| Поле | Смысл |
|------|--------|
| `version` / `schema_version` | Версия формата (**1** поддерживается) |
| `min_agent_version` | Бинарник secondary ≥ этой SemVer, иначе `join` останавливается **до** смены конфига |

Сначала обновите secondary, если pack требует более новый min agent.

---

## 6. Осознанные ограничения

| Тема | Позиция |
|------|---------|
| DNS A для `vpn.` | Вручную, пока нет единой DNS-интеграции |
| Статические VLESS / QR | Могут устареть при смене Reality; канон — **`/sub/<token>`** |
| Session API `…/subscription` **410** | Только operator API, не клиентская подписка |
| Авто-удаление старого Secondary | **Нет** — явное remove после наблюдения |
| Параллельный deploy + global `NETDUCTOR_SSH_PORT` | Не гонять primary/secondary deploy параллельно в одном процессе (R13) |

---

## 7. CLI

```text
netductor deploy secondary --primary … --primary-key … --host … --password … --sni …
netductor secondary status
netductor secondary prepare-pack
netductor ssh-hosts list|forget|clear
netductor svc-paths status
```
