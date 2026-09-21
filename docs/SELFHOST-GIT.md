# Self-hosted Git (facts)

## Question
Ready forge (Forgejo/Gitea) vs bare `git` + own wrappers in netductor?

## Facts

| Need | Bare git over SSH | Forgejo/Gitea |
|------|-------------------|---------------|
| Push/pull private source | Yes (`git init --bare`) | Yes |
| RAM | ~0 (sshd only) | typically ~150–400 MB idle |
| Attack surface | SSH keys only | HTTP(S) app + DB + optional Actions |
| Web diff / blame / PRs | No | Yes |
| Issues / projects | No | Yes |
| CI | `post-receive` hooks or external runner | Built-in Actions or Woodpecker |
| Container registry | Separate (`registry:2`) | Built-in or separate |
| Multi-operator ACL | SSH authorized_keys only | Users/teams/repos |
| netductor UI integration | Trivial (list repos, run `git`) | API possible, heavier |

## Recommendation (single operator / family)

1. **Default: bare repositories on primary over existing SSH (port 52222)**  
   - Same keys as netductor operator  
   - Behind VPN optional but SSH already key-only + fail2ban  
   - Release build: local Mac or `post-receive` → `go build` / upload assets  
   - Lowest maintenance and closest to “part of the ecosystem” without a second product

2. **Forgejo only if** you need web review, multiple writers, or in-forge CI UI regularly. Still bind to VPN/localhost.

3. **Do not** build a custom “mini-GitLab” inside netductor (web UI + auth + PR model): cost approaches Forgejo, quality lags, permanent maintenance.

## Minimal bare layout (sketch)

```text
/var/lib/netductor/git/netductor.git   # bare
git remote add vps ssh://root@PRIMARY:52222/var/lib/netductor/git/netductor.git
```

Optional later: `netductor git init|list` CLI and Admin list of bare repos — thin wrapper, not a forge.
