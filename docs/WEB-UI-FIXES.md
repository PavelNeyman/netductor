**EN** · [RU](ru/WEB-UI-FIXES.md)

# Web UI / Fleet deploy — pending fixes

Status: **recorded only** — do not implement until operator prioritizes.  
Related error analysis below; code changes deferred.

## Labels & placeholders (Installer → Fleet / Primary)

| Field (current) | Problem | Desired |
|-----------------|---------|---------|
| **CORE host** + placeholder `p2.nd.example.com` | Confusing: form already asks **Primary host** (IP). “CORE” is internal jargon. Placeholder looks like a real example hostname, not a prompt. | Rename to **Primary domain** (or “Primary FQDN”). Placeholder/hint: *«укажите домен хоста primary»* / *“FQDN for the primary host (DNS A → primary IP)”*. |
| **VPN host** | Same: should track **secondary**, not abstract “VPN”. | Rename to **Secondary domain** (FQDN). Hint: *«домен для secondary»* / *“FQDN for secondary (DNS A → secondary IP)”*. |
| **Domain label** | No explanation what it is or why. | Short help: optional `DOMAIN=` org label in conf only; **does not** invent `p.`/`i.` hostnames; LE uses explicit primary/redirect FQDNs. |
| **SNI** | No explanation. | Hint: Reality TLS camouflage name (e.g. `api.vk.me`); written as `DEFAULT_SNI` / used by sing-box. |

Apply the same wording in **TUI wizard** and **CLI help** for parity.

## Checkboxes

| Control | Issue | Desired |
|---------|-------|---------|
| **Lampac** | Missing checkbox in Web Fleet (Git + TG present). | Add Lampac checkbox like CLI/TUI `--with-lampac`. |
| **CF proxy** | Unclear if needed. | Prefer **remove** from default UI (or hide under “Advanced”). Canonical setup is **DNS-only (grey cloud)** + origin LE on `:8443`. Orange CF often breaks multi-level names / LE on origin. Keep only if someone deliberately uses CF edge certs. |

## Secondary provision error (explained — fix later)

### Symptom

```text
POST /v1/fleet
step primary: … done
step secondary ERROR: secondary provision: ssh dial: ssh: handshake failed:
  open /var/lib/netductor/secondary/ssh_known_hosts.json: no such file or directory
```

### Cause

1. Primary step finished on the VPS (paths under `/var/lib/netductor` exist **there**).
2. Secondary provision runs **from the Mac** (`netductor-op`): SSH client + TOFU host-key callback in `internal/secondary/hostkey.go`.
3. On first connect, the callback tries to **create/update**  
   `paths.SecondaryDir()/ssh_known_hosts.json` → default **`/var/lib/netductor/secondary/ssh_known_hosts.json`**.
4. On macOS that directory is **not** writable (and usually does not exist). `MkdirAll`/`WriteFile` fail → error is wrapped as `ssh: handshake failed: open … no such file`.
5. This is **not** “secondary VPS rejected the key” and **not** a missing file on the secondary server — it is the **operator machine** TOFU store path.

### Intended fix (when implementing)

- Store operator-side TOFU under a **user-writable** base, e.g. `~/.netductor/ssh_known_hosts.json` or `$NETDUCTOR_STATE` defaulting to `~/.netductor` **on op binary**, while node keeps `/var/lib/netductor`.
- Or set `NETDUCTOR_STATE=$HOME/.netductor` in `operator serve` / deploy before secondary SSH.
- Ensure `MkdirAll` errors are not swallowed; surface “cannot write known_hosts” clearly.

### Workaround until fixed

```bash
sudo mkdir -p /var/lib/netductor/secondary
sudo chown "$(whoami)" /var/lib/netductor/secondary
# then retry Fleet secondary / deploy secondary only
```

Or export before serve:

```bash
export NETDUCTOR_STATE="$HOME/.netductor"
mkdir -p "$NETDUCTOR_STATE/secondary"
netductor-op operator serve
```

(Only helps if all TOFU reads honor `NETDUCTOR_STATE` — verify when implementing.)

## Out of scope here

- Dual-node smoke / hardware e2e (see OPEN_ITEMS).
- Release/Formula sync (done for v0.9.92 separately).
