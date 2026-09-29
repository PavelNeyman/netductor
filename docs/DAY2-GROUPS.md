# Day-2 groups (shared IA)

**Source of truth:** `internal/opcatalog.Groups()` + `Action.Section`.

| Group | Sections |
|-------|----------|
| home | overview, updates |
| users | vpn |
| fleet | nodes |
| edge | edge |
| media | nvr |
| data | backup, dns, git |
| adv | probes |

**Surfaces**

- **Web Control** — tabs from `groups` in `GET /v1/catalog`
- **TG Tools** — `toolsHubHTML` → `m:cat:<group>` → actions `m:op:<id>`
- **TUI Ops → Day-2 catalog** — same Groups / Actions; runs CLI mapping via remote `netductor`

**Rule:** new day-2 capability → add `Action` in `opcatalog.All()` with a section that belongs to a Group. Do not invent UI-only tabs.
