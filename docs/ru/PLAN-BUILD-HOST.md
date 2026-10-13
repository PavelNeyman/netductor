
## Статус (0.9.374)

- Agent plane **:8789** отдаёт `/api/build-host/*`
- Job-маршруты: Bearer + host ownership
- Enabled = online agent (stale registry не ломает ship)
- TTL онлайн 3 мин
- `build-host test` — статус, без SSH с primary
- `build-host remove <id>`

# PLAN-BUILD-HOST — отдельный VPS под сборку

**0.9.373:** removed primary-side `gitstore.DeployBuildHost`/SSH probe; WatchBuildHosts uses heartbeat only.


**0.9.372:** `BuildHostEnabled` uses build-hosts registry (Mac deploy); legacy primary SSH deploy path removed from CLI; `RunRemoteTestBuild` stubbed; Web/TG i18n polish.


Управление (jobs, UI, publish, git) остаётся на **primary**.  
Опциональный VPS только **исполняет** test/build и отдаёт артефакты обратно.

Сейчас: режим **SSH** (0.9.361).  
Позже: agent по mTLS по аналогии с secondary.

```bash
netductor build-host set --host IP --port 52222 --enable
netductor build-host test
```

Без build-host — как раньше, сборка на primary.  
Darwin/Mac-wait не меняется.
