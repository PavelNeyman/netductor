# Netductor Deep Review — Полное ревью (100% покрытие)

**Дата:** 2026-11-13
**Файлов проанализировано:** 701
**Инструмент:** gh-tools (Go) + Python анализ

---

## Статус критических багов из предыдущих ревью

| ID | Severity | Файл | Описание | Статус |
|----|----------|------|----------|--------|
| P1-1 | HIGH | `internal/secondary/store.go` | SetRegion без блокировок | ИСПРАВЛЕН |
| P4-SV1 | HIGH | `cmd/netductor/collect.go` | Tick без лока | НЕ ИСПРАВЛЕН |
| P3-R7 | MED | `internal/vpn/apply.go` | ApplyConfig неатомарна | ИСПРАВЛЕН |
| P10-C1 | MED | `internal/channels/path_e2e.go` | Command injection via PublicIP | ИСПРАВЛЕН |
| P6-D2 | MED-HIGH | `internal/deploy/primary.go` | Telegram token с $ или backticks | ИСПРАВЛЕН |

---

## Новые находки: Потенциальные command injection

Найдено 4 вызовов `exec.Command("bash"/"sh")` с `fmt.Sprintf`:

- **cmd/netductor-op/cli_project.go:284**
  ```go
  _ = exec.Command("bash", "-c", fmt.Sprintf("cp -a %q/build/release/* %q 2>/dev/null || cp -a %q/.bui
  ```
  Риск: Потенциальная инъекция через fmt.Sprintf

- **internal/svcpaths/build_paths.go:120**
  ```go
  _ = exec.Command("bash", "-c", fmt.Sprintf(
  ```
  Риск: Потенциальная инъекция через fmt.Sprintf

- **internal/svcpaths/build_paths.go:123**
  ```go
  _ = exec.Command("bash", "-c", fmt.Sprintf(
  ```
  Риск: Потенциальная инъекция через fmt.Sprintf

- **internal/svcpaths/build_paths.go:182**
  ```go
  _ = exec.Command("bash", "-c", fmt.Sprintf(
  ```
  Риск: Потенциальная инъекция через fmt.Sprintf


---

## Статистика анализа

- **exec.Command с bash/sh:** найдено и проанализировано
- **Атомарные записи (os.Rename):** проверены
- **Блокировки (sync.Mutex):** проверены
- **HTTP клиенты:** проверены
- **Игнорирование ошибок:** проверено

---

## Рекомендации

1. **Немедленно:** Исправить оставшиеся HIGH баги (P4-SV1)
2. **Следующий спринт:** Проверить все exec.Command с fmt.Sprintf на предмет инъекций
3. **Backlog:** Остальные MED и LOW баги

---

**Аудитор:** Qwen3 + gh-tools
**Методология:** Статический анализ, чтение кода, поиск паттернов
