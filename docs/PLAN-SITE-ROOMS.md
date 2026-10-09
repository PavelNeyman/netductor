**EN** · [RU](ru/PLAN-SITE-ROOMS.md)

# Plan: Site rooms / zones

**Status:** planned (2026-10-09) — implement after this doc is accepted.  
**Architecture:** single backend API + thin UIs (Web / TG / TUI / CLI). No second store. No public photo URLs.

## Goal

Inside an existing **Site** (location: home / flat / office — already `internal/sites`), group **rooms** (kitchen, yard, office) with optional notes, tags, one photo, and **links** to NVR cameras / edge device ids. Rooms are inventory + navigation, **not** a new recorder or live stack.

## Non-goals

- Frigate / AI / map floor plans  
- Public or unauthenticated photo CDN  
- TG-deploy of edge from room card  
- Replacing Site / MikroTik / RPi binding  
- Nested rooms beyond one level  

## Model

```
Site (existing: sites.json)
 └── Room
      id, name, description, tags[]
      camera_ids[]   → nvr.Camera.ID (soft links)
      edge_ids[]     → agent device_id (optional)
      notes
      photo           → optional single file next to JSON
```

### Room JSON

Path: `/var/lib/netductor/sites/<site_id>/rooms/<room_id>.json`

```json
{
  "id": "kitchen",
  "site_id": "home",
  "name": "Kitchen",
  "description": "",
  "tags": ["indoor"],
  "camera_ids": ["cam-aabbcc"],
  "edge_ids": [],
  "notes": "",
  "has_photo": true,
  "updated": 0
}
```

Photo (optional): `/var/lib/netductor/sites/<site_id>/rooms/<room_id>.jpg` (or `.png`).  
**Id:** `[a-z0-9-]{1,32}`. Max photo **2 MiB**, jpeg/png only.

Site registry stays in `sites.json`. Rooms never invent a Site.

## Backend

Package: `internal/sites` (extend) or `internal/sites/rooms.go`.

| Op | Behavior |
|--|--|
| ListRooms(siteID) | read dir, parse JSON |
| GetRoom / UpsertRoom / DeleteRoom | atomic write; delete removes JSON + photo |
| SetPhoto / GetPhoto | write/read bytes; reject oversize / bad type |
| Validate links | camera_ids: warn if unknown NVR id (do not hard-fail); edge_ids optional check against edge registry |

### API (session auth only)

| Method | Path | Body / notes |
|--|--|--|
| GET | `/api/sites` | already exists |
| GET | `/api/sites/{id}/rooms` | `{ rooms: [...] }` |
| POST | `/api/sites/{id}/rooms` | upsert room JSON (no photo binary) |
| POST | `/api/sites/{id}/rooms/delete` | `{ "id": "kitchen" }` |
| GET | `/api/sites/{id}/rooms/photo?id=` | image bytes, `Content-Type`; **session required** |
| POST | `/api/sites/{id}/rooms/photo` | multipart or base64 JSON; max 2 MiB |

CLI:

```text
netductor sites rooms list <site>
netductor sites rooms add <site> --id kitchen --name "Kitchen" [--cam id]*
netductor sites rooms delete <site> <room>
netductor sites rooms photo <site> <room> /path/to.jpg
```

opcatalog entries under section `nodes` or new `sites` (same as Sites today).

## UI (thin)

Same pattern as NVR: **list → card**, no duplicate stores.

| Surface | Behavior |
|--|--|
| **Web Control** | Site picker → rooms table → form add/edit (name, tags, camera multi-select from `/api/nvr/cameras`) → photo upload → delete. Photo via session GET. |
| **TG** | Fleet → Sites → site card → **Rooms** → list buttons → room card: name, tags, **one photo** (sendPhoto with bytes from API/file, not public URL), buttons → linked cams (`m:nvr:card` / live link). |
| **TUI** | list / add name+id / delete — no photo binary in TUI. |
| **Installer tab** | optional later; Control is enough for v1. |

### Live / cams

Room does **not** embed players. Links only:

- TG/Web: open existing NVR card / `/api/nvr/live?id=`  
- Same VPN/session rules as NVR UI plan  

## Phases

### Phase 0 — Store + CLI (no UI)

- [ ] `Room` type + load/save under `sites/<id>/rooms/`  
- [ ] CLI list/add/delete/photo  
- [ ] Unit tests: id validation, photo size reject  

**Exit:** `netductor sites rooms list home` works on primary.

### Phase 1 — API + opcatalog

- [ ] Register routes under session mux  
- [ ] opcatalog: rooms list / upsert / delete / photo  
- [ ] Doctor optional: sites dir writable  

**Exit:** curl/session can CRUD a room + fetch photo.

### Phase 2 — Web

- [ ] Control: Sites/Rooms panel (table + form + photo)  
- [ ] Camera multi-select from NVR list  

**Exit:** operator manages rooms without CLI.

### Phase 3 — TG + TUI

- [ ] Site card → Rooms → room card + photo  
- [ ] TUI list/add/delete  

**Exit:** parity of **data**; Web remains richest for photo upload.

## Security

- All API behind **operator session** (same as NVR).  
- Photos never under plain static `/public`.  
- TG sends photo as bot media from primary file read, not a world-readable HTTP link.  
- No cross-site room id collision: path is scoped by `site_id`.  

## Depends on

- Existing `internal/sites` Site registry (**done**).  
- NVR camera ids for links (**done**; soft link).  
- Hardware e2e **not** required to ship store/API/UI.

## Out of order / later

- Multiple photos per room  
- Floor plan image  
- Auto-assign cam by DHCP hostname  
- Rooms in backup/restore explicit path (state dir already in backups if path is under `/var/lib/netductor`) — verify once in Phase 1  

## Acceptance

1. Create site `home` (existing CLI).  
2. Add room `kitchen` with one cam id + jpeg.  
3. Web lists room, shows photo, edits tags.  
4. TG shows room card with photo and cam button.  
5. Delete room removes JSON + photo.  
6. No photo reachable without session / bot.

---

**Next:** implement Phase 0 → 1 → 2 → 3 when operator says **делай**.
