# Netductor Admin Shortcut (iOS / iPadOS / macOS) — русский

**EN:** [../SHORTCUT-IOS.md](../SHORTCUT-IOS.md)

**Только документация.** Shortcut собирается один раз на устройстве. VPS **не** генерирует и не раздаёт файлы `.shortcut`.

Цель: Shortcut оператора → **API через VPN/туннель** после **Face ID** + **локального файла session-токена**. Конечные пользователи этим не пользуются.

## Модель безопасности

1. API только через **ваш VPN** или SSH-туннель на `127.0.0.1:8787`.
2. **Session token** через `netductor vpn session` или Telegram `/session` — не master token внутри Shortcut.
3. **Face ID / код** в начале Shortcut.
4. Токен в **локальном файле** (На iPhone / На Mac; лучше не класть токен в iCloud).

## Выпустить session

```bash
sudo netductor vpn session 72
```

Или Telegram: `/session 72`

На телефоне: Файлы → **На iPhone** → `Netductor/session.txt` (только токен).

## Сборка на iPhone

### A. Создать

1. **Команды** → **+** → имя `Netductor Admin`.
2. **Аутентификация** (Face ID / код). При ошибке — стоп.

### B. Токен

3. **Получить файл** → `На iPhone/Netductor/session.txt` (или «Спрашивать каждый раз»).
4. **Текст из входных данных** → **Задать переменную** `Token`.

### C. Меню

5. **Выбрать из меню**: Список · Добавить · Отключить · Ссылка/QR пользователя.

### D. База API

По умолчанию: **`127.0.0.1:8787`**.

С iPhone обычно:

- **Telegram** для повседневной админки, или
- **SSH local forward**, затем `http://127.0.0.1:8787`, или
- позже bind API на VPN-интерфейс (опционально).

### E. HTTP

**Список:** `GET http://BASE/vpn/users`  
Header: `Authorization: Bearer <Token>`

**Добавить:** `POST http://BASE/vpn/users`  
Body: `{"name":"alice","note":"phone"}`  
Затем опционально `GET .../vpn/users/alice/qr` → Просмотр.

**Отключить:** `POST http://BASE/vpn/users/alice/disable`

Ответ link содержит **subscription** (VLESS+HY2).

### F. QR

PNG отдаёт сервер; Shortcut только показывает.

## Другие устройства

Тот же Apple ID — Shortcut может синкаться через iCloud. **Session-файл** при необходимости обновляйте на каждом устройстве; по возможности не храните session в iCloud Drive.

## Чеклист

- [ ] Face ID в начале
- [ ] Токен из локального файла / «Спросить»
- [ ] Нет долгоживущего master в тексте Shortcut
- [ ] API только через VPN или туннель
- [ ] Обновлять session через `/session` по истечении
