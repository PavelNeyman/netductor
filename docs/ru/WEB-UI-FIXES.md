> **IMPLEMENTED in 0.9.93** (TOFU path, labels, CF removed from main form; Lampac checkbox kept).

**RU** · [EN](../WEB-UI-FIXES.md)

# Web UI / Fleet — отложенные правки

Статус: **только зафиксировано**, без реализации, пока оператор не приоритезирует.

## Подписи и плейсхолдеры

| Сейчас | Проблема | Нужно |
|--------|----------|--------|
| **CORE host** / `p2.nd.example.com` | Уже есть Primary host (IP); «CORE» — жаргон; плейсхолдер как будто готовый FQDN | **Primary domain** — «укажите домен хоста primary» |
| **VPN host** | Не привязан к secondary | **Secondary domain** — «домен для secondary» |
| **Domain label** | Без пояснения | Опциональная метка `DOMAIN=` в conf; **не** создаёт p./i.; LE только с явными FQDN |
| **SNI** | Без пояснения | Маскировка Reality (напр. `api.vk.me`) |

То же в TUI и CLI.

## Галочки

| | |
|--|--|
| **Lampac** | Добавить в Web Fleet (как Git/TG) |
| **CF proxy** | Убрать из основного UI или «Advanced»; канон — серое облако + LE на origin |

## Ошибка secondary (разбор)

Симптом: после успешного primary — `open /var/lib/netductor/secondary/ssh_known_hosts.json: no such file`.

Причина: TOFU host key пишется **на Mac** в `/var/lib/netductor/...` (пути ноды), каталог на macOS недоступен → handshake падает. Это **не** отказ secondary VPS.

Временный обход: `sudo mkdir -p /var/lib/netductor/secondary && sudo chown $USER ...` или `NETDUCTOR_STATE=$HOME/.netductor` (после фикса кода).

Постоянный фикс: TOFU у op в `~/.netductor`, не в `/var/lib/netductor`.
