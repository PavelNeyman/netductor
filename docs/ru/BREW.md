**RU** · [EN](../BREW.md)

# Homebrew

- Tap: `pavelneyman/netductor` (репозиторий Formula)
- **Никогда** `sha256 :no_check` — brew отказывается / ломает install
- После релиза: `sha256sum` артефактов `netductor-op-*` / `netductor-*` → вписать в Formula
- `brew reinstall netductor` / `netductor-op` после обновления SHA
- Workstation binary: **netductor-op**; node binary на VPS не из brew по умолчанию
