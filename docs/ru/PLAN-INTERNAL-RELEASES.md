# Internal releases (план / статус)

Сборка и зеркалирование релизов netductor на primary (не только GitHub как единственный источник).

## Цели

- `release local` / mirror-fetch / build на VPS
- SHA256SUMS и совместимость со `stack apply`
- Опционально auto-build при mirror-fetch; metrics `local_release_*`

## CLI / API (ориентир)

```bash
# статусы и теги
# API: /api/release/local, git-tags, mirror-fetch, build, build-status, prune
```

## Ограничения

- Darwin op-бинарники — с Mac или skip_darwin
- Private GitHub — нужен `github_token`
- Не путать с `git project build` (продукты) vs internal netductor release build

## Документы рядом

`RELEASE.md`, `CI-WORKFLOWS.md`, handoff stop-line = tip после выкладки.
