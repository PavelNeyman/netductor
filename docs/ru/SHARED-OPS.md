# Общие ops-модули

**RU** · [EN](../SHARED-OPS.md)

Поведение на нескольких типах хостов — один пакет (hardening SSH, opcatalog, deploy), без копипасты скриптов.  
Пример бага: secondary оставался на :22, primary на :52222 — лечится общим DropInConf.
