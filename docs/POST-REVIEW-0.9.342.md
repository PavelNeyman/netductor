# 0.9.342

- Auto **recover-drill** mark after successful `Recover` / `RecoverFromSecondary` (`MarkRecoverDrill`); CLI `backup drill-mark` uses same helper
- Action required: backup verify **missing** / **stale ≥14d** (in addition to fail)
- OPEN_ITEMS: guest read-only session + OIDC deferred
- SR family/operator already shipped (TG `btn_sr_config` / `btn_sr_family`) — no change
