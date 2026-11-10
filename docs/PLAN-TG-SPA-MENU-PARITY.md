# TG SPA menu parity

Widget profile: **no free-text input**. Workarounds:

## Text input via chat message

1. SPA button → `beginTextInput` (prompt message, id in `chatExtra`)
2. User sends value
3. `completeInput` **deletes user message + prompt**; short ack auto-deletes ~45s
4. State cleared when flow finishes

Used for: rename, add user, custom quota GiB.

On-screen pad remains available as alternate for short values.

## QR

Not embedded in widget. User card **QR** → photo (or link) in chat → **deleted after 60s**.

## Other SPA actions

Alerts pick, git sync/ship, edge guest/luci, stack, reboot challenge, backup restore, quota presets, link/sub.
