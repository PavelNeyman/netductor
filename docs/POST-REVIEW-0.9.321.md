# 0.9.321 — alert recovery

When `ClearAlert(key)` runs and that key was previously sent to TG, enqueue:
`✅ Recovered…` under key `ok:<original>`.

Covers svcpath, secondary offline/uplink/sb, channel, path e2e, probes, services, mismatch, firewall, backup, git/mac, generic fallback.
Log-attach companion keys (`*:log`) skip recovery to avoid noise.

Apply: `netductor stack apply v0.9.321`
