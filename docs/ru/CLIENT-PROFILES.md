# Клиентские профили VLESS+Reality

## Обязательно
- **flow=xtls-rprx-vision** на core и relay.
- Без flow: `flow mismatch: expected xtls-rprx-vision, but got none`.

## Ссылка
`vless://UUID@HOST:443?encryption=none&flow=xtls-rprx-vision&security=reality&sni=SNI&fp=firefox|chrome&pbk=PBK&sid=SID&type=tcp#name`

## Клиенты
| Клиент | Заметки |
|--------|---------|
| Happ | предпочтителен на LTE |
| Shadowrocket | проверить flow после импорта |
| INCY | проверить flow |
| sing-box | явно flow vision |
| v2rayN / Nekobox | не XTLS-none |

Dual: relay = основной, core = домашний.
