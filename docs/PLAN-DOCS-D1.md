# Docs audit then RU (D-1) — last

**Status (2026-10-20):** RU pairs added under `docs/ru/` for keepers. Historical EN-only plans that were already deleted stay gone.

**Order:** finish code fixes → audit docs → delete low-value → only then translate. Translation is last.

## Already low value (candidates to delete or fold)

| Doc | Why |
|--|--|
| `PLAN-REMOVE-TG-TOPICS.md` | Status implemented 0.9.269. Historical. |
| `PLAN-TG-MENU.md` | Status implemented 0.9.288/289. Historical. |
| `PLAN-EDGE-DEPLOY-HYBRID.md` | Status done; still claims `edge.Harden` which has no caller. Fold a one-liner into OPEN_ITEMS or delete after F5. |
| `YOUTUBE-ADS.md` | Explainer, not operator procedure. Optional. |
| `RECOVER-BOT.md` | Useful, but example tag is `v0.9.113`. Update or fold into DISASTER-RECOVERY before translating. |

## Keep, no RU yet

`DISASTER-RECOVERY.md` (S-3 note added), `RELEASE.md`, `CI-WORKFLOWS.md`, `CLEANUP.md`, `TG-ALERTS-CHANNEL.md`, `GH-PRIVATE-SETUP.md`, `OPENWRT-LAB.md`.

## Later

After the delete/fold pass, translate the survivors EN→RU in pairs. Not before.
