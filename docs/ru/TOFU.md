# TOFU (Trust On First Use) для SSH

## Идея
1. **Первый** коннект — host key сохраняется  
2. **Дальше** — ключ должен совпасть  
3. **Сменился** — отказ (MITM или переустановка)

## Управление без правки файлов

```bash
netductor ssh-hosts list
netductor ssh-hosts forget 192.168.88.1:22
netductor ssh-hosts forget --kind relay 92.255.77.253
netductor ssh-hosts clear --kind mt
```

TUI: **SSH known hosts** (workstation / operator).

API (session):
- `GET /api/ssh-hosts`
- `DELETE /api/ssh-hosts?id=…&kind=mt|relay`
- `POST /api/ssh-hosts/clear?kind=…`

## Файлы
- MikroTik: `StateDir/mikrotik/known_hosts.json`
- Relay: `StateDir/relay/ssh_known_hosts.json`

## Strict
`NETDUCTOR_MT_STRICT=1` — неизвестные MT-хосты не записываются.
