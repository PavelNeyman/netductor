**EN** · [RU](ru/PLAN-SITE-ROOMS.md)

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


## Implementation sketch (2026-10-08)

Do not build until a site has a stable id.

1. Store `/var/lib/netductor/sites/<site>/rooms/<id>.json` plus `photo.jpg`. Id is `[a-z0-9-]{1,32}`.
2. API session: `GET/POST /api/sites/{site}/rooms`, `POST .../photo` (limit 2 MB, jpeg/png).
3. TG: site card button Rooms → list → one photo + links to cameras already in NVR. No public URL.
4. Web/TUI read the same API. No second store.

NVR two-way audio stays out of this. Cameras are links, not a new recorder.
