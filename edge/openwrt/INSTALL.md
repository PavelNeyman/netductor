# Netductor edge agent (OpenWrt)

Outbound-only Go agent. No shell site installer in-tree.

```sh
ARCH=arm64   # arm | amd64 | mipsle
TAG=v0.8.20
curl -fsSL -o /usr/sbin/netductor-agent \
  "https://github.com/PavelNeyman/netductor/releases/download/${TAG}/netductor-agent-linux-${ARCH}"
chmod 755 /usr/sbin/netductor-agent
mkdir -p /etc/netductor-agent
cat > /etc/netductor-agent/config <<CFG
SERVER=https://your-vps:8787
TOKEN=edge-token-from-vps
DEVICE_ID=site1
INTERVAL=60
CFG
chmod 600 /etc/netductor-agent/config
# procd init — see releases notes
```
