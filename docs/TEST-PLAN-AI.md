# AI test plan (on VPS with SSH access)

**RU:** [ru/TEST-PLAN-AI.md](ru/TEST-PLAN-AI.md)

Run **after** owner provides SSH to a clean/reinstalled Debian VPS. No physical clients, no Telegram UI, no Wi-Fi feel.

## Preconditions

- Root SSH key access
- Outbound HTTPS to GitHub / codeload
- Debian 12+ preferred

## Phase A — Bootstrap install

1. Prefer materialize script:
   ```bash
   curl -fsSL https://raw.githubusercontent.com/PavelNeyman/Netductor/main/bootstrap.sh -o /tmp/fv.sh
   sudo bash /tmp/fv.sh --non-interactive
   ```
   Or with config file if owner prepared `/root/netductor.conf`.
2. Capture full log. Expect exit 0 or documented warns only.
3. `test -f /var/lib/netductor/installed_version` → read version (≥ 0.4.1).
4. `test -f /etc/netductor/install.conf`.
5. `test -f /etc/netductor/clients/operator/subscription.txt` and non-empty.

## Phase B — Host tools

6. `command -v netductor && netductor version
7. `sudo netductor doctor` → exit 0; no FAIL; note WARN (variant).
8. `sudo netductor doctor` → exit 0.
9. `sudo netductor vpn list` → includes `operator` on.
10. `sudo netductor vpn link operator` → two lines (vless + hysteria2) or subscription body.

## Phase C — Services

11. `systemctl is-active sing-box blocky`
12. `sing-box check -c /usr/local/etc/sing-box/config.json`
13. `dig @127.0.0.1 example.com +short` non-empty
14. `ss -lntp | grep -E ':443|:8443|:53'` (or equivalent)
15. If API enabled: `curl -fsS http://127.0.0.1:8787/health`
16. Session: `TOK=$(sudo netductor vpn session 1 | head -1)` then
    `curl -fsS -H "Authorization: Bearer $TOK" http://127.0.0.1:8787/vpn/users`
17. Docker panels (if enabled): `docker ps` shows expected containers; ports on 127.0.0.1.

## Phase D — Idempotent upgrade

18. Re-run:
    ```bash
    sudo bash /tmp/fv.sh --upgrade --non-interactive
    ```
    or bootstrap again `--upgrade`.
19. Expect skips for healthy modules; UFW still active; same operator UUID (jq).
20. `netductor doctor` still exit 0.

## Phase E — VPN user lifecycle (server-side only)

21. `netductor vpn add aitest`
22. Artifacts under `/etc/netductor/clients/aitest/subscription.txt`
23. `disable` / `enable` / `revoke aitest`
24. After revoke, directory gone; sing-box still active.

## Phase F — Negative / safety

25. Confirm `PasswordAuthentication no` **only if** authorized_keys present.
26. Confirm no `ufw --force reset` side effects (SSH still works).
27. API without Bearer → 401.
28. Optional: `netductor probe sysbench` only (fast); full `--default` if time (long, third-party).

## Out of scope for AI

- Phone/TV apps, real Wi-Fi, Telegram chat UX, streaming QoE, payment/provider panel.

## Pass criteria

- Phases A–E green
- No lockout from SSH
- Doctor fail=0

Document failures with command + stderr for code fixes.
