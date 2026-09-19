# Edge agents after primary reinstall

## Scenario

Primary (control plane) is wiped/reinstalled. OpenWrt routers keep the same agent binary and local config (`device_id`, old `device_token`, server URL).

## What actually happens

Edge trust is **state on primary** (`/var/lib/netductor/edge/…`) plus secrets (`edge_bootstrap_token`, per-device tokens).

| Step | Behavior |
|------|----------|
| 1. Clean primary | Registry empty, **new** `edge_bootstrap_token` |
| 2. Agent heartbeat with **old** `device_token` | **401** — token unknown / not approved |
| 3. Agent re-enroll with **bootstrap** | Works **only if** router still has the **new** bootstrap (or operator updates it) |
| 4. Enroll success | Device appears as **`pending`** (same stable `device_id` if unchanged) |
| 5. Operator **Approve** | New `device_token` issued; agent must store it and then heartbeat works |

So the “waiting list” (**pending → approve**) is correct **after a successful enroll**. It is **not** automatic merely because the router is online with an old token.

## Not automatic

- Old `device_token` alone does **not** restore `approved` (by design — prevents silent takeover of a fresh primary).
- If bootstrap on the router is stale, enroll fails until the operator updates bootstrap (or re-runs edge provision with the new token).

## Preferred recovery paths

1. **Restore primary from backup** (`.ndenc` / state) that includes edge registry + secrets → routers keep working without re-approve.  
2. **Clean primary**: distribute new `edge_bootstrap_token` to agents → enroll → **pending** → approve in TG/Admin/CLI.  
3. Optional future: “recovery code” / mTLS client cert bound to device — not implemented.

## Operator checklist (clean primary)

1. Install primary; note new bootstrap token.  
2. Update each router agent config (bootstrap + server URL if IP changed).  
3. Wait for pending list / TG notify.  
4. Approve known MAC/board/`device_id`.  
5. Confirm heartbeat / metrics.



## LAN recovery page (preferred for remote sites)

1. On **new** primary: `netductor edge recovery [--site home-msk]` → one-time code (24h).
2. Optional: `netductor edge register <device_id> --site home-msk` (pre-declare).
3. On site Wi‑Fi open `http://<router-lan-ip>:7879/netductor-recovery`.
4. Enter **Primary URL** + **code** → agent writes config with `CONTROL_ONLY=1` (no UCI/template apply).
5. Agent enrolls → **pending** → operator **Approve** in CLI/TG.
6. Link inventory: `netductor edge set-site <device_id> <site_id>` (Locations / sites).

Firewall: publish **:7879 only on LAN**, never WAN.

Export/import edge registry: `netductor edge export -o …` / `import …`.
