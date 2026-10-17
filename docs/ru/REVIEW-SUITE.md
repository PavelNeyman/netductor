# Комплекс ревью netductor (обязательный)

**Назначение:** единый канонический чеклист «полного комплекса» после волн фич, security-проходов или перед apply на прод.  
**Не** составляйте укороченный список в чате — следуйте этому документу (или [EN](../REVIEW-SUITE.md)).  
**Язык:** любое устойчивое изменение документации выходит **EN + RU** с **одинаковым смыслом** (не однострочный stub «см. EN»). Короткие `POST-REVIEW-*.md` — только указатели на релиз, не замена комплекса.

**Связанные:** [AGENT_HANDOFF.md](../AGENT_HANDOFF.md) · [OPEN_ITEMS.md](../OPEN_ITEMS.md) · [OPCATALOG.md](../OPCATALOG.md) · [I18N.md](../I18N.md) · [PLAN-TG-MENU.md](../PLAN-TG-MENU.md) · [AGENTS.md](../../AGENTS.md)
Журнал чисток (после 0.9.403): [CLEANUP-PASSES-2026-10.md](CLEANUP-PASSES-2026-10.md).

---

## Когда гонять

| Триггер | Объём |
|--|--|
| После волны фич (security, UI, DNS, SSH, …) | **Полный** (все разделы) |
| Одна мелкая правка | **Мини**: §1 тесты + §3 stop line + затронутый UI |
| Перед `stack apply` чужого tip | **Полный** минимум §1–§5 + §8 |
| Фраза «весь комплекс» / «полный ревью» | **Полный**, §7 (долг тестов) не пропускать |

Stop line и дату фиксируйте в коммите/PR или в `docs/POST-REVIEW-<version>.md` как указатель; **результат** — в ответе оператору; handoff обновлять при смене tip.

---

## 1. Автотесты (блокер)

### 1.1 Прогон

```bash
export GOPROXY=https://proxy.golang.org,direct
go test ./internal/... -count=1
go test ./cmd/... -count=1
```

- Перед тегом релиза — **обязательный pass**.
- При 502 прокси — public `GOPROXY`, не отмена suite.

### 1.2 Обязательство по покрытию

| Правило | Суть |
|--|--|
| Новая прод-логика | В том же изменении — тест, если можно без root/VPS |
| Пакет без `*_test.go` | См. §1.3; фича в пакете → тест **или** явный defer в OPEN_ITEMS с причиной |
| Security-пути | challenge, allowlist, SHA download, session, path containment — **тесты предпочтительнее** «только руками» |

### 1.3 Пакеты без тестов (долг)

| Пакет | Приоритет | Заметки |
|--|--|--|
| `internal/integrity` | **P0** | **тесты 0.9.329** |
| `internal/notify` reboot | **P0** | **тесты 0.9.329** |
| `internal/firewall` | P1 | truncate/firstLines/detectRole — **тесты 0.9.330** |
| `internal/channels` | P1 | FormatPathE2E/FormatText — **тесты 0.9.330** |
| `internal/logs` | P2 | schedule — **тесты 0.9.331** |
| `internal/servicenet` | P2 | Enabled/Collect — **тесты 0.9.331** |
| `internal/svcpaths` | P2 | peerOf — **тесты 0.9.331** |
| `internal/devices` | P2 | List — **тесты 0.9.331** |
| `internal/incident` | P2 | Collect — **тесты 0.9.331** |
| `internal/hardening` | P1 | DropInConf/RemoteHardenScript/SSHPort — **тесты 0.9.330** |
| `internal/dnsblock` | P2 | catalog — **тесты 0.9.331** |
| `internal/cli18n` | P3 | T/Lang — **тесты 0.9.331** |
| `internal/version` | P3 | Release |
| `internal/fleet` | P2 | **тесты 0.9.331** |
| `internal/onboard` | P2 | **тесты 0.9.331** |
| `internal/cleanup` | P2 | **тесты 0.9.331** |
| `internal/upstream` | P2 | **тесты 0.9.331** |
| `internal/addons` | P3 | **тесты 0.9.331** |
| `internal/tlsle` | P3 | **тесты 0.9.331** |
| `cmd/netductor-op` | P2 | catalog smoke — **тесты 0.9.330** |

**Обязательство:** каждый полный прогон либо (a) добавляет тест по открытому P0/P1 в затронутых пакетах, либо (b) явно пишет, что для diff P0/P1 долга нет.

---


### 1.4 Инварианты TG UX (блокер — `cmd/netductor-tg`)

Ловят **регрессии чата**, которые API/i18n-ревью не видит. Гонять вместе с §1.1 (`go test ./cmd/...`).

| Инвариант | Зачем | Тест |
|--|--|--|
| `/menu` и `/start` → **`forceHub`** (не `reply` по сохранённому id) | После очистки истории edit бьёт в **невидимое** сообщение | `TestMenuCommandsUseForceHub` |
| `wait_*` → **`completeInput`** (не `sendHTML`) | Иначе prompt + текст пользователя + новое меню | `TestWaitHandlersPreferCompleteInput` |
| Users: **`m:vpn_devices`** + `vpn_devices_btn` | kind=device ≠ локатор IP (`m:devices`) | `TestUsersListWiresVpnDevicesCallback` |
| Критичные callback обрабатываются | Мёртвые кнопки | `TestNoObviousOrphanMenuCallbacks` |

**Ручной smoke:** один раз создать VPN user из TG на мажорном TG-релизе; `/menu` после clear chat.


## 2. Tip, релизы, formula

| Проверка | Критерий |
|--|--|
| `VERSION` == `internal/version.Release` | точное совпадение |
| GitHub release `v$VERSION` | node, tg, agent (multi-arch), op darwin/linux, `SHA256SUMS` |
| Публичная выдача | HTTP 200 на `netductor-linux-amd64` |
| `Formula/netductor-op.rb` | версия + реальные SHA256; **нет** `Formula/netductor.rb` |
| Текст релиза | user-visible изменения |

---

## 3. Handoff и планы

| Проверка | Критерий |
|--|--|
| Stop line EN | **= tip** |
| Stop line RU | **= tip**, смысл как в EN |
| OPEN_ITEMS (+ RU) | stop актуален; закрытое отмечено |
| Нет конфликта stop в шапке | hist только глубже по файлу |
| PLAN-* | чекбоксы фаз в **том же** изменении ([AGENTS.md](../../AGENTS.md)) |

---

## 4. Качество документации (EN ↔ RU)

| Проверка | Критерий |
|--|--|
| Новый устойчивый док | EN в `docs/` **и** RU в `docs/ru/` |
| Паритет смысла | Те же разделы, обязательства, команды; не stub на 5 строк |
| Этот комплекс | `REVIEW-SUITE.md` и `ru/REVIEW-SUITE.md` правятся вместе |
| Индекс | Ссылки в `docs/README.md` / `docs/ru/README.md` |

---

## 5. Паритет UI (API — источник истины)

Поверхности: **API** · **TG** · **Web** · **op TUI / opcatalog** · **CLI**.

| Класс | Ожидание |
|--|--|
| Day-2 в opcatalog | Строка в `internal/opcatalog` **до** UI-only кода |
| Деструктивное | Challenge или явный confirm; session на API |
| Status/metrics | Не raw dump, где договорились о таблицах |
| Security-фичи | TG + API минимум; Web/opcatalog в той же волне или OPEN_ITEMS |

### 5.1 Треки security/DNS (ориентир)

| Фича | API | TG | Web | opcatalog |
|--|--|--|--|--|
| SSH allowlist | ✅ | ✅ `m:sshkeys:` | ✅ | ✅ |
| Reboot challenge | ✅ | ✅ | ✅ | ✅ |
| Integrity / doctor | collect/doctor | alerts | — | — |
| Canary | ✅ | ✅ | catalog | ✅ |
| Channel tables | status | ✅ | metrics | — |

---

## 6. Карточки Telegram и меню

| Проверка | Критерий |
|--|--|
| Хабы | Status / Users / Fleet / Tools / Operator ([PLAN-TG-MENU](../PLAN-TG-MENU.md)) |
| Нет клона Status в Tools | Tools ≠ второй Status |
| Callbacks | Без двусмысленных префиксов (`m:sshkeys:` vs `m:ssh:rm:`) |
| Формат | Таблицы/списки; не длинный JSON в user status |
| Дубли | Одно действие не в двух хабах без причины |

---


| Инварианты TG UX (§1.4) | **обязаны** пройти на том же tip |
| Singleton hub | Callbacks **edit**; `/menu` **forceHub**; ввод текста **completeInput** |
| Профили устройств vs IP | `m:vpn_devices` ≠ `m:devices` — обе подписи в Users |

## 7. Двуязычность UI (EN/RU)

| Проверка | Критерий |
|--|--|
| TG | `T()` / i18n; без EN-only кнопок на новых flow |
| Web | ключи в `I18N.en` и `I18N.ru` |
| opcatalog | `label_en` + `label_ru` |
| Recovery-алерты | EN и RU |
| Доки | §4 |

---

## 8. Security / product smoke (флот)

Не замена §1. Гонять при касании VPN, agent, firewall, SSH.

| Проверка | Заметки |
|--|--|
| `netductor doctor` primary | fail=0 или объяснённый WARN |
| `netductor host-audit` | CLEAN или известный residual |
| `netductor channel status` | согласован с реальностью |
| Secondary agent | версия = tip после upgrade |
| SSH allowlist | inventory без enforce, пока не решите |
| Client egress | если волна трогала policy |

---

## 9. Глубокое code review (diff + затронутые пакеты)

Обязательно на **полном** комплексе. Объём: коммиты с прошлой stop line (или явный range).

| Проверка | Критерий |
|--|--|
| Корректность | Логика = продуктовые правила; edge cases обработаны |
| Контракты API | JSON стабилен или версионирован; единый стиль ошибок |
| Concurrency | Нет гонок на shared map/file без lock |
| Идемпотентность | apply/firewall/allowlist/baseline безопасно повторять |
| Наблюдаемость | Сбои в doctor/alerts/logs; не глотать security-критичное |
| Зависимости | Без лишних тяжёлых deps; `go.mod` tidy |
| Тесты | Новая логика по §1.2 |

**Как:** читать изменённые файлы целиком в пакетах; не считать «просмотрено» по одному commit message.

---

## 10. Security review

| Проверка | Критерий |
|--|--|
| Authn/Authz | Session на mutating API; TG только operator; нет голых destructive routes |
| Agent plane | Только allowlist cmd; mTLS :8789 |
| Trust boundaries | Доверие secondary на :9443 задокументировано |
| Secrets | 0600; токены не в docs/git; reboot challenge one-shot + TTL |
| Supply chain | SHA256 на download; ограниченный tag; skip-verify = footgun |
| Injection | Нет shell из user input без allowlist; path containment |
| SSH | Detect vs enforce; enforce предупреждает про ключи хостера |
| Экспозиция сети | API :8787 localhost; plain agent не на WAN; host-audit CLEAN |

**Severity:** Critical · High · Medium · Low · Info.

---

## 11. Необходимость рефакторинга

| Проверка | Критерий |
|--|--|
| Дубли | Одна логика в TG/Web/CLI без helper → R-item или мелкий fix |
| God-функции | Новые огромные функции / смешение concern → backlog |
| Мёртвый код | Удалить или ticket |
| Именование | Не возвращать двусмысленные callback-префиксы |
| Слои | UI не тащит policy — только `internal/` / API |
| Старые R-items | `REVIEW-PROGRESS` / OPEN_ITEMS; не открывать закрытое без новых фактов |

**Правило:** полный комплекс **обязан** либо перечислить follow-up рефакторинга, либо явно написать **«рефакторинг не требуется»** для diff. Фразы «вроде ок» без этого — неполный отчёт.

---

## 12. Шаблон ответа оператору

1. **Tip / release**
2. **Тесты**
3. **Handoff / OPEN_ITEMS / доки EN+RU**
4. **Паритет UI**
5. **TG / i18n**
6. **Code review** (§9)
7. **Security review** (§10)
8. **Рефакторинг** (§11)
9. **Находки**
10. **Команды apply**

---

## История документа

- **0.9.344+:** §1.4 инварианты TG UX + тесты `cmd/netductor-tg`.

| Версия | Изменение |
|--|--|
| 0.9.331 | §9–§11: code / security / рефакторинг; unit-тесты оставшихся пакетов |
| 0.9.330 | P1/P2: firewall, channels, hardening, op catalog |
| 0.9.329 | Первый полный комплекс EN+RU; integrity + reboot |



## Web UI + версии (обязательно)

Перед релизом с `netductor-op` или `internal/operator/web/`:

```bash
./scripts/check-versions.sh
./scripts/check-web-ui.sh
```

Browser smoke: [ru/SMOKE-WEB-OP.md](SMOKE-WEB-OP.md).

**Правило:** дифф web без static + пометки smoke = **неполный suite**.

**Класс ошибок:** `var version = "x.y.z"` в `cmd/*/main.go` — только `ndver.Release`.

## Статические проверки Web UI

```bash
./scripts/check-web-ui.sh
```

Ловит `.hidden !important` vs `style.display`, `window.ND_TOKEN`, базовые ключи i18n.
