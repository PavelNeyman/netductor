# Сделать GitHub private и связать с netductor

**EN** · private GitHub + netductor tokens (operator runbook)

Две разные задачи — два разных PAT (уже поддерживается):

| Файл секрета на primary | Назначение | Минимум прав |
|--|--|--|
| `/etc/netductor/secrets/github_token` | Releases / download assets / mirror | `repo` (classic) или Contents+Releases на нужных репах |
| Отдельный token для **private clone** git projects (если иной) | `git project sync` private repos | Contents read |

## Шаги

1. Репозиторий netductor (и apps) → **Private**.
2. Classic PAT или fine-grained с доступом к private assets.
3. На primary:
   ```bash
   install -m 600 /dev/stdin /etc/netductor/secrets/github_token <<EOF
   ghp_…
   EOF
   ```
4. Проверка: `netductor update status` / скачивание asset из private release.
5. Web/TG: Settings → GitHub token Save (пишет тот же secret на primary через session API).

## Private releases

Stack apply / agent upgrade читают GitHub API с token. Без token private release → 404.

## git projects

`netductor git project add org/repo` — sync через token или deploy key по политике репо.

## Не коммитить

PAT только в secrets (0600), не в git, не в TG-сообщениях, не в job log.

## Отключить Actions на аккаунте

Личный GitHub: по репам API или UI Settings → Actions → Disable. Org: org-level disable.

См. также runbook в handoff и `RELEASE.md`.
