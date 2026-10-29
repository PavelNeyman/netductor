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

## 2. Коммиты (docs / update)

| SHA | Суть |
|-----|------|
| `e79e87b` | Merge PR #1 |
| `d00c78c` | Phase 0 audit EN/RU |
| `357a09d` | Phase 1 contract + tests |
| *(этот)* | Work summary + Phase 2 |

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
