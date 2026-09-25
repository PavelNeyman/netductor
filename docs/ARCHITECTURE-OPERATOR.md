# Operator architecture

## Principle

```
[ Web UI ] [ TUI ] [ CLI ]
       \      |      /
        v     v     v
   internal/operator  (use-cases: DeployPrimary, DeploySecondary, DeployEdge, Fleet, credentials, tunnel, session)
        |
   internal/deploy    (SSH scripts, assets)
        |
   nodes (netductor API + agents)
```

UIs must not embed divergent deploy logic. New capability → operator/deploy first → expose to **all** thin UIs.

## Node

API server + services. No product web admin on VPS.

## Telegram

Operator-facing day-2 bot on the node. Not a second deploy path.
