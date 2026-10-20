# Day-2 группы (общая IA)

**Источник истины:** `internal/opcatalog.Groups()` + `Action.Section`.

| Группа | Sections |
|--------|----------|
| home | overview, updates |
| users | vpn |
| fleet | nodes |
| edge | edge |
| media | nvr |
| data | backup, dns, git |
| adv | probes |

**Поверхности**

- **Web Control** — вкладки из `groups` в `GET /v1/catalog`
- **TG Tools** — `toolsHubHTML` → `m:cat:<group>` → actions `m:op:<id>`
- **TUI Ops → Day-2 catalog** — те же Groups / Actions; CLI через remote `netductor`

**Правило:** новая day-2 возможность → `Action` в `opcatalog.All()` с section из Group. Не выдумывать вкладки только в UI.
