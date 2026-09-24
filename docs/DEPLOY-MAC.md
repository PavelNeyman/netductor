# Deploy from Mac (TUI / CLI) — two VPS

Goal: primary + secondary fully automatic from this machine. **No primary→secondary SSH.**

## Prerequisites

- Fresh Debian VPS ×2, password SSH on :22
- DNS (Cloudflare):
  - `primary.<base>` → primary IP (DNS only)
  - `i.<base>` → primary IP (DNS only, or orange only for `i.`)
  - `vpn.<base>` → secondary IP (DNS only)
- `brew install netductor` (or release binary)

## TUI

```text
netductor tui
→ Wizard → Primary VPS
    host, password, gen key=yes, SNI, domain base, LE email
→ run (Ctrl+R / confirm)
→ Wizard → Secondary VPS
    host, password, SNI  (primary host/key already in TUI settings)
→ Wizard → Add-ons (optional)
    Telegram / Lampac / Git+Registry on chosen host
```

## CLI (parity with TUI)

```bash
netductor deploy primary \
  --host PRIMARY_IP --password '…' --generate-key \
  --sni api.vk.me \
  --domain-base netductor.neyman.top \
  --le-email you@gmail.com

# new shell: primary is on 52222
export NETDUCTOR_SSH_PORT=52222
netductor deploy secondary \
  --primary PRIMARY_IP --primary-key ~/.ssh/netductor_primary \
  --host SECONDARY_IP --password '…' --sni api.vk.me
```

| TUI field | CLI flag |
|-----------|----------|
| host / password / gen_key / key_path / key_pass | `--host --password --generate-key --key --key-passphrase` |
| sni | `--sni` |
| domain_base / le_email | `--domain-base --le-email` |
| secondary host/password/sni | `--host --password --sni` (+ `--primary --primary-key`) |
| Add-ons lampac/git | `--with-lampac --with-git-registry` on primary; TG via Add-ons wizard |

Secondary: Mac → primary `prepare-pack` → Mac SSH to secondary (password once) → harden.

## Installed automatically (core)

**Primary:** api, sing-box, blocky, domain+LE redirect :8443, mTLS :8789, metrics, backup, SSH 52222 key-only.

**Secondary:** sing-box, agent→primary mTLS, operator pubkey, password off.

**Not automatic:** Telegram / Lampac / Git+Registry → Add-ons.

## Key

`~/.ssh/netductor_primary` (chmod 600). Backup the private key.

## Verify

```bash
ssh -i ~/.ssh/netductor_primary -p 52222 root@PRIMARY 'netductor doctor'
curl -sS https://i.BASE:8443/healthz
```


## Credentials on Mac (automatic after deploy)

After primary/secondary deploy, netductor writes:

```text
~/.netductor/credentials/
  primary-<host>-<ts>/
    secrets-full.tgz   # full /etc/netductor/secrets + conf + LE + devices.json
    README.txt
    extract/
  latest-primary.txt → summary
```

Re-collect anytime:

```bash
netductor credentials collect --host PRIMARY --key ~/.ssh/netductor_primary --role primary
netductor credentials collect --host SECONDARY --key ~/.ssh/netductor_primary --role secondary --port 22
```
