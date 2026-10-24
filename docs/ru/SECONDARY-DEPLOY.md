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
6. Bootstrap SP/PS: SCP material на secondary `/tmp/nd-svc-material.json` с **`trap EXIT` rm** и правами **600**.
7. Ping только peer (S→`10.87.10.1`/`10.87.11.1`, P→`10.87.10.2`/`10.87.11.2`); units active; **health timer** active.
8. Ожидание в registry: именно этот `AgentID` **online** и `seen=` не старше **90 с** (свежий heartbeat после URL на SP).
9. `preferred_secondary_ip` пишется только если IP в registry — **IPv4** для этого AgentID.

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
| 7 | По желанию | Клиентский `/sub/<token>` после preferred IP / DNS |

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
