# Backbone WireGuard (опционально)

**RU** · [EN](../BACKBONE-WG.md)

Идея: служебный WG между primary и secondary (SP/PS), отдельно от пользовательского VLESS.

- Не замена SSH :52222 и mTLS :8789
- Не для ноутбуков оператора «вместо» текущего доступа
- Обсуждение стабильности uplink / обхода — в планах, не обязательный канон freeze

Полный EN-текст: детали интерфейсов, ролей, break-glass.

### Схема multiplex (sing-box 1.14+)
- Outbound uplink: max_connections и т.д.
- Inbound primary: только enabled+padding.

