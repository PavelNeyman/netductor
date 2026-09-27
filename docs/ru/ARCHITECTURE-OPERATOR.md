**RU** · [EN](../ARCHITECTURE-OPERATOR.md)

# Оператор vs нода

- **`netductor-op`** (Mac/PC): деплой, TUI, local WebUI, credentials, session/tunnel к API
- **`netductor`** (VPS): install/serve/vpn/doctor/agent/tg — **server plane**
- Один backend `internal/deploy` + API; UI тонкие
- Деплой флота с Mac: primary → secondary → domain/LE → addons
- Секреты оператора остаются на Mac (`~/.netductor/…`), private key primary→secondary не кладём
