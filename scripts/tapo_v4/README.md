# Tapo V4 / TPAP helper (freeKC)

MIT — https://github.com/freeKC/tapo-v4-protocol

Used by netductor `internal/tapo` when Camera Account login fails with newer firmware (SPAKE2+ / encrypt_type 4 / -40211).

## Dependencies

```bash
pip3 install requests pycryptodome
```

## CLI

```bash
python3 cli.py login CAMERA_IP 'cloud-password'
python3 cli.py exec CAMERA_IP 'cloud-password' getDeviceInfo '{}'
```

Env: `TAPO_CRED_HASH=md5|sha256` (default tries both).

Install on primary: copy tree to `/opt/netductor/scripts/tapo_v4/`.
