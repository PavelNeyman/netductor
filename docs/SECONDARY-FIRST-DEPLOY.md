# Secondary first deploy — clean hoster defaults

**Assumption:** first deploy means a **fresh VPS** from the operator’s point of view.
Provider images often ship UFW/firewall rules that block Netductor secondary baseline.

## Before / during first secondary deploy

1. **Restricted ports must not have administrator ALLOW** (preflight fails otherwise):
   - `80/tcp`, `8787`, `8788`, `8789`
2. Typical hoster noise to remove explicitly:
   - `ufw delete allow 80/tcp` (and IPv6)
   - ISPManager / panel ports (e.g. `1500`) — optional but recommended for secondary
3. Keep management path open until harden finalize:
   - `22/tcp` ALLOW during bootstrap
   - `443/tcp` will be managed for VLESS
4. UFW default **routed** may show as `disabled` when IP forwarding is off — treated as acceptable (no forward = deny).

## Operator checklist

```bash
ufw status numbered
# remove hoster ALLOW on 80 (required)
ufw delete allow 80/tcp
# optional: panel
# ufw delete allow 1500/tcp

netductor host-baseline apply secondary   # should exit 0
```

Do **not** `ufw --force reset` on a live box without a console path.

## Code

- `preflightSecondaryRestrictedAllows` — blocks deploy if 80/8787–8789 already ALLOW
- Secondary bootstrap keeps `:22` until SSH harden finalize
