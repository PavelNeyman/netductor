# Ревью post-0.8.0

Полный текст: [EN](../REVIEW-2026-09-20-POST.md).

**P0:** recovery `:7879` на всех интерфейсах — только LAN/firewall; URL primary ограничить.  
**P1:** clip URL только за VPN; self-update проверять SHA256SUMS; mTLS на `:8788`.  
**Рефакторинг:** middleware API, verify release, bind recovery на private IP.
