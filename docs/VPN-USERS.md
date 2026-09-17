See docs/ru/VPN-USERS.md


## Secondary must list every user UUID

Client preferred links target **secondary** (`vpn.*` / RU IP). Primary `sing-box` has all users; secondary `relay-in` must too.

- On user add/rename/delete: `ApplyConfig` bumps secondary `config_ver`; agent rewrites config from `ExportRelayBundle`.
- Force: `netductor relay sync` then wait one heartbeat.
- Symptom of missing UUID: operator link works, second user link to same host fails (auth/handshake error).
