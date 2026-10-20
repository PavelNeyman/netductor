# Dead code verdicts (F5) — not a batch delete

Rechecked 2026-10-20 on tip after 0.9.429. A symbol is **delete** only if grep shows the definition and nothing else (no test, no doc-as-API).

## Delete (definition only)

| Symbol | Where | Note |
|--|--|--|
| `cmd/netductor.withSession` | serve helpers | Superseded by `apiSessionGate`. |
| `deploy.scpTo` | `sshutil.go` | No caller. Live path is `scpToRouter`. |
| `deploy.PrepareEdgeOffline` | wrapper | Callers use `PrepareEdgeOfflineOpts`. |
| `git.BuildProject`, `WaitVpsJob`, `ExtractAndBuild` | no callers | |
| `install.PrunePeerBackups` | duplicate | Live: `secondary.prunePeerArchives`. |
| `install.AllowAgentMTLSFromIP` | comment says do not call | |
| `nvr.EnsureStorageReady` | no caller | |
| `opcatalog.GroupLabel` | no caller | |
| `operator.FleetDeploy` | no caller | Live: `FleetDeployWithReport`. |
| `edge.Harden` | no caller | Doc `PLAN-EDGE-DEPLOY-HYBRID` is stale on this. |
| `edge.DeleteTemplate` | no caller | |
| `git.ReleaseVpsSlot` | no caller | |
| `metrics.LatestProbes` | no caller | |
| `nvr.LeaseRemainingSec` | no caller | |
| `opcatalog.ByID`, `Destructive` | no caller | |
| `vpn.ParseMismatchCount` | no caller | |
| `edgeagent.ShellApply`, `ParseUCIBatch`, `VLESSClientConfig` | no caller | |
| `deploy.LoadCachedTemplate` | no caller | |
| `dnsblock.DefaultBlockyDenyURLs` | no caller | |
| `svcpaths.WaitPing` | no caller | |
| `vpn.RevokeSubToken` | **not dead** | Wired in 0.9.428. |

## Keep

| Symbol | Why |
|--|--|
| `edge.ValidBearer` | Test-only, but tests use it. Delete only together with retargeting tests onto `DeviceIDFromAuth` (R-5). |
| `tg.formatVPNList` | Used by `helpers_test.go`. Production uses `formatVPNListPretty`. Delete with the test, or keep as test helper. |
| `cli18n.Register` | No production caller found; confirm it is not a plugin hook before delete. |
| `tapo.Client.GetDeviceInfo` | Alias. Harmless. Delete only if Tapo package is being edited. |
| `cmd/netductor-op` `menuItem` / `capture` / styles | TUI leftovers. Delete only inside a TUI cleanup commit, not alone — easy to break the menu. |
| Constants (`LabelLampac`, `FactUSBPresent`, …) | Duplication, not dead. R-6. |

## Do not delete in one commit

Take the **Delete** table one package at a time. After each: `go test` that package. Stop if a build tag or reflection use appears (none known).
