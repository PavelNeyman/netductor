**RU** · [EN](../AGENT_HANDOFF.md)

# AGENT Handoff (RU)

**Стоп-линия: v0.9.441** (2026-10-20). offline USB apk + deps e2fs.


## Security stop (2026-10-20)

See SECURITY-REVIEW-2026-10-20.md. Tip **v0.9.445** closes H-1/H-3/H-2/N-10 and M-3/M-5/M-7/N-8/L-3. Open: M-1, M-4, N-9, M-2, M-6, L-1.

## Где остановились

| Тема | Состояние |
|--|--|
| Релиз | tip **v0.9.427**; последний опубликованный релиз на GitHub **v0.9.426** (0.9.427 ещё не выпущен); `Formula/netductor-op.rb` на 0.9.426 с настоящим SHA256. Formula — **только netductor-op** |
| Последнее ревью | Полный комплекс, **статический**, 2026-10-20 (тесты НЕ запускались, код не менялся): [REVIEW-REPORT-2026-10-19.md](REVIEW-REPORT-2026-10-19.md); план и открытые пункты F1–F10: [REVIEW-PLAN-2026-10-19.md](REVIEW-PLAN-2026-10-19.md) |
| Перед следующим тегом | выполнить `go test ./internal/... ./cmd/... -count=1` (в ревью 2026-10-20 пропущено) |
| Главные находки ревью | S-1 sub-токен VPN не отзывается вместе с пользователем (повторное имя оживляет старую ссылку); S-2 recovery TLS молча откатывается на skip-verify; Q-1 в Web не хватает четырёх i18n-ключей (`edge_st_g_bak`, `rd_mode`, `rd_pos`, `result_hint`); ~45 проверенных мёртвых функций (DC-A) |
| Web UI | Спокойная палитра (slate-акцент), Operate сгруппирован в op-cards, топология control plane на Servers, **Result dock** в Operate (снизу/справа/сверху, вкладки Log/History) |
| VPN config | обновление бинаря **не** переписывает sing-box. Новый JSON только после `vpn apply` или sync secondary |
| Отложено | Web terminal + FM; read-only guest-сессия оператора; OIDC (только дизайн) |
| Открытый lab | dual-node smoke, железо e2e (OpenWrt/Tapo/MikroTik) |
| Долг по докам | у 19 EN-документов нет RU-пары (находка D-1) |

## Последние версии (0.9.421–0.9.427)

- **0.9.427:** offline-prep использует закэшированный `edge_bootstrap_token`, если SSH на primary `:52222` недоступен; понятнее подсказка при закрытом соединении (fail2ban / ключ / passphrase).
- **0.9.426:** USB-пакеты OpenWrt сначала из офлайн-кэша на Mac; вкладка **Log** в Result dock показывает stdout job/deploy; карточка edge probe нормализует `{facts, plan}`.
- **0.9.425:** «резиновая» раскладка Result dock; позиция через select; JSON больше не печатает `null`; op-cards/таблицы адаптивные.
- **0.9.424:** панель иконок в Result dock; позиция ⬇/➡/⬆ работает всегда; разумный размер при смене ориентации.
- **0.9.423:** секции BTN в Operate привязаны к панелям shell; inline-превью под нажатым действием; распаковка конверта nodeFetch; понятнее подсказка про 401.
- **0.9.422:** Operate Result dock (в стиле VS Code) — позиции, режимы (панель / карточка / обе), History (последние 40), настройки в `operator.conf` (`result_*`).
- **0.9.421:** спокойный Web UI; op-cards в Operate; карточки Edge USB day-2; топология на Servers (SP/PS/mTLS, uplink, PB/BP).

План аудита Web: `UI-BUTTON-AUDIT.md` (прохождение оператором каждой кнопки).

## Архив — сводка 0.9.320–0.9.349

- Metrics UI, recovery-алерты EN/RU, таблицы Channel
- unattended-upgrades, integrity (+SSH), backup age, SERVFAIL
- Reboot: TG + **API/Web/opcatalog** (код 3 мин, one-shot)
- Blocky DoT на primary; SSH allowlist (`m:sshkeys:`)
- **0.9.341–342:** Action required на Status; `stack preflight` + SHA; VPN kind user/device; auto recover-drill mark; backup-verify age в Attention
- SR family/operator уже в TG (`btn_sr_config` / `btn_sr_family`)

Apply (после публикации assets):

```bash
netductor stack preflight v0.9.349
netductor stack apply v0.9.349
netductor secondary cmd <id> upgrade:v0.9.349
```

## Архив — ранние стоп-линии (история)

- _Исторический стоп v0.9.414_ (2026-10-17): thin-UI jobs; журнал [CLEANUP-PASSES-2026-10](CLEANUP-PASSES-2026-10.md).
- _Исторический стоп v0.9.406:_ probe ставит USB kmod, затем facts. _v0.9.405:_ USB kmod OpenWrt с кэша Mac. _v0.9.404_ (tip=prod): hygiene 0.9.403…404; DNS custom URL; recovery CA pin.
- _Исторический стоп v0.9.403_ (2026-10-16): комплекс ревью [REVIEW-SUITE](REVIEW-SUITE.md).

Полная EN-история (журнал версий 0.9.277–0.9.414, заметки 0.9.1xx): [AGENT_HANDOFF.md](../AGENT_HANDOFF.md).
