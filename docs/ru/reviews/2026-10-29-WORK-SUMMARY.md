**RU** · [EN](../../reviews/2026-10-29-WORK-SUMMARY.md)

# Сводка работ для ревью — 2026-10-28 … 2026-10-29

> Единый handoff для code review. Без секретов.  
> Репозиторий: `PavelNeyman/netductor`.

## 1. Хронология

| Тема | Итог |
|------|------|
| Prod P/S на **v0.9.596** | Update OK; plane OK; VPN через A |
| UFW на Secondary | Восстановлен (`firewall apply secondary`) |
| Peer registry | Только A, **slot 0**, active |
| Ложные edge mTLS (Cudy) | Revoke + удаление каталогов; secondary cert сохранён |
| PR #1 план update-system | В `main` |
| **Phase 0** audit | commit `d00c78c` |
| **Phase 1** contract | commit `357a09d` |
| Сводка + **Phase 2** primitives | текущий commit |

**B (второй secondary)** и полная раскатка **Edge** отложены: приоритет — unified update system.

## 2. Цепочка коммитов для ревьюера (от старых к новым)

```text
e79e87b  Merge PR #1: план update-system, policy/templates EN/RU
d00c78c  docs: Phase 0 baseline audit EN/RU
357a09d  feat(update): Phase 1 contract + tests
c98ccf3  docs+feat(update): work summary; Phase 2 resolve + job registry
(next)   бесшовный DownloadReleaseAsset → ResolveReleaseAsset; миграция в этом файле
```

```bash
git log --oneline e79e87b^..HEAD
go test ./internal/update/ -count=1 -run 'Contract|AssetName|Resolve|JobRegistry|Legacy'
```

**Прод, на котором уже сидит оператор:** релиз **v0.9.596**. Коммиты после тега на `main` — docs + contract/engine; должны оставаться **безопасными для наката** без обязательной переписи конфигов (см. §9).


## 3. Файлы

См. EN §3 (те же пути): план, аудиты, `UPDATE-CONTRACT`, `internal/update/contract.go`, release scripts, этот review.

## 4. Исходы Phase 0–1

Карта entry points; ID `node|tg|agent|op`; state machine; `go test ./internal/update/`.

## 5. Phase 2 в этом изменении

`ResolveReleaseAsset`, registry job «один на instance», фиксация trust boundary SHA256SUMS. Полная матрица отказов на всех live-path — follow-up.

## 6. Как ревьюить

```bash
git log --oneline e79e87b^..HEAD
go test ./internal/update/ -count=1
```

Читать план, контракт, Phase 0 audit, эту сводку.

## 9. Бесшовное обновление с v0.9.596

Прод уже на **v0.9.596**. Требования миграции ведутся **здесь** (план UPDATE-SYSTEM не переписываем).

| ID | Требование | Статус |
|----|------------|--------|
| **M1** | CLI `netductor update …` с 0.9.596 продолжает работать | **Сделано (compat)** |
| **M2** | Обязательная миграция конфигов Primary/Secondary | **Не требуется** (только docs/contract/resolve) |
| **M3** | TG/Web Updates не ломаются | **Сделано (без ломки)** |
| **M4** | Автопереход на новый engine «под капотом» | **В работе** (Resolve на download path) |
| **M5** | Откат на assets v0.9.596 / snapshot | **Описано** |
| **M6** | Update node не трогает чужой edge agent | **Контракт** |

Полная замена UI каталога — этапы 5–6, без flag day.
