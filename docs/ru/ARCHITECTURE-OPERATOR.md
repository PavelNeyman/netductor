# Оператор vs нода

**RU** · [EN](../ARCHITECTURE-OPERATOR.md)

Два бинаря:

| Бинарь | Где | Роль |
|--------|-----|------|
| `netductor-op` | Mac/PC | TUI, deploy, operator serve, credentials |
| `netductor` | VPS | install, serve, vpn, doctor, agent plane |

Деплой качает **node**-asset с Release, не op-бинарь.  
Новые use-case: сначала backend (`operator`/`deploy`/node API), потом все UI.
