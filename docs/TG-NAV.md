# TG navigation map (canonical paths)

One domain → **one** hub. No parallel menus.

```
Main
├── Status          metrics + services + counts only → buttons Fleet / Users
├── Users           VPN users (only place)
├── Fleet
│   ├── Nodes       list / rename / SSH hosts / secondary enroll·sync·exit
│   ├── Routers     edge devices
│   ├── Sites
│   └── Addons
├── Tools           ops only (DNS, Backup, Probes, NVR, Git, Guest, Loc, Updates, mTLS)
├── Operator        session / admin / audit / refresh links
├── Lang
└── Help
```

**Redirects** (legacy catalog sections):

| `m:ops:…` | goes to |
|-----------|---------|
| overview | Status |
| vpn | Users |
| nodes | Nodes hub |
| edge | Routers |
| nvr / dns / backup / probes / git | product hub |

**Not in Status:** full node list, full VPN list, flow mismatch dump.  
**Not in Tools:** Nodes, Users, Routers (Fleet only).  
**Not in Nodes:** Backup (Tools only).
