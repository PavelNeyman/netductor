# Operator Web UI — browser smoke (обязательно для диффов `internal/operator/web/`)

Против **локального** `netductor-op operator serve` после пересборки op. ~2 минуты.

## Предусловия

- `netductor-op version` = tip `VERSION`
- Hard-refresh (Cmd+Shift+R) после замены бинаря
- DevTools → Network

## Чеклист

| # | Шаг | Ожидание |
|--|--|--|
| 1 | Installer | Формы Fleet/Primary видны |
| 2 | Вкладка **Control** | Tunnel + подменю (не пустой экран) |
| 3 | **Settings** | Поля host / key видны |
| 4 | Settings: сохранить | OK, после reload поля на месте |
| 5 | OpenWrt → **Probe** | Не голый `Unauthorized`; карточка или текст ошибки SSH |
| 6 | Network `/v1/*` | Заголовок `X-Netductor-Token` |
| 7 | RU/EN | Кнопка Probe и Settings переводятся |

## Автоматика

```bash
./scripts/check-web-ui.sh
./scripts/check-versions.sh
```

## Правило

Дифф web/op UI: в отчёте suite обязательно `Web smoke: done | skipped (причина)`.
