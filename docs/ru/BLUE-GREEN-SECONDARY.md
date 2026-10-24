# Blue-Green Secondary (приёмка R5)

Базовая версия кода: **v0.9.499+**. Полный текст: [../BLUE-GREEN-SECONDARY.md](../BLUE-GREEN-SECONDARY.md).

Кратко: `inventory/register` → `peer add` (слот) → bootstrap B → `validate` → `ready` → тест по IP B → `promote` → DNS → окно наблюдения → `rollback` / `remove`.

**Promote ≠ DNS. Register ≠ deploy.**
