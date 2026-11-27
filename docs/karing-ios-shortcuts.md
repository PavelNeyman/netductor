# Karing + iOS Shortcuts (iOS 18 / 26 / 27)

Official Karing schemes ([karing.app scheme](https://karing.app/en/cooperation/scheme)):

```
karing://install-config?url=<urlencoded YAML URL>&name=<profile name>
karing://connect
karing://disconnect
karing://reconnect
```

Enable in Karing: **Settings → System Scheme Call → karing://** (and clash:// if needed).

There is **no** scheme to “switch active profile to X”. Practical pattern:

1. Import **two** netductor profiles once (e.g. `operator-mobile` + `operator-fullproxy`) via TG **Karing import**.
2. Automations only **reconnect** (or re-import if you insist on one active config).
3. Optional: Karing app action in Shortcuts — toggle VPN **On/Off** (see Karing FAQ: iOS Shortcuts → Karing → ON switch).

Wi-Fi automations can run **without asking** on current iOS (Apple list includes Wi-Fi among auto-run triggers). Turn off **Ask Before Running**.

---

## A. One-tap import (manual shortcut)

1. Shortcuts → **+** → **Open URLs**
2. URL = deep link from Telegram (**Karing import** button), or:

   `karing://install-config?url=<YOUR_SUB_BASE>/sub/<TOKEN>/karing/operator-mobile.yaml&name=nd-operator-mobile`

3. Save as `nd · import mobile`.
4. Repeat for `operator-fullproxy` / `family`.

---

## B. Automation: join home Wi‑Fi → reconnect

1. Shortcuts → **Automation** → **+** → **Wi-Fi**
2. Network: your home SSID(s) (or Any Network if you prefer)
3. **Is Joined**
4. Run immediately / **Don’t Ask**
5. Actions:
   - **Open URLs** → `karing://reconnect`  
     (or `karing://connect`)
6. Optional second action: **Show Notification** “Karing reconnect (Wi‑Fi)”

## C. Automation: leave Wi‑Fi (→ cellular) → reconnect

1. Automation → **Wi-Fi** → same home SSID → **Is Left** (or disconnected)
2. Don’t Ask
3. **Open URLs** → `karing://reconnect`

LTE vs Wi‑Fi then both force a clean reconnect; pick the right profile in Karing UI once, or re-run import shortcut for the profile you want on cellular.

## D. Cellular-focused profile (optional)

If you want **fullproxy** only on mobile data:

1. Import both profiles from TG.
2. On **Is Left** home Wi‑Fi automation, add **Open URLs**:

   `karing://install-config?url=.../karing/operator-fullproxy.yaml&name=nd-operator-fullproxy`

   then `karing://reconnect`

3. On **Is Joined** home Wi‑Fi:

   `karing://install-config?url=.../karing/operator-mobile.yaml&name=nd-operator-mobile`

   then `karing://reconnect`

Re-import is heavier than SR’s in-client SSID rules, but it is what Karing’s public API allows.

---

## E. System VPN On Demand

**Settings → VPN → Karing → Connect on Demand** (or Karing **Always keep connected**).

Keeps tunnel up; does **not** switch netductor rule profiles.

---

## Telegram

User card → **Karing** → profile → **Karing import** (YAML + deep link).  
**Shortcuts** button opens this setup summary in the bot.
