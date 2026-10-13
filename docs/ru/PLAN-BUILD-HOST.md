# PLAN-BUILD-HOST — отдельный VPS под сборку

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
