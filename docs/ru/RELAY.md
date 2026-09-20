# Secondary (VPN entry) — ранее «relay»

> **Канонический CLI: `netductor secondary` и `netductor fleet`.** Команды `netductor relay` в текущем бинарнике **нет**.
> Полный EN: [RELAY.md](../RELAY.md) · [FLEET.md](../FLEET.md) · [PLAN-SECONDARY-VPN-ONLY.md](../PLAN-SECONDARY-VPN-ONLY.md).

RU **secondary** = только вход VLESS (не зеркало primary).

```text
Телефон --VLESS Reality--> RU secondary --uplink--> primary (зарубеж) --> Интернет
```

## С primary

```bash
netductor fleet provision-secondary --host RU_IP --password '…' [--sni api.vk.me]
netductor secondary sync
netductor fleet status
```

Вручную: `secondary export` → `scp` → на RU (`NETDUCTOR_VERSION=0.8.1` + bootstrap) → `secondary join`.

## День 2

```bash
netductor secondary sync
netductor secondary status
netductor secondary links
```
