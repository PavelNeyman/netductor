**RU** · [EN](../PLAN-NVR-TAPO.md)

# ПЛАН: Tapo C200 → OpenWrt agent → primary NVR

## Хранение (locked)
Запись на **primary** (позже опционально home NAS). Через VPN only. Шифрование at-rest.

## Locked decisions
VPN-only доступ к просмотру; multi-camera/multi-site inventory; motion minimal; night/PTZ через port pytapo/Go; live с учётом privacy (ссылки/Web за VPN, не публичный TG stream).

## Факты C200
RTSP; Wi‑Fi; PTZ; IR night; не все HA plugins подходят — ориентир pytapo/официальное API.

## Agent OpenWrt
1. Список DHCP leases (IP, MAC, hostname)
2. Идентификация камер
3. Static lease
4. Путь на primary: RTSP proxy/port forward / agent relay — без записи на 16MB flash Cudy (buffer USB/sd optional)

## Recording primary
Короткие сегменты; retention настраиваемый; ffmpeg/go2rtc/minimal recorder; storage expand/NFS later.

## UI
Admin/op: список, даты, download; TG: файл по выбору; live осторожно (privacy).

## Защита
Шифрование тома/файлов; ключ не у хостера в открытом виде.

## Фазы
A discovery → B path+record MVP → C UX/live → D hardening.  
Прогресс реализации и версии NVR — хронология в EN (0.7.x–…); смысл: MVP agent+leases+recorder+TG/doctor, Go port pytapo, two-way audio via go2rtc — planned.

## Cudy
Мало flash/RAM — не писать архив на роутер; только edge/agent.

## Ссылки исследований
См. EN §8.
