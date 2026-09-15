# Telegram operator bot UI

## Layout rule

| Placement | Purpose | Examples |
|-----------|---------|----------|
| `reply_markup` (under message) | Navigation | Main menu, Users, Nodes, back |
| Message body (`<tg-button>`, tables) | Actions for current screen | Access, rename, enable/disable, VLESS/Core/HY2, node upgrade |

Never put the same action in both places (e.g. Add user + Main menu duplicated).

## Code

- `cmd/netductor-tg/keyboards.go` — under-message keyboards  
- `cmd/netductor-tg/format.go` — HTML + in-body buttons  
- `cmd/netductor-tg/handlers_cb.go` / `handlers_msg.go` — callbacks  

## Screens

- **Users list:** table/rows + in-body open/access/rename/add; under: Main menu  
- **User hub:** enable/disable/revoke/access in body; under: Users + Main menu  
- **Access + QR:** mode switch VLESS/Core/HY2 in body; under: User card + Users + Menu  
- **Node card:** metrics/journal/upgrade/reboot in body; under: Nodes + Menu  
