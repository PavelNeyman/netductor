# SSH access to netductor nodes

## Fleet key

Core holds ed25519 at `/root/.ssh/id_ed25519`. The public key is installed on relay/edge during enroll. Password auth on relay is typically disabled after provision.

```bash
ssh -i netductor_vps_id_ed25519 root@CORE_IP
ssh -i netductor_vps_id_ed25519 root@RELAY_IP
```

Never commit private keys to git.
