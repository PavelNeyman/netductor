# Web UI redesign options (operator)

**Context (0.9.413+):** Long deploys already run as **jobs** (`/v1/jobs`). Stream fallback for edge/fleet/primary/secondary/build-host is **removed** from Web. Remaining pain is **information architecture**, not the job backend.

**Related:** [PLAN-THIN-UI-JOBS.md](PLAN-THIN-UI-JOBS.md) · [ru/WEB-UI-REDESIGN.md](ru/WEB-UI-REDESIGN.md)

---



## Implementation status

| | |
|--|--|
| **Skins A / B / C** | **Done** — Settings → UI skin (`localStorage` `nd_op_skin`). Shared forms/jobs. C = menu structure, not terminal chrome. |
| **Terminal (xterm / agent PTY)** | **Deferred** |
| **File manager (agent fs_*)** | **Deferred** |

Switch: Settings → Skin (live on change); Save persists.


## Interactive mockups

After `netductor-op serve`:

```text
http://127.0.0.1:7373/mockups
```

Switcher: **A** (home cards) · **B** (sidebar) · **C** (TUI-like). Navigation works; deploy buttons are inert.


## Current problems

1. **Three top modes** (Installer / Control / Settings) mix “one-time deploy” and “day-2 ops” without a clear home.
2. **Installer sub-tabs** (Fleet, Primary, Build, Secondary, OpenWrt, MikroTik, NVR, Creds) are all equal weight — operators rarely need all of them in one session.
3. **Control** exposes a dense catalog of primary API actions (VPN, DNS, SSH, reboot…) as raw-looking buttons.
4. **Jobs** were easy to miss (now: header badge + panel); log is still a separate wall of text.
5. **Visual** language is functional dark form soup — little hierarchy, few empty states, weak progress.

---

## Option A — “Wizard home + Ops drawer” (recommended)

| Zone | Role |
|--|--|
| **Home** | 3 large cards: *New fleet / node*, *OpenWrt site*, *Day-2 on primary* |
| **Active job** | Sticky bottom or side: status, cancel, last log lines |
| **Ops** | Single “Operate” with left nav groups (Status, VPN, Edge, DNS, Backup, Advanced) |
| **Settings** | Connection + language only |

**Pros:** Matches mental model (setup vs run). Jobs always visible.  
**Cons:** Requires restructuring `index.html` panels (1–2 focused passes).

---

## Option B — “Single sidebar (app shell)”

Classic admin shell: left sidebar sections, content pane, top bar (lang + jobs + update).

- Sidebar: Deploy ▾, Sites ▾, VPN, Network, Security, Jobs, Settings  
- Content: one form or table at a time  
- Deploy wizards full-width steps (already partial for OpenWrt)

**Pros:** Familiar, scales when features grow.  
**Cons:** More CSS/layout work; mobile needs collapse.

---

## Option C — “Minimal Mac-first”

Ship **Web only for OpenWrt + Jobs + session tunnel**; keep Fleet/Primary/Secondary **TUI/CLI**.

**Pros:** Smallest Web surface, least confusion.  
**Cons:** Breaks “one UI for everything”; operator must switch tools.

---

## Shared UX rules (any option)

1. One **primary action** per screen (Deploy / Save / Apply).  
2. **Jobs** = source of truth for long work; never silent.  
3. Prefill from `operator.conf` with visible “from config” hints.  
4. Empty states with next step, not blank tables.  
5. EN/RU via existing `data-i18n` — no new hardcoded strings.

---

## Suggested sequence

1. Done: jobs-only deploy path + header jobs badge.  
2. Pick **A or B**.  
3. Implement shell + move existing panels without rewriting backends.  
4. Polish Control catalog (group + search) last.

## Hybrid shell (0.9.415)

Skins A/B/C removed. Single layout:

- Left sidebar (collapsible to icons), Settings at bottom
- Jobs in right drawer (topbar icon + statusbar)
- Bottom status bar (API/secondary/paths/channel, theme, clock)
- Content = existing Deploy / Operate / Settings panels

Design reference: operator hybrid preview (sidebar + cards density + jobs drawer).
