# Flow mismatch (VLESS Reality + Vision)

## What the log means

```
inbound/vless[vless-reality]: process connection from A.B.C.D:port:
  flow mismatch: expected xtls-rprx-vision, but got none
```

Server user has `flow: xtls-rprx-vision`. Client authenticated (UUID OK) but sent **empty flow**.

This is **not** a random scanner: scanners without a valid UUID do not reach this error the same way (Reality often falls through to handshake dest). A valid UUID + empty flow = **our client profile without Vision**.

Official behaviour (sing-box / Xray): server and client **must** both set `xtls-rprx-vision` for TCP+REALITY Vision inbounds.

## Typical causes

1. **Old share link / QR** saved before `flow=` was in the URI.
2. **Client UI** cleared Flow (Happ / Shadowrocket / v2rayN “Flow” empty).
3. **Several profiles** still point at **core** (`netductor.work.gd` / primary IP) while you only “use” secondary — iOS/macOS may keep background reconnect on inactive servers.
4. **Config / rule mode** in Shadowrocket that rebuilds outbound without `flow`.
5. **Second device on same NAT** (phone, TV, another OS user) still on old profile — alerts show **one public IP** for the whole home.

## What we saw on primary (2026-09-17)

- Source IP **46.147.247.69** = home ISP NAT (not secondary `92.255.77.253`).
- Secondary uplink to primary uses `flow=xtls-rprx-vision` and connects cleanly from **92.255**.
- Generated links include `flow=xtls-rprx-vision` (core and secondary).
- All server users have Vision flow set.

So spikes were almost certainly **a home client talking to core without flow**, not a sing-box bug and not the RU secondary uplink.

## How to verify

On primary:

```bash
journalctl -u sing-box --since "1h" | grep "flow mismatch"
# optional debug (noisy):
# set log.level=debug in /usr/local/etc/sing-box/config.json && systemctl restart sing-box
```

On client:

- Open profile → **Flow = xtls-rprx-vision**
- URI must contain `flow=xtls-rprx-vision`
- Delete duplicate servers aimed at core IP/domain if you only want secondary
- Fully disconnect VPN / kill client process after changing profiles

## Alert spam

Mismatch alerts aggregate by source IP over 30m. One broken profile can retry dozens of times → “48 in 30m”. Fix the client; optional future: raise threshold or suppress known home IP after first alert.
