# Netductor Security & Code Review

**Date:** 2026-11-13 19:50

## gosec: 0 security issues

| Severity | Rule | File | Line | Details |
|---|---|---|---|---|

## Regex patterns:

```
Found 334 issues
  /tmp/netductor-src/cmd/netductor/api_jobs.go:36 [ERROR-IGNORE] _ = primaryJobStore()
  /tmp/netductor-src/cmd/netductor/cli_edge.go:205 [ERROR-IGNORE] _ = sitesAttach(site, did)
  /tmp/netductor-src/cmd/netductor/cli_edge.go:282 [ERROR-IGNORE] _ = sitesAttach(args[2], args[1])
  /tmp/netductor-src/cmd/netductor/cli_secondary.go:206 [ERROR-IGNORE] _ = execLocal("ssh-keygen", "-f", "/root/.ssh/known_hosts", "-R", host)
  /tmp/netductor-src/cmd/netductor/cli_secondary.go:483 [ERROR-IGNORE] _ = execLocal("ssh-keygen", "-f", "/root/.ssh/known_hosts", "-R", host)
  /tmp/netductor-src/cmd/netductor/cli_secondary.go:530 [ERROR-IGNORE] _ = execLocal("systemctl", "try-restart", "sing-box")
  /tmp/netductor-src/cmd/netductor-agent/agent_ops.go:602 [EXEC-SHELL] _ = exec.Command("sh", "-c", "mountpoint -q /mnt/netductor || (block info 2>/dev/null; for p in /dev/sda1 /dev/sda /dev/sdb1; do [ -b $p ] && mount $p /mnt/netductor && break; done)").Run()
  /tmp/netductor-src/cmd/netductor-agent/agent_ops.go:660 [EXEC-SHELL] out, err := exec.Command("sh", "-c", script).CombinedOutput()
  /tmp/netductor-src/cmd/netductor-agent/build_role.go:183 [EXEC-SHELL] _ = exec.Command("bash", "-lc", "cd '"+workDir+"' && tar -czf dist.tgz -C dist . 2>/dev/null || tar -czf dist.tgz outbin 2>/dev/null || true").Run()
  /tmp/netductor-src/cmd/netductor-agent/build_role.go:185 [ERROR-IGNORE] _ = uploadArtifacts(client, cfg, out.Job.ID, distTGZ)
  /tmp/netductor-src/cmd/netductor-agent/build_role.go:224 [EXEC-SHELL] cmd = exec.Command("bash", "-lc", "cd '"+src+"' && go test ./... 2>&1; CGO_ENABLED=0 go build -o '"+workDir+"/outbin' ./... 2>&1 || CGO_ENABLED=0 go build -o '"+workDir+"/outbin' . 2>&1")
  /tmp/netductor-src/cmd/netductor-agent/facts_local.go:25 [EXEC-SHELL] if out, err := exec.Command("sh", "-c", "awk -F': ' '/^Serial/{print $2; exit}' /proc/cpuinfo").Output(); err == nil {
  /tmp/netductor-src/cmd/netductor-agent/facts_local.go:76 [EXEC-SHELL] if matches, _ := exec.Comm
```

