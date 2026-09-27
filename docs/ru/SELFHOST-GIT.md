**RU** · [EN](../SELFHOST-GIT.md)

# Self-hosted git (thin)

## Зачем
Приватный git + простой CI/registry без тяжёлого GitLab; интеграция в netductor UI.

## Модель
Bare repos под state; `netductor git` CLI/API; список репо, commits, diff; pipeline trigger минимальный; optional docker registry addon.

## Не делаем
Полноценный Forgejo/GitLab с issues/MR UI как продукт.

## Деплой
Через COMPONENTS/addons на primary; данные в бэкапе state; бинарники с Release.
