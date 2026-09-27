# Сборка бинарников

**RU** · [EN](../BUILD-BINARIES.md)

Артефакты Release: `netductor-linux-*` (node), `netductor-tg-*`, `netductor-op-*` (darwin/linux).

`CGO_ENABLED=0`, ldflags `-s -w`. Вместе с релизом — `SHA256SUMS`. Formula SHA = digests op-файлов.
