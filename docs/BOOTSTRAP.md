# Bootstrap

```bash
export NETDUCTOR_VERSION=0.8.23   # optional; default in script is 0.8.1
curl -fsSL https://raw.githubusercontent.com/PavelNeyman/netductor/main/bootstrap.sh | bash
```

Installs `/usr/local/bin/netductor` from the **`v0.8.23`** GitHub Release (override with `NETDUCTOR_VERSION`).

Then:

```bash
netductor tui --mode vps
# or
netductor install
```

See [INSTALL.md](INSTALL.md) · [DEPLOY.md](DEPLOY.md).
