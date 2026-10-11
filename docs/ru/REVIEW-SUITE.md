# Комплекс ревью netductor (обязательный)

**Назначение:** единый канонический чеклист «полного комплекса» после волн фич, security-проходов или перед apply на прод.  
**Не** составляйте укороченный список в чате — следуйте этому документу (или [EN](../REVIEW-SUITE.md)).  
**Язык:** любое устойчивое изменение документации выходит **EN + RU** с **одинаковым смыслом** (не однострочный stub «см. EN»). Короткие `POST-REVIEW-*.md` — только указатели на релиз, не замена комплекса.

**Связанные:** [AGENT_HANDOFF.md](../AGENT_HANDOFF.md) · [OPEN_ITEMS.md](../OPEN_ITEMS.md) · [OPCATALOG.md](../OPCATALOG.md) · [I18N.md](../I18N.md) · [PLAN-TG-MENU.md](../PLAN-TG-MENU.md) · [AGENTS.md](../../AGENTS.md)

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
| `internal/firewall` | P1 | role/ufw helpers |
| `internal/channels` | P1 | face/uplink |
| `internal/logs` | P2 | schedule |
| `internal/servicenet` | P2 | VIP |
| `internal/svcpaths` | P2 | health file |
| `internal/devices` | P2 | registry |
| `internal/incident` | P2 | pack |
| `internal/hardening` | P1 | path checks |
| `internal/dnsblock` | P2 | URL allow |
| `internal/cli18n` | P3 | ключи EN/RU |
| `internal/version` | P3 | Release |
| `cmd/netductor-op` | P2 | catalog smoke |

**Обязательство:** каждый полный прогон либо (a) добавляет тест по открытому P0/P1 в затронутых пакетах, либо (b) явно пишет, что для diff P0/P1 долга нет.

---

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

## 9. Фокус code review (свежий diff)

| Фокус | Критерий |
|--|--|
| Trust boundaries | session, mTLS, agent cmd allowlist |
| Download / update | SHA обязателен, если нет явного env footgun |
| Path containment | clip/git/NVR под корнями |
| Secrets | 0600; токены не в handoff |
| Ошибки | не глотать security-check |

---

## 10. Шаблон ответа оператору

1. **Tip / release**  
2. **Тесты** — результат; что добавили  
3. **Handoff / OPEN_ITEMS / доки EN+RU**  
4. **Паритет UI** (дельты)  
5. **TG / i18n**  
6. **Находки** — severity + сейчас / отложить  
7. **Команды apply**, если tip сдвинут  

---

## История документа

| Версия | Изменение |
|--|--|
| 0.9.329 | Первый полный комплекс EN+RU; тесты integrity + reboot; ссылка из handoff/AGENTS |

