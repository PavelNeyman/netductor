# Homebrew

**RU** · [EN](../BREW.md)

**Жёсткое правило:** в `Formula/netductor.rb` **никогда** не ставить `sha256 :no_check` — brew падает.

После каждого релиза op-ассетов: `sha256sum netductor-op-*` → вставить digests в Formula, version = tag.  
Альтернатива: `netductor-op update` с Release.
