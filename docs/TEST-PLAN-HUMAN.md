# Human test plan (owner only)

**RU:** [ru/TEST-PLAN-HUMAN.md](ru/TEST-PLAN-HUMAN.md)

What an AI **cannot** do without your hands / devices / accounts. Do these after a **clean VPS** recreate.

## Before install

1. [ ] New Debian VPS, note public IP.
2. [ ] Put **your SSH public key** in `root` (or default user) `authorized_keys`.
3. [ ] Optional: Telegram bot token + your numeric admin id.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/Netductor/main/bootstrap.sh -o /tmp/fv.sh
sudo bash /tmp/fv.sh
# or: curl … | sudo bash
```

4. [ ] Install finishes without hard error; `cat /etc/netductor/READY.txt` shows subscription lines.
5. [ ] Still can SSH in (password auth should be off **only if** key worked).

## VPN from real clients

6. [ ] Phone: import **subscription** (or VLESS line) into preferred app (e.g. Streisand / Hiddify / NekoBox).
7. [ ] Browse / messenger works over VPN.
8. [ ] Optional: test HY2 profile (`insecure` expected for self-signed).
9. [ ] Laptop client same check.
10. [ ] TV / other device if you care.

## Operator flows you own

11. [ ] `sudo netductor vpn add alice` → send her subscription out-of-band → she connects.
12. [ ] `disable` / `enable` / `revoke` as you intend.
13. [ ] Telegram bot (if enabled): only **your** account can run commands.
14. [ ] Shortcuts + API (if you use them): session token, Face ID gate, only over VPN/tunnel.

## Panels (browser)

15. [ ] `ssh -L 3001:127.0.0.1:3001 -L 8090:127.0.0.1:8090 -L 8091:127.0.0.1:8091 root@VPS`
16. [ ] Kuma: create admin once, optional monitors.
17. [ ] OpenSOHO / Beszel: login with secrets under `/etc/netductor/secrets/`.

## OpenWrt (physical router)

18. [ ] Flash/reset router if needed; copy `openwrt/` + `site.conf` with real Wi-Fi password.
19. [ ] Run `install-openwrt.sh`; Wi-Fi SSIDs/password work from phone.
20. [ ] Optional: VPN client on router; LAN devices exit via VPS as designed.

## Subjective / policy

21. [ ] Latency and streaming quality acceptable for your city.
22. [ ] `netductor probe` results make sense vs provider expectations.
23. [ ] Decide whether HY2 `insecure=1` is acceptable for your threat model.

## Report back

Paste: install log tail, `netductor doctor` output, client app name, any FAIL lines. AI can then fix code.
