# Plan: Secondary SSH password → key-only

## Goal

Password only for UI bootstrap connect. After operator pubkey is installed and **independently** verified with a new key-only SSH session, all further Secondary provision uses key-only. No password fallback. Harden is fail-closed with `sshd -t` and no `pkill -f`.

## Sequence

1. **Bootstrap** — dial port (22) with password only + TOFU host key callback.
2. **InstallOperatorKeys** — append operator pubkey; verify line present.
3. **Key-only gate** — close bootstrap; dial again with PublicKeys only (operator private key); run `true`; fail closed on error.
4. **Transport KeyVerified** — zero password from in-memory ProvisionIn; refuse password client.
5. **Install / join / health** — key client only.
6. **Harden** (later slice) — backup configs, `sshd -t`, allow new port, apply, verify key on 52222, then disable password; independent rollback timer; no pkill.

## Non-goals

- UI password field removal
- Changing TOFU host\|port model
- Automatic promote / traffic cutover

## Status

- [x] Audit map recorded (WORK-SUMMARY)
- [x] Key-only gate in ProvisionFromCore + pass operator private key on initial
- [x] Remove pkill; add sshd -t before restart (minimal harden safety)
- [x] Full independent harden rollback systemd timer (8m) + prepare/finalize split
- [ ] Integration tests against temporary sshd

- [x] Two-phase harden + rollback timer (0.9.618)
