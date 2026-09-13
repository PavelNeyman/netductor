# Автоматизация площадки MikroTik + RPi

## Модель
- **Site** — логическая запись (`sites add`)
- **RPi OpenWrt** — edge-агент (как Cudy), VPN
- **MikroTik** — только маршруты (RSC / push-rsc)

## Порядок

### A. С core (чеклист)
```bash
netductor sites bootstrap --id home --name "Home" --rpi-lan 192.168.88.2
```

### B. RPi
1. OpenWrt на RPi, LAN в сторону MT (например `192.168.88.2/24`).
2. Agent из releases (arch arm64/armv7) → enroll к core.
3. Approve в Admin/TG → получить `device_id`.
4. ```bash
   netductor sites add --id home --name Home --rpi <device_id> --mt mt-home
   ```

### C. MikroTik
```bash
netductor sites rsc home
# вставить в Terminal
# или API push-rsc с host/password
```
Маршрут: LAN → gateway RPi (`192.168.88.2`).

### D. Проверка
Клиент LAN → MT → RPi (agent VPN) → relay → core.

Wi‑Fi на RPi обычно не нужен — AP на MT или отдельно.
