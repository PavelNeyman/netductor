# 0.9.323 host security / prophylaxis

- unattended-upgrades security-only, no auto-reboot; alert host:reboot-required
- TG reboot primary/secondary with 4-digit one-shot code (3 min)
- integrity manifest: binaries + key configs + SSH authorized_keys fingerprints; collect alert integrity:drift
- backup:age alert if newest backup > 72h
- dns:servfail spike alert (sing-box journal)
- new blocky installs: DoT upstreams (Quad9/Cloudflare TLS) + plaintext fallback
- canary mismatch alerts already in place (no change)
