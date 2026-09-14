# Тесты VPS (русский)

**EN:** [../VPS-TESTS.md](../VPS-TESTS.md)

Единый раннер оборачивает распространённые community-пробы. Скрипты идут во **временный каталог** и **удаляются при выходе** (trap). Сводная таблица — только в терминал.

## Запуск

```bash
sudo netductor probe                 # меню checklist
# curated third-party scripts: see docs/VPS-TESTS history / operator scripts
sudo netductor probe        # отобранный набор
sudo netductor probe            # долго
sudo netductor probe yabs sysbench
sudo bash install.sh --tests
```

| Id | Источник |
|----|----------|
| ipregion | ipregion.vrnt.xyz |
| censor_geoblock / censor_dpi | vernette/censorcheck |
| iperf_ru | itdoginfo/russian-iperf3-servers |
| yabs | yabs.sh |
| ip_check_place | IP.Check.Place |
| bench | bench.sh |
| ipquality | Check.Place -EI |
| sysbench | локальный пакет |

**Важно:** сторонние установщики могут оставить пакеты в системе (например `sysbench`). Мы чистим только свой temp, не откатываем apt.

Модель доверия: вы запускаете удалённые скрипты; при сомнениях — только на throwaway/test VPS.
