**RU** · [EN](../REVIEW-REPORT-2026-10-19.md)

# Отчёт полного комплекса ревью — tip v0.9.427 (2026-10-20)

Выполнено по [REVIEW-SUITE.md](REVIEW-SUITE.md). План работ и статус кусков: [REVIEW-PLAN-2026-10-19.md](REVIEW-PLAN-2026-10-19.md).
Охват: весь репозиторий, `main` = `451a83bd` (v0.9.427), 533 Go-файла, ~80,8 тыс. строк продакшн-кода, 141 файл тестов.

**Ограничения владельца в этом прогоне:** автотесты (§1) **не запускались** по просьбе; код **не менялся** (только документация); доступа к живому флоту не было (§8 не выполнялся). Всё ниже — статический анализ и чтение кода.

## 0. Метод (воспроизводимо)

| Инструмент | Зачем | Примечания |
|--|--|--|
| `gofmt -l`, `go vet ./...`, `go build ./...` | формат / vet / компиляция | Go 1.22.2; зависимости из зеркал GitHub (`golang.org/x/*` → `github.com/golang/*`, `gopkg.in/yaml.v3` → `go-yaml/yaml`) в временной копии, `go.mod` репозитория не тронут |
| `deadcode` (x/tools v0.28, RTA) | функции, недостижимые из любого `main` | прогон для linux/amd64 и darwin/arm64 (результат идентичен); build-тегов и GOOS-файлов нет, достижимость не зависит от архитектуры |
| скрипт на `python` | неиспользуемые типы/переменные/константы уровня пакета | считает ссылки внутри пакета и `pkg.Name` снаружи; каждое попадание перепроверено `git grep` |
| `scripts/check-*.sh` | статические проверки репозитория | версии, version pins, web UI, паритет UI |
| ручное чтение | auth, токены, recovery TLS, guest-портал, edge store | см. §3 |

Правило мёртвого кода: символ считается **мёртвым**, только если нет ссылок вне тестов, он не достигается через reflection/шаблоны (в репозитории их нет), не является методом под интерфейс, и `deadcode` согласен на обеих ОС.

## 1. Tip / релиз / formula (§2)

- `VERSION` = `internal/version.Release` = 0.9.427 ✅; version pins ✅.
- GitHub: последний опубликованный релиз — **v0.9.426**; **v0.9.427 ещё не выпущен** (tip впереди релиза, как и записано в handoff). `Formula/netductor-op.rb` = 0.9.426 с настоящим SHA256 без `:no_check` ✅; есть только `netductor-op.rb` ✅.

## 2. Тесты (§1)

Не запускались (просьба владельца). Статические факты: `go build ./...` проходит; все четыре бинаря и кросс-сборка агента (arm, mips, mipsle, 386) компилируются. Таблица долга §1.3: пакетов уровня P0/P1 без тестов не осталось. Инварианты TG §1.4 — это тесты, они не запускались.

## 3. Находки

Серьёзность: Critical · High · Medium · Low · Info. Колонка «Предложение» — что сделать (в этом прогоне код не менялся).

### 3.1 Безопасность (§10)

| ID | Сер. | Находка | Предложение |
|--|--|--|--|
| S-1 | **Medium** | `vpn.RevokeSubToken` нигде не вызывается. Удаление VPN-пользователя (`RevokeNative`) оставляет строку в `vpn_sub_tokens.json`; **если позже создать пользователя с тем же именем, старая subscription-ссылка снова становится рабочей** у всех, у кого она сохранилась. Кроме того, `handleSubscription` не проверяет `Enabled`, поэтому у *отключённого* пользователя тело подписки по-прежнему отдаётся (UUID в sing-box уже не работает, но список серверов утекает). | Вызывать `RevokeSubToken(name)` в `RevokeNative`; отклонять отключённых в `SubscriptionBody`; добавить тест |
| S-2 | **Medium** | HTTP-клиент recovery (`install/backup.go`, `recoveryHTTPClient`) откатывается на `InsecureSkipVerify`, когда нет CA-пина **или файл пина есть, но не является валидным PEM** (тихий даунгрейд). По каналу идут зашифрованные бэкапы и файл с ключом. | Fail closed при нечитаемом пине; для первого контакта — явный флаг или TOFU с записью отпечатка |
| S-3 | Low | На secondary `BACKUP_KEY.txt` лежит рядом с архивами `.ndenc` в `peers/core/` — шифрование «в покое» на этом хосте фактически обнуляется. | Зафиксировать как принятый риск или разделить хранение ключа (только оператор) |
| S-4 | Low | Несравнение за постоянное время для секретов: guest `Code`/`Token`, `ResolveSubToken` (линейный перебор через `==`), код reboot challenge. Смягчено TTL, one-shot и rate limit на `/sub/`. Edge-токены уже используют `constEq` (sha256 + `ConstantTimeCompare`). | Вынести `constEq` в общий helper и применять везде |
| S-5 | Low | `http.ListenAndServe*` без таймаутов `http.Server` в 6 местах (guest captive portal в недоверенной гостевой Wi-Fi, agent `:8790`, secondary recovery, serve/operator на loopback). Во всём репозитории всего 4 настройки таймаутов → исчерпание медленными клиентами. | Единый конструктор в `internal/httpx` с Read/Write/Idle timeout |
| S-6 | Low | Сборка команд через shell: `echo y \| ufw delete <n>`, где `n` берётся из вывода `ufw`; **приватный ключ WireGuard передаётся в argv** (`bash -c "echo '<priv>' \| wg pubkey"`, `svcpaths/bootstrap.go:44`) и виден в `/proc/<pid>/cmdline`. | argv + stdin (`cmd.Stdin`), без shell |
| S-7 | Low | `curl … get.docker.com \| sh` в `install/lampac.go:71` противоречит правилу цепочки поставок «проверять SHA256» (§10). | Пакет `docker.io` из дистрибутива или закреплённый/проверяемый скрипт |
| S-8 | Low | `edge/store.go`: `_ = saveDevices(m)` в 8 местах; `Approve` возвращает токен, который мог не сохраниться. Тихий отказ на критичном пути (§9, наблюдаемость). | Пробрасывать ошибку |
| S-9 | Info | `InsecureSkipVerify` также в `probes/run.go`, `tapo/client.go`, `tapo/klap.go`, `vpn/sni_health.go` — локальные самоподписанные цели / SNI-проба, так задумано; у двух нет комментария-обоснования `//nolint:gosec`. | Добавить комментарии |
| S-10 | Info | **Проверено, всё в порядке:** глобальный `apiSessionGate` (defence-in-depth для `/api/*`, явный список исключений); `serve` отказывается от не-loopback bind; edge `Deny`/`Revoke` обнуляют токен устройства, поэтому `DeviceIDFromAuth` не сопоставит его; секреты пишутся с `0600`; `math/rand` не используется; reflection и шаблонов нет; найденные записи с `0644` — несекретные конфиги. | — |

### 3.2 Корректность / качество (§9)

| ID | Сер. | Находка | Предложение |
|--|--|--|--|
| Q-1 | Medium | i18n Web UI: `check-web-ui.sh` сообщает о **4 ключах `data-i18n`, отсутствующих в JS-словаре**: `edge_st_g_bak`, `rd_mode`, `rd_pos`, `result_hint` (Result dock, 0.9.422–0.9.426). Нарушает §7 (нужны и `I18N.en`, и `I18N.ru`). | Добавить ключи EN+RU |
| Q-2 | Low | `go vet`: `internal/git/projects.go:108` — самоприсваивание `list[i].CreatedAt = list[i].CreatedAt` (ничего не делает; задуманное поле-источник неизвестно). | Исправить или удалить |
| Q-3 | Low | **129 из 533 Go-файлов не отформатированы gofmt** (в основном `cmd/*`); проверки форматирования нет. | Один механический коммит `gofmt -w` + добавить `gofmt -l` в комплекс |
| Q-4 | Info | `install/backup_verify.go:52` — константа 4 ГиБ переполняет `int` на 32-битных платформах; пакет не входит ни в один 32-битный бинарь (агент кросс-собирается нормально). | Использовать `int64` |
| Q-5 | Info | WARN `check-web-ui.sh`: два `/v1` fetch без `X-Netductor-Token` (`app.js:779`, `:2130`); `/v1/meta` (строка версии), похоже, открыт намеренно, а в `:779` заголовки собираются выше — вероятно ложные срабатывания. | Подтвердить в browser smoke |

### 3.3 Документация (§3, §4)

| ID | Сер. | Находка | Статус |
|--|--|--|--|
| D-1 | Medium | **У 19 EN-документов нет RU-пары**, что противоречит правилу AGENTS.md: CI-WORKFLOWS, CLEANUP, DAY2-GROUPS, DISASTER-RECOVERY, GH-PRIVATE-SETUP, OPENWRT-LAB, PLAN-CI-NEXT, PLAN-EDGE-DEPLOY-HYBRID, PLAN-GIT-CI-OPS, PLAN-INTERNAL-RELEASES, PLAN-NVR-IMPLEMENTATION, PLAN-NVR-UI, PLAN-REMOVE-TG-TOPICS, PLAN-TG-MENU, RECOVER-BOT, RELEASE, REVIEW-PROGRESS, TG-ALERTS-CHANNEL, YOUTUBE-ADS. | Открыто — в OPEN_ITEMS |
| D-2 | Medium | Устаревшие маркеры tip: в `AGENTS.md` baseline v0.9.261 и Project State 0.8.30; в handoff сверху v0.9.414/0.9.421; в OPEN_ITEMS стоп v0.9.422/0.9.414; верх RU-handoff противоречил сам себе; в EN-handoff было несколько склеенных блоков `# AGENT Handoff`. | **Исправлено в этом прогоне** (EN+RU) |
| D-3 | Info | `PLAN-EDGE-DEPLOY-HYBRID.md` утверждает, что `ssh_harden` переиспользует `edge.Harden`, но у `edge.Harden` нет вызывающих (см. DC-A). | Согласовать при удалении мёртвого кода |
| D-4 | Low | Битые относительные ссылки: `ru/OPEN_ITEMS.md` вёл на пять документов без RU-пары (следствие D-1); `AGENT_HANDOFF.md` ссылался на удалённый `TG-MEDIA-TOPICS.md`. | **Исправлено в этом прогоне** |

## 4. Мёртвый код (перепроверен)

### DC-A. Мёртв везде (нет ссылок в продакшне, тестах и как документированный API)

| Пакет | Символы |
|--|--|
| `cmd/netductor` | `withSession` (заменён глобальным `apiSessionGate`) |
| `cmd/netductor-op` | `capture`; тип `menuItem` + методы `Title/Description/FilterValue`; поле модели `list list.Model` (и его импорт `bubbles/list`); стили `docStyle`, `helpStyle`, `chipStyle` |
| `cmd/netductor-tg` | `shareURIFrom`; `var lastAccessMsg`; const `statusSh` |
| `internal/cli18n` | `Register` |
| `internal/deploy` | `scpTo`; `PrepareEdgeOffline` (обёртка — вызывающие используют `PrepareEdgeOfflineOpts`) |
| `internal/git` | `BuildProject` и транзитивно `WaitVpsJob`; `ExtractAndBuild`; `shouldPauseAfterStep` |
| `internal/install` | `PrunePeerBackups` (дубль живой `secondary.prunePeerArchives`); `AllowAgentMTLSFromIP` (в комментарии сказано, что вызывать нельзя); `var unwantedPackages` (вытеснен определением по подстроке — проверить покрытие purge) |
| `internal/logs` | `runOut` (живая — `install.runOut`) |
| `internal/mikrotik` | `Policy` |
| `internal/notify` | `htmlEsc` (у агента своя) |
| `internal/nvr` | `EnsureStorageReady` |
| `internal/opcatalog` | `GroupLabel`, функция `Sections()` (поле `Group.Sections` живое) |
| `internal/operator` | `FleetDeploy` (живая: `FleetDeployWithReport`) |
| `internal/registry` | `Catalog` (живая: `CatalogDetail`) |
| `internal/secondary` | `HardenSSH` (legacy; `mikrotik.HardenSSH` — другая, живая функция) |
| `internal/svcpaths` | `WaitPing` |
| `internal/tapo` | `Client.GetDeviceInfo` (alias) |
| `internal/upstream` | `Save` (нет вызывающих; pins сейчас только читаются) |
| `internal/edge` | `Harden` |
| `internal/vpn` | `RevokeSubToken` — **не безобидный мёртвый код, см. S-1** |
| константы / переменные | `devices.reFrom`; `edgestorage.ExpectUSBFlag`, `FactUSBPresent`, `FactExtrootActive`, `FactExpectUSB` (агент/UI используют строковые литералы — дублирование); `fleet.LabelLampac`, `LabelBot`; `guest.TokenTTL`; `ndconfig.DefSvcSPPrimary/SPSecondary/PSPrimary/PSSecondary` (дубли `svcpaths.AddrP_*`); `servicenet.LegacyCIDR`, `Prefix`; `sites.Location` (alias) |

### DC-B. Живы только потому, что их вызывают тесты (в продакшне не используются)

`tg.formatVPNList`, `deploy.LoadCachedTemplate`, `dnsblock.DefaultBlockyDenyURLs`, `edge.DeleteTemplate`, **`edge.ValidBearer`** (тесты проверяют более строгое условие, чем продакшн-путь `DeviceIDFromAuth`, то есть дают ложную уверенность — см. R-5), `edgeagent.ShellApply`, `edgeagent.ParseUCIBatch`, `edgeagent.VLESSClientConfig`, `git.ReleaseVpsSlot`, `metrics.LatestProbes`, `nvr.LeaseRemainingSec`, `opcatalog.ByID`, `opcatalog.Destructive`, `vpn.ParseMismatchCount`; а также тестовый helper в продакшн-файле: `edge.resetUSBPrevForTest`.

### DC-C. Проверено и **не** мертво (ложные срабатывания наивных инструментов)

`FleetDeployWithReport`, `PrepareEdgeOfflineOpts`, `CatalogDetail`, `mikrotik.HardenSSH`, `install.runOut`, `addons.Register`, `jobs.Store.Register`, `mtls.Dir`, `htmlEsc` агента. Достижимость одинакова на linux/amd64 и darwin/arm64.

Удаление DC-A убирает примерно 45 функций и ~25 констант/переменных; для DC-B решать по каждому пункту: удалить вместе с тестом либо оставить и задокументировать как публичный API.

## 5. Рефакторинг (§11)

| ID | Приоритет | Пункт | Почему |
|--|--|--|--|
| R-1 | High | Функции-монстры: `registerEdgeAPI` 937 строк, `handleCallback` (TG) 779, `handleGitCB` 666, `runDoctorNative` 637, `registerSessionAPI` 597, `registerGitAPI` 567, `handleMessage` 540, `registerNVRAPI` 533, `runDeploy` 480, `DeployEdge` 385 | Не тестируются, источник конфликтов слияния; разбить на обработчик на маршрут/callback и табличную диспетчеризацию |
| R-2 | Medium | Дубли helper-ов: `htmlEsc`×2, `shellQuote`×2, `runOut`×2, `randomToken`×3, `truncate`×4, `esc`×2 | Вынести в небольшой внутренний пакет (`internal/textutil`, `internal/sysexec`) |
| R-3 | Medium | 32 места с `bash -c`/`sh -c`, много `_ = exec.Command(...).Run()` | Один helper `sysexec`: только argv, таймаут через context, логирование ошибок |
| R-4 | Medium | `internal/httpx` есть, но серверы создаются разрозненно (S-5) | Один защищённый `NewServer` |
| R-5 | Medium | В edge-auth три почти одинаковые функции (`ValidBearer`, `DeviceIDFromAuth`, `RequireApproved`) | Одна функция, возвращающая `(deviceID, approved)`; тесты направить на продакшн-путь |
| R-6 | Low | Константы продублированы между пакетами (последняя строка DC-A) | Единый источник в `ndconfig` |
| R-7 | Low | `internal/operator/serve.go` — 1208 строк, смешаны маршрутизация, jobs, deploy | Разделить по ответственности |
| R-8 | Low | Тестовый helper в продакшн-файле; legacy-обёртки ради тестов (DC-B) | `export_test.go`; удалить обёртки |

Ранее закрытые R* из `REVIEW-PROGRESS.md` не переоткрывались.

## 6. Слои / паритет UI (§5–§7)

`check-ui-parity.sh`: OK для doctor, vpn, edge, secondary, mtls, nodes, sites, backup, nvr, deploy, fleet, ssh-hosts. TG: эмитируется 299 различных callback `m:*`, **без обработчика — 0** (эвристика; настоящая защита — `TestNoObviousOrphanMenuCallbacks`). i18n Web: см. Q-1.

## 7. Шаблон ответа оператору (§12)

1. Tip: локально 0.9.427, опубликован релиз 0.9.426. 2. Тесты: не запускались (владелец). 3. Доки: handoff / OPEN_ITEMS / AGENTS обновлены EN+RU; D-1 открыт. 4. Паритет UI: OK, Q-1. 5. TG: сирот нет. 6. Ревью кода: Q-1…Q-5. 7. Безопасность: S-1…S-10. 8. Рефакторинг: R-1…R-8 (не «рефакторинг не нужен»). 9. Кандидаты fix-now (нужно согласие владельца, все небольшие): S-1, Q-1, Q-2, коммит gofmt, удаление DC-A. 10. Команд apply нет (tip не двигался).
