# CLI ↔ TUI parity (v0.8.90)

Canonical bootstrap UI: **framed TUI** (`netductor tui` → Wizard tab).  
Legacy: `tui_deploy_wizards.go` (huh) still reachable from Tools → wizard-* for scripts.

## Deploy (Mac → two VPS)

| Capability | CLI | TUI framed wizard |
|------------|-----|-------------------|
| Primary install + harden 52222 | `deploy primary` | Wizard → Primary |
| SSH key gen / path / passphrase | `--generate-key --key --key-passphrase` | gen_key, key_path, key_pass |
| Reality SNI | `--sni` | sni |
| Domain + LE | `--domain-base --le-email` | domain_base, le_email |
| Lampac / git-registry | `--with-lampac --with-git-registry` | Wizard → Add-ons |
| Telegram bot | (Add-ons / secrets on host) | Add-ons + token/admin fields |
| Secondary Mac-direct | `deploy secondary --primary --primary-key --host --password` | Wizard → Secondary (primary from Settings) |
| Primary key passphrase | `--primary-key-passphrase` | key_pass |
| Secondary existing key | `--secondary-key` | — (CLI only) |
| Credentials dump | `credentials collect` + post-deploy | post-deploy step |

## Day-2 ops

| Capability | CLI | TUI | TG |
|------------|-----|-----|-----|
| doctor / status | ✓ | Tools | ✓ |
| vpn users | ✓ | partial | ✓ |
| secondary sync/status | ✓ | partial | ✓ |
| mtls list/rotate | ✓ | Tools | Tools |
| edge list/register | ✓ | Wizard OpenWrt | ✓ |
| backup / recover | ✓ | Tools | Tools |
| nvr | ✓ | Wizard NVR | partial |
| domain / tls le | ✓ on host | via primary deploy | — |

## Secondary flow (locked)

```
Mac ──SSH key──► primary :52222
       netductor secondary prepare-pack
       ◄── JSON (token, bundle, mTLS client material)

Mac ──SSH password once──► secondary :22
       install operator pubkey (Mac .pub)
       write mTLS + join bundle
       start agent → primary :8789
       disable password auth
```

No primary→secondary SSH. Private key never leaves Mac.
