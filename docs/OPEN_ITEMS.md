# Open items

## Owner / later

- Hardware e2e (OpenWrt guest, Tapo NVR)
- SMTP alerts (mailbox)
- Mobile day-2
- Full WebUI form-label i18n
- Finish MikroTik Web/CLI parity; NVR already Control API

## Not doing

- Public VPS admin
- Deploy from Telegram
- Password day-2 tunnel


## Ideas / later design (not scheduled)

### MikroTik API via RPi agent (optional)

- Do **not** expose RouterOS API to the internet.
- Edge agent on LAN RPi talks to MikroTik API; reports to primary over existing mTLS (:8789).
- Bootstrap stays Mac → SSH/RSC (`DeploySite`); API path is day-2 observe/control only.
- Requires API enabled on MT, bind LAN-only, least-privilege user, secrets only on Pi.

### Sites: rooms / zones (Web-first)

- Model: Site → Room (name, description, tags, photos) → devices (cameras, APs, …).
- Storage on node (`/var/lib/netductor/sites/…`); photos not in git; serve via session API only.
- **Web:** full CRUD + gallery. **TUI:** list/ids only, no photos. **TG:** read-mostly (room card + one photo + buttons).
- After Fleet/MikroTik parity and hardware e2e; not a deploy blocker.
