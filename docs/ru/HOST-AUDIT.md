# Аудит хоста: агенты хостера, слушающие порты, эталон

**RU** · [EN](../HOST-AUDIT.md)

Как провести полный аудит VPS netductor (primary или secondary) на **чужой мониторинг / CM / RMM**, лишние listeners и дрейф относительно ожидаемой поверхности. Результат — логи для оператора или ИИ.

Связанный код: `scripts/run-host-audit-bundle.sh`, `scripts/audit-hoster-agents.sh`, `scripts/collect-host-audit.sh`, `netductor host-audit`, `internal/install/host_agents.go`, allow-list файрвола, [PORTS.md](../PORTS.md), [SECURITY.md](../SECURITY.md).

---

## 0. Одноразовая команда для оператора (на VPS)

Запускать **от root** на primary и/или secondary. Только чтение (без purge). Скрипты тянутся с GitHub `main` (можно закрепить tag). Детект apt residual: `*zabbix*`, `*timeweb*` sources/keyrings (0.9.198).

```bash
# Полный bundle → /tmp/nd-host-audit-<host>-<utc>/ + .tar.gz
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/run-host-audit-bundle.sh | bash
```

Закрепить ref:

```bash
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/scripts/run-host-audit-bundle.sh \
  | NETDUCTOR_AUDIT_REF=main bash
```

Снять архив на ноутбук и отдать на разбор:

```bash
scp -P 52222 root@PRIMARY:/tmp/nd-host-audit-*.tar.gz .
scp -P 52222 root@SECONDARY:/tmp/nd-host-audit-*.tar.gz .
```

### Что внутри архива

| Файл | Содержание |
|------|------------|
| `host-audit.txt` | Полный текстовый dump (ss, units, packages, firewall, ssh, cron, …) |
| `hoster-agents.txt` | Проба denylist (zabbix/salt/RMM/…) |
| `extra-signals.txt` | Top CPU, процессы denylist, established TCP, failed units, хвост auth |
| `MANIFEST.txt` | host, ref, время |
| `*.stderr` | Ошибки скриптов, если были |

**Не** класть в bundle приватные ключи и ключи бэкапа (скрипты редактируют / пропускают).

### Без GitHub с VPS

```bash
scp -P 52222 scripts/run-host-audit-bundle.sh scripts/collect-host-audit.sh scripts/audit-hoster-agents.sh root@HOST:/tmp/nd-audit/
ssh -p 52222 root@HOST 'bash /tmp/nd-audit/run-host-audit-bundle.sh'
```

### Деструктивный purge (только после разбора)

```bash
netductor host-audit --purge    # если есть в установленной версии
# или: bash audit-hoster-agents.sh --purge
```

---


## 1. Принципы

| Плоскость | Политика |
|-----------|----------|
| **Входящий** | Default deny; **только белый список** роли. Не строить защиту на бесконечных точечных deny. |
| **Исходящий** | Default allow (нода — выход VPN). Не закрывать исходящий по умолчанию. |
| **Агенты** | На управляемых нодах нет хостерского мониторинга, CM и RMM. |
| **Эталон** | После успешного deploy+harden — снимок listeners/units/packages; дальше diff → WARN/FAIL. |

---

## 2. Ожидаемая поверхность прослушивания

### Secondary

| Порт | Сервис |
|------|--------|
| 52222/tcp | SSH (после harden — только ключ) |
| 443/tcp | sing-box VLESS |
| WSS (напр. 8445) | SP/PS |
| Loopback | agent, health |

### Primary

| Порт | Сервис |
|------|--------|
| 52222 | SSH |
| 443 | Reality / uplink |
| 8443 | redirect (LE) |
| 80 | только ACME при необходимости |
| 8444 | WSS SP (типично) |
| 8787, 8789 | **не** в публичный WAN |
| 9118, 5000, 53, … | loopback / service-net по PORTS.md |

Всё остальное на `0.0.0.0` — finding, пока не обосновано.

---

## 3. Каталог угроз / мусора

- **Мониторинг:** Zabbix, NRPE, Netdata, node_exporter, telegraf, collectd, PCP, Datadog, New Relic, Elastic Agent, Splunk, OTel, CloudWatch, Monit, Glances. Порты: 10050, 10051, 5666, 19999, 9100, 9273, 8125, 2812.  
- **CM:** salt, puppet, chef, landscape. Порты: 4505, 4506, 8140.  
- **RMM:** Mesh, Tactical RMM, AnyDesk, TeamViewer host.  
- **Панель/гипервизор:** агенты панелей; `qemu-guest-agent` / `open-vm-tools` — отдельное продуктовое решение.  
- **Шум дистрибутива:** avahi, cups, rpcbind, лишний MTA.  
- **Компромисс:** лишние ключи SSH, user-systemd, `ld.so.preload`, майнеры, скрытые listeners.

Справочник IoC: [linux-edr-iop](https://github.com/messede-degod/linux-edr-iop).

---

## 4. Пошаговый аудит

На **каждой** VPS от root. Сначала только чтение.

### Шаг 0 — подготовка

```bash
mkdir -p /tmp/nd-audit && cd /tmp/nd-audit
hostname; date -u; cat /etc/netductor/VERSION 2>/dev/null; netductor version 2>/dev/null
```

### Шаг 1 — продуктовый collect

```bash
bash /path/to/netductor/scripts/collect-vps-state.sh | tee nd-state-$(hostname).txt
```

### Шаг 2 — скрипт хостерских агентов

```bash
sudo bash audit-hoster-agents.sh 2>&1 | tee hoster-agents-$(hostname).txt
echo EXIT:$?
```

Код `1` = есть findings. **`--purge` только после разбора лога.**

### Шаг 3 — полный bundle

```bash
sudo bash /path/to/netductor/scripts/collect-host-audit.sh 2>&1 | tee host-audit-$(hostname).txt
```

### Шаг 4 — опционально глубже

```bash
apt-get install -y rkhunter 2>/dev/null || true
rkhunter --check --sk 2>&1 | tee rkhunter-$(hostname).txt
```

### Шаг 5 — архив

```bash
tar -czf nd-audit-$(hostname)-$(date -u +%Y%m%dT%H%MZ).tgz \
  nd-state-*.txt hoster-agents-*.txt host-audit-*.txt rkhunter-*.txt 2>/dev/null
```

Унести с машины (SCP). Скрипты не должны писать приватные ключи в лог.

### Шаг 6 — передача на разбор

К архиву указать: роль (primary/secondary), ожидался ли harden/ufw, был ли вход поддержки хостера.

---

## 5. Как читать findings

| Finding | Действие |
|---------|----------|
| zabbix/telegraf/… active | purge / повтор harden; unit mask |
| порт 10050/9100/… | снять агент; проверить allow-list |
| нет ufw / inactive | поставить ufw или iptables fallback; `firewall apply` |
| :8789 с мира | сузить CIDR; disarm api-public |
| только qemu-ga | политика leave/remove, не авто-critical |
| CLEAN + :631 | выключить cups/avahi |

После purge — шаги 2–3 до `RESULT: CLEAN` и совпадения с матрицей роли.

---

## 6. Скрипты

| Скрипт | Назначение |
|--------|------------|
| `audit-hoster-agents.sh` | denylist; опционально `--purge` |
| `collect-host-audit.sh` | полный текстовый bundle |
| `collect-vps-state.sh` | состояние продукта/stack |
| `netductor host-audit` | denylist из бинаря |
| `netductor firewall status\|apply` | входящий allow-list |
| `netductor doctor` | FAIL при чужих агентах / отсутствии fw |

---

## 7. Эталон после чистого деплоя

```bash
mkdir -p /var/lib/netductor/baseline
ss -tulnp > /var/lib/netductor/baseline/ss.txt
systemctl list-unit-files --state=enabled > /var/lib/netductor/baseline/enabled-units.txt
dpkg -l > /var/lib/netductor/baseline/dpkg.txt
ufw status verbose > /var/lib/netductor/baseline/ufw.txt 2>/dev/null || iptables-save > /var/lib/netductor/baseline/iptables.txt
date -u > /var/lib/netductor/baseline/created
```

Сравнение с эталоном в doctor — в backlog, пока не реализовано. AIDE — по желанию на primary.

---

## 8. Чеклист продукта

- [x] denylist + audit script + CLI (0.9.192+)  
- [x] firewall: ufw + направление fallback  
- [ ] расширить denylist (RMM, otel, avahi/cups) — синхрон script ↔ Go  
- [ ] `collect-host-audit.sh`  
- [ ] doctor FAIL по агентам / fw / лишним портам  
- [ ] статус fw в TG/Web  
- [ ] baseline в конце install  
- [ ] явная политика qemu-ga в SECURITY.md  

---

## 9. Безопасность

- Сначала read-only, потом `--purge`.  
- `cloud-init` по умолчанию не трогать.  
- Авто-heal файрвола с таймера — только алерт; apply по явной команде оператора (риск lockout).