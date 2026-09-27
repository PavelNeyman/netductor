# Self-host Git, CI, registry

**RU** · [EN](../SELFHOST-GIT.md)

## Git
Bare-репозитории под `/var/lib/netductor/git`. CLI/API/Admin/TG: list/create/delete.  
`post-receive` может запускать pipeline.

## CI
Изоляция по умолчанию (docker/podman). YAML с `run:` steps (синтаксис близок к GitHub Actions для переноса).  
`NETDUCTOR_CI_HOST=1` — escape на хост (не по умолчанию).

## Registry
Локальный OCI :5000 (часто localhost only). Образы для своих сборок.

## Обновление
Компоненты optional в COMPONENTS; recover ставит если были в манифесте.
