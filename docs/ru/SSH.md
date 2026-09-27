**RU** · [EN](../SSH.md)

# SSH

- Порт **52222** на primary и secondary после harden
- Первый вход: пароль провайдера → укладка Mac operator key → password off
- Один и тот же ключ оператора на обе VPS
- Day-2: `ssh -i ~/.ssh/netductor_primary -p 52222 root@HOST`
