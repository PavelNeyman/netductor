# Оркестратор stack (0.9.116+)

**RU** · [EN](../STACK.md)

Тонкий слой над unit’ами и бинарниками netductor.

```bash
netductor stack status
netductor stack apply [vX]
netductor stack rollback
netductor stack watchdog
```

Apply: снимок prev → скачать node+tg → restart → health → при сбое rollback.  
Мёртвый bot чинится с SSH через `stack apply`, не через TG.
