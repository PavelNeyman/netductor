# Fleet: primary + secondary

Полный EN: [FLEET.md](../FLEET.md).

- **primary** — control plane
- **secondary** — только RU VPN entry

```bash
netductor fleet provision-secondary --host IP --password '…'
netductor secondary sync
netductor fleet status
```

CLI `relay` нет — используйте `secondary` / `fleet`.
