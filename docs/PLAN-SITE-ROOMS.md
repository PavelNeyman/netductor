# Plan: site rooms / zones (deferred)

Status: **ideas only** — not in current sprint.

## Model

```
Site
 └── Room (name, description, tags, optional photos)
      └── devices (NVR cam ids, AP, notes)
```

## Surfaces

| UI | Scope |
|----|--------|
| Web Control | Full CRUD + photo upload |
| TUI | Names/ids only |
| TG | List room, one photo, deep links to cams/status |

## Storage

Node-local under `/var/lib/netductor/sites/<site>/rooms/`. API behind operator session / mTLS — no public static URLs for photos.

## Depends on

Stable site registry + NVR identity; hardware e2e optional.
