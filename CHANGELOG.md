## 0.9.549

- UFW leftover check matches exact source and port fields. A longer IP or a port with an extra digit no longer blocks remove.


## 0.9.548

- peer ready refuses if state changed during validate, including active. UFW leftover check is source+port, not any mention of the IP. RU runbook puts the traffic test before ready.


## 0.9.547

- peer ready re-checks state after validate and refuses removed/removing. UFW cleanup confirms sources are gone. Runbook puts the traffic test before ready/promote.


## 0.9.546

- UFW cleanup fails closed on status errors and still deletes rules when inactive. Secondary validate requires an exact PeerID. peer remove is retryable via removing.


## 0.9.545

- peer validate on Secondary uses local material.json. peer remove deletes UFW rules and peer material. Shadowrocket helper keeps both profiles on a shared hostname.


## 0.9.544

- CLI --primary-ssh-port uses ParseSSHPort. Web payload omits an empty port and is covered by ssh_port_test.js.


## 0.9.543

- Web omits primary_ssh_port when the field is empty. CLI errors if --primary-ssh-port has no value.


## 0.9.542

- One SSH port parser for TUI, Web, Fleet, and deploy. Empty is default; abc, 22abc, 0, -1, and 65536 are errors.


## 0.9.541

- Promote=true is rejected before any SSH. Primary SSH port and peer id are in TUI, Web, and Fleet. Invalid CLI port is an error.


## 0.9.540

- Deploy refuses Promote=true. Traffic test and peer promote stay separate operator steps.
- PrimarySSHPort and PeerID flow through SecondarySpec, fields, and CLI. /api/secondary/links skips empty SID.


## 0.9.539

- PeerAdd refuses a removed id. unitMissing requires a unit-specific message. Empty SID is omitted from /sub.
- DeploySecondary uses PrimarySSHPort for all primary SSH and rejects an operator binary older than pack min_agent_version before provision.


## 0.9.538

- PeerRemove no longer treats a D-Bus "No such file or directory" as a missing unit. Only a unit-specific systemd message is idempotent.


## 0.9.537

- PeerRemove treats a missing systemctl binary as a start failure, not a missing unit. Configs stay if systemctl never ran.


## 0.9.536

- PeerRemove stops units first and does not delete configs if stop fails. Missing units are idempotent. State stays until cleanup succeeds.
- Duplicate Secondary labels get a stable -2/-3 suffix.


## 0.9.535

- /sub/{token} lists every recently seen Secondary with a shared hostname and unique labels. Zero or stale LastSeen is omitted.
- PeerRemove fails closed: unit/file cleanup errors are returned and the peer is not marked removed.


## 0.9.534

- Registry/git bootstrap uses set -e so an earlier failure is not hidden by a later success. git init stays optional. Duplicate print removed.


## 0.9.533

- DeployPrimary fails on product conf write, set-sni, and fleet bootstrap. Doctor, telegram, domain, lampac, and registry stay explicit warnings.


## 0.9.532

- requirePublicIPv4 also rejects CGNAT 100.64/10, TEST-NET, 198.18/15, and 240/4.
- SSH key passphrase is not TrimSpace-d before parse.


## 0.9.531

- Provision parses an encrypted SSH key with its passphrase. Resume is key-only (password cleared). Initial rejects a separate secondary key.
- TUI precheck uses the normalized SSHMode. Non-public IPv4 is rejected before SSH.


## 0.9.530

- Resume precheck sends the primary key passphrase only when the dial key is the primary key.
- SecondaryFromFields leaves ssh_mode empty when both password and key are set without an explicit mode, so ResolveSecondarySSH can reject the ambiguity.


## 0.9.529

- All SSH/SCP to the secondary host use secondaryDialKey (SecondarySSHKey, else PrimaryKey). Primary connections stay on PrimaryKey.
- TUI resume precheck sends the secondary key, not the root password. Log line is mode-specific.


## 0.9.528

- Secondary SSH mode: initial = :22 + root password; resume = :52222 + key. Empty mode infers from which credential is set. Both without mode is an error.
- Resume does not fall back to port 22. TUI and Web have an explicit mode field. Final note prints the slot PS WSS port.


## 0.9.527

- LoadMaterialForHost accepts primary-complete or ForSecondary material. Probe, health timer, and tunnel-addr reads use it.
- LoadMaterialStrict still requires all four primary privates for regen. Secondary file is not treated as corrupt.


## 0.9.526

- Slot 0 is legacy only when addrs, ifaces, and ports are all empty. A lone PortSP is incomplete, not slot-0 defaults.
- ProbeHealth uses slot 0 only when material.json is absent. Corrupt or identity-fail writes material-invalid and returns error.


## 0.9.525

- resolveBootstrapParams validates identity and fills missing iface/ports from SlotTable[slot] before any bootstrap side effect. No silent slot-0 fallback for other slots.
- Mismatched ports fail closed. Peer validate and ProbeHealth use the same resolver.


## 0.9.524

- BootstrapPrimary/Secondary validate tunnel identity before deps, file writes, firewall, or unit restarts.
- healthTimerProbeAddrs is the fail-closed addr picker; corrupt material never installs the timer with slot 0.


## 0.9.519

- P2: DeploySecondary ships normalized peer material (slot 0 SlotTable defaults included). WireGuard keys from the original JSON are preserved.


## 0.9.518

- UFW residual open-port detect: match `PORT/tcp (v6) ALLOW IN Anywhere (v6)` (IPv6-only open rules).
- Shell UFW checks share the same pattern as Go `ufwStatusHasOpenPort`.
- DeploySecondary: parse PortSP/PortPS; validate all four ports/WSS (+ iface/addrs) against SlotTable before SCP.
- Slot 0 defaults include PortSP/PS and WSS_PS from SlotTable.

## 0.9.517

- assertPeerMaterialIdentity: PortSP/PS and WSS_SP/PS must match SlotTable.
- UFW residual check matches "ALLOW IN Anywhere" (and without IN).
- DeploySecondary: require WSS_PS for slots 1–3.

## 0.9.516

- assertPeerMaterialIdentity: registry slot match; required secondary/ifaces/addrs.
- DeploySecondary: no slot-0 defaults for slots 1–3.
- UFW: --force delete open WSS; fail if Anywhere remains when ufw active.

## 0.9.515

- PeerAdd: assert material PeerID/IP/slot; refuse IP rebound to different ID.
- DeploySecondary: refuse SCP if mat.PeerID != peerID or slot out of range.
- BootstrapPrimary/Secondary: WSS UFW from peer public IP only; drop open allow PORT/tcp.

## 0.9.514

- LoadMaterialStrict reads keysRoot/material.json (testable with SetKeysRootForTest).

- PeerValidateLocal/recover: require material.PeerID == peerID (empty rejected).
- Legacy migrate writes PeerID into peers/<id>/material.json.

## 0.9.513

- PeerValidateLocal: when peerID set, load peers/<id>/material only (never legacy first).
- Regression: validate peer-b uses slot1 ifaces; missing peer fails.

## 0.9.512

- Promote=true: peer validate → ready (no --force-ready) → promote.
- UFW WSS_SP allow from secondary public IP only (not 0.0.0.0/0).

## 0.9.511

- DeploySecondary: ufw 8789 from secondary SP/PS WG IPs + WSS_SP; fail if ufw active and rules fail.
- Primary: unit checks + ping secondary WG endpoints (with B→P pings).
- Promote=true runs peer ready --force-ready + peer promote (not raw preferred write).

## 0.9.510

- DeploySecondary: svc-paths peer add (slot-aware) instead of legacy bootstrap-primary alone.
- Secondary/primary verify uses material iface/addrs/WSS, not hardcoded slot 0.
- logPeerJournal: neutral message (not only after commit).

## 0.9.509

- Document peer-ops.log: diagnostic audit only; recovery uses op.json/peers.json.
- logPeerJournal: plane success + journal fail → stderr warning, no rollback.

## 0.9.508

- PreferredFailRestoresExisting: assert modes 0644/0644/0600 on all three files.
- TestPeerPromoteAndRollbackBothFail: promote+rollback dual write failure joins both errors.

## 0.9.507

- Remove Disabled_* promote tests; restore preferred-fail-with-existing coverage.
- ActiveWriteFail rollback asserts all three files + secrets mode 0600.

## 0.9.506

- snapshotPromote: only IsNotExist = missing; other read/dir errors abort before mutate.
- Capture file modes; restore with original perms; injectable writeFileFn for tests.
- Promote fault tests: unreadable path pre-check; active write fail + full rollback; secrets fail clears new preferred.

## 0.9.505

- PeerPromote: full registry+files snapshot; rollback restores or deletes active/preferred when absent pre-op.
- Fix TestRecoverIndexFailedJournalErrorJoined operator precedence.
- Tests: active_peer_id write fail; preferred cleared when none before.

## 0.9.504

- Recovery/PeerAdd: join checkpoint errors with bootstrap/index failures (visible).
- validatePeerOp: slot must exist in SlotTable; strict secondary/primary IP.
- No silent slot-0 fallback on unknown material slot.
- rollbackPromote returns joined errors; promote surfaces rollback failures.
- Tests: invalid slot, index_failed+journal fail, preferred fail rollback, secrets path.

## 0.9.503

- PeerAdd: journal write failures abort before keygen/bootstrap; checkpoint after bootstrap required.
- Recover: phase bootstrap/index_failed skips re-bootstrap; validates op integrity.
- Preferred paths injectable; promote success tested hermetically.
- Fault tests: journal fail, index_save fail + second recover, corrupt op, unknown phase.

## 0.9.502

- Fault-injection tests: recover after material/bootstrap_failed; no key regen; promote requires ready; slot/WSS isolation.
- keysRoot overridable in tests; bootstrapPrimaryFn injectable.

## 0.9.501

- API: validate returns HTTP 409 before JSON body when checks fail.
- API: promote/remove/rollback report inventory_ok / inventory_error (no silent ignore).
- Docs: R5 runbook is acceptance procedure, not production certificate.

## 0.9.500

- R5 close-out: Blue-Green runbook + acceptance checklist (docs/BLUE-GREEN-SECONDARY.md).
- scripts/bg-secondary-preflight.sh on Primary; OPEN_ITEMS BG status; help lists inventory.

## 0.9.499

- R4 UI: Web Secondary panel inventory (list/register candidates).
- Telegram: Fleet → Secondaries (list/register); no deploy from bot.
- CLI: netductor inventory list|add.

## 0.9.498

- R2: peer add op.json phases + `peer recover`; full journal.
- R3: inventory.Register/List (planned candidates, no network); Primary API `/api/secondaries`, `/api/svc-paths/peers`.
- Operator: GET/POST `/v1/secondaries` (local inventory, register ≠ deploy).
- Tests: inventory register, peer id/IP, unique WSS_SP slots.

## 0.9.497

- R1 Blue-Green blockers: PeerValidateLocal; ready requires validate; promote only from ready.
- Per-slot SP WSS server units + unique WSS_SP ports (B cannot rewrite A restrict-to).
- PeerAdd: no legacy material.json overwrite; no DefaultPolicy reset on 2nd+ peer; migrate errors abort.
- PeerPromote: fail-closed preferred/active writes with rollback; peer-ops journal.
- PeerRemove: disable --now, remove conf/units, then free slot.
- BootstrapPrimaryWith explicit side-effect opts.

## 0.9.496

- Blue-Green: service peer registry, slots SP/PS, peer add|ready|promote|rollback|remove.
- Material: corrupt/incomplete never regenerates keys.
- Deploy: preferred_secondary_ip only when Promote=true.
- hostRole shared; sing-box apply restores previous config on failed restart.
- Per-slot WG ifaces and PS wstunnel client units on primary.

## 0.9.495

- svc-paths: reuse WG material on same secondary IP; refuse different peer without --force-new-material (one HA secondary on fixed /30).
- Failover: agent owns ticks when active; timer is fallback; min 20s between counted samples.
- applyUserUplink: atomic config write; systemctl restart sing-box fail-closed + is-active.
- Probe role: secondary_core_url / material.json when WG ifaces down.
- Harden iptables fallback: NETDUCTOR chain ends with DROP.

## 0.9.494

- Failover: user_path ps → uplink 10.87.11.1 (PS); sp → 10.87.10.1; vless → public primary.
- VLESS health: local :443 on secondary (not SP ping).
- Failover cycle: exclusive flock (agent + timer); StableSeconds gates failback; saveState/desired errors returned.
- RemoteJoin: rollback binary on join or post-join health fail; prev binary outside trap dir.
- schema_version required (no 0→1); ValidDotVersion exactly major.minor.patch.

## 0.9.493

- R6/R16: host-key mismatch aborts deploy (no auto ForgetSSHHost/ClearHostKeys/retry); ForceReinstall / force_reinstall only after offline verify.
- Health timer: always rewrite script; failover tick failures exit non-zero and log (no silent success).
- R10: effective PasswordAuthentication via sshd -T must be no (fail-closed).
- prepare-pack: validate top-level schema_version; chmod 600 material fail-closed; drop primary ufw 8445.
- RemoteJoin: arch amd64/arm64, binary rollback on join fail, rm /root/bundle.json on EXIT.
- version.ValidDotVersion + stricter Less; reject invalid min_agent_version.

## 0.9.492

- Docs: SECONDARY-DEPLOY (EN/RU) — SSH TOFU policy (R6/R16), deploy end-state, R22 manual failover checklist, R15 bundle contract; links from SECURITY, DEPLOY-MAC, docs index.

## 0.9.491

- R15: SecondaryBundle schema Version + MinAgentVersion; join rejects unsupported schema / agent too old; prepare-pack emits schema_version/min_agent_version.
- R12: extractJSONObject fail-closed (no legacy from-{ to-EOF fallback); prepare-pack validates agent_id/token/bundle.
- wstunnel: pin v10.1.9, verify official SHA256 (linux amd64/arm64) before install.
- R10: after key-fail on :22, probe day2 for listen:22 and PasswordAuthentication=no.
- version.Less exported for SemVer comparisons.

## 0.9.490

- R6: password deploy no longer ForgetSSHHost by default; Go TOFU mismatch fails; wipe only ForceReinstall or explicit forget + one retry after mismatch.
- Material on secondary: trap EXIT rm /tmp/nd-svc-material.json; chmod 600 before bootstrap.
- R3 residual: installHealthTimer daemon-reload/enable/--now + is-active fail-closed; try-restart secondary-agent fail-closed (absent unit soft).
- Deploy verify: health timer active on P and S; wait online requires seen= within 90s (fresh heartbeat after SP).
- secondary status: append seen=<unix> for deploy freshness checks.

## 0.9.489

- R6: password secondary deploy no longer always ClearHostKeys; only on host-key mismatch + one retry.
- R11: svc-paths material temp file Write/Close errors are fatal.
- R12: prepare-pack / material JSON via extractJSONObject (Decoder), not first `{` to EOF.
- R10: :22 post-harden probe logs refused vs auth-denied vs other (still fail only if key login succeeds).
- R1 residual: post-join :443 check aligned with RemoteJoin (`:443([^0-9]|$)`).

## 0.9.488

- R2: Material.ForSecondary strips primary WG private keys (SPPrivP/PSPrivP); bootstrap-primary stdout and BootstrapSecondary persist only the secondary-safe copy.
- R3: BootstrapPrimary/BootstrapSecondary hard-fail on conf write, unit write, and critical systemctl enable/restart (no silent `_ =`).
- P2: wait online + preferred_secondary_ip use awk exact field match (`$1 == AgentID`); preferred writes only when value is IPv4 (no hostname fallback into preferred_secondary_ip file).

## 0.9.487

- R7/R8: SP/PS verify pings peer only (S→10.87.10.1+10.87.11.1, P→10.87.10.2+10.87.11.2). No self-ping fallback.
- R5: wait online for prepare-pack AgentID, not any secondary. preferred_secondary_ip taken from that row PublicIP when it is an IP.
- R1: RemoteJoin :443 check uses :443([^0-9]|$) instead of broken \\b.


## 0.9.477

- NVR: MaxCameraGB + per-camera/global quota before ingest; agent refuses record when camera dir full
- Guest desk: staff-only (exclude guest SubnetCIDR); DeskAllowCIDR / NETDUCTOR_GUEST_DESK_CIDR
- Recovery UI: HTTPS with ephemeral self-signed (or NETDUCTOR_RECOVERY_TLS_CERT/KEY); HTTP only if NETDUCTOR_RECOVERY_HTTP=1

## 0.9.476

- recovery form: Primary URL must be https (lab: NETDUCTOR_RECOVERY_ALLOW_HTTP=1)
- guest desk: no default PIN 0000; rate-limit PIN fails (5 / 15m / IP)
- agent mTLS: fail-closed when material present but unreadable (no silent bearer fallback)
- agent NVR: ValidCameraID + rtsp(s) only + segment 10..3600; path under nvr root
- NVR ingest: hard 64 MiB LimitReader per segment

## 0.9.475

- BackupSecondaryLocal: defer remove plaintext; wipe partial .ndenc on encrypt fail
- install Run: lampac / git / registry soft (like telegram) — do not block recover drill

## 0.9.474

- deploy SSH: stop clearing known_hosts on every hop (TOFU restored; ClearHostKeys only for explicit reinstall)
- RemoteJoin: download binary into mktemp -d 0700, not fixed /tmp/netductor.new
- BackupSecondaryLocal: require backup_key (no silent plaintext .tar.gz)
- recover: pre/post install errors + agent-allow sync feed critical; drill only if all pass
- NDG2: strict trailing-byte check; tests for reorder + iptables policy drift

## 0.9.473

- NDG2: per-chunk AAD (index) + authenticated end marker (count+size); reject truncate/forge/reorder
- recover: drill only if sing-box, blocky, netductor-api are active after restart
- baseline iptables: chain policies (`:INPUT DROP`) included in diff
- ufw mTLS trust: exact SvcSP/PS CIDR match (not substring 10.87)
- OpenWrt facts: FLASH_KB via ash/printf (no gawk strtonum)

## 0.9.472

- firewall Collect: `ufw status numbered` — first-match WAN rule order (early ALLOW 22 fails even if DENY exists later)
- NDG2 decrypt: end marker mandatory; legacy streams without marker rejected (re-backup)

## 0.9.471

Security close-out from review of 0.9.468–0.9.470:

- P1.1: restoreHostnameFromBackup — no shell; strict hostname charset; /etc/hosts via Go
- P1.2: ufw policy checks TCP proto; mTLS requires restricted ALLOW; iptables fallback never OK
- P1.3: Recover fails + no drill-mark if vpn apply or agent firewall fails
- P2.1: sing-box pin 1.11.15 + mandatory SHA for install/cache
- P2.2/P2.3: baseline Default: policy + iptables-save diff in doctor
- P2.4/P2.7: offsite require SHA256+Size headers; 512MiB download cap
- P2.5: NDG2 end marker + reject truncated nonce
- P2.6: backup_pull:<cmd_id> + WaitForBackupPull matches cmd_id in log
- P2.8: recovery UFW allow only after CIDR validation (disarm rolls back)

## 0.9.470

- host baseline: full ufw rule-set diff vs `baseline/ufw.txt` (new + missing ALLOW/DENY) without ufw reset
- doctor: WARN `snapshot new ufw rules` / `snapshot missing ufw rules`; OK only when ports+units+ufw match

## 0.9.469

- offsite: primary sends X-Netductor-Backup-SHA256/Size; secondary verifies hash+size before promote; sidecar `.sha256`; log includes `sha=`
- sing-box: embedded SHA-256 pins for v1.11.7 (amd64/arm64/armv7/mipsle/riscv64); cache + install enforce pin (env override still works)
- firewall Collect: line-based ufw policy (22 vs 52222); DENY/ALLOW and mTLS restrict checks without ufw reset

## 0.9.468

- firewall iptables fallback: critical rules return error (no false success)
- offsite: backup_pull requires BACKUP_KEY; WaitForBackupPull needs this-run ack + key=ok in log
- secondaryUpgrade: download into private MkdirTemp (no /tmp/asset.new race)
- DNSBL dig: SERVFAIL/empty → unknown; only NXDOMAIN/NOERROR clean
- api-public: fail + ufw rollback if disarm timer not active
- edge vpn: init scp/enable errors when VLESS deferred
- backup verify: private temp dir
- sing-box cache: unique CreateTemp publish
- recovery upload: 512MiB MaxBytesReader, temp+rename, ReadTimeout

## 0.9.462

OpenWrt deploy (review):
- **Dry-run** probe is read-only (`bootstrapUSB=false`); real deploy may install USB kmods
- **sing-box path:** prefer USB mounts; internal only if **overlay** free ≥ need (not max root/overlay)
- **opkg USB packages:** fail-closed; require usb_storage/sd_mod after install
- **kmod dir:** no first-dir fallback when kernel is known
- **VPN init/chmod:** errors are fatal
- **extroot → skip VPN:** RESULT PARTIAL + error to re-run after reboot
- docs/OPENWRT-LAB.md dry-run contract

Deferred: min sing-box build experiment, full phase resume machine, WANReachable probe.

## 0.9.461

Review closure (ordered):
- **TG devices:** one visible slice + pageSize 12 + `m:vpn_devices:p:N`; long names → `m:dev:i:page:idx`
- **Backup restore:** `filepath.Base` + path under backups dir; list names Base’d
- **Callback tests:** 64-byte helpers, devices visible slice, backup traversal
- **Web:** single `bindSidebarNav` (desktop collapse / mobile drawer); no capture dual-handler
- **i18n:** users/devices th_*; channel Secondary/Path e2e headers via T()
- **TUI:** handleMouse prefers View() hit-rects (tab/lang/row)
- **Docs:** TG-UI-PATTERN hub in-body actions exception (EN/RU)

## 0.9.460

- **TG:** Registry + mTLS certs screens via T() (RU/EN)
- **TUI:** catalog viewport + PgUp/PgDn/Home/End + wheel; Output viewport + scroll/wheel
- **Web:** mobile sidebar toggle (btnSideOpen) + backdrop; close on nav

## 0.9.459

UI residual (no GitHub assets yet — code on main):
- **TUI:** termW without forced 40 cols; list viewport follows cursor; narrow single-column; mouse header matches update badge + mode
- **Web:** 100dvh; site/srv-grid minmax(min(100%,…)); mobile sidebar overlay + dock rules
- **TG:** table headers via T(th_*); NVR Live on EN hub; RU guest/dns/backup labels

## 0.9.458

- **Recovery auto-pin:** secondary heartbeat sends `recovery_fp`; primary stores `/etc/netductor/secrets/recovery_peer.fp` (+ per-device under state)
- **Ship cancel:** `netductor project cancel <id>` — state cancelled + best-effort `docker kill` of CI container (`ship-<id>`)
- Pause still waits end of step; cancel is explicit stop path
- **CA rollover:** server cert backup errors checked; rollback errors reported
- **NVR ingest:** EvalSymlinks fail-closed after MkdirAll

## 0.9.457

- **N-19** large backups: chunked AES-GCM (NDG2); legacy CBC decrypt kept for old archives
- **N-18** recovery TLS: CA pin or **fingerprint** required; `INSECURE=1` only explicit; secondary writes `recovery.fp`
- **N-30** recover stream to disk (512 MiB cap), no full-archive RAM
- **N-23** CI secrets via gha jobEnv, not process `Setenv`
- **N-24** ship publish fails on copy/tar/SHA errors
- **N-27** no-tests / soft `|| true` removed — tests must pass
- **Ship lifecycle**: Pause still waits end of step; Timeout waits grace before failed (not Pause)
- **T-5** node rename callbacks by list index

## 0.9.456

Security / reliability residual after tip reviews:
- **N-26** build-host heartbeat: cmds from token host only (not body device_id)
- **N-25** /result + CompleteAgentBuild: ownership + waiting_external gate
- **N-20/N-32** backup: refuse plaintext without key; tar exit checked
- **N-21** download: CreateTemp instead of dest+.new
- **N-31** guest store: fail-closed on corrupt JSON
- **N-22** HTTP body limits per upload path (NVR 64M, backup/artifacts 512M)
- **UI:** Web catalog single source (by_group); Doctor not wiped by tables; TG routers parent; backup button-row markup; sites hub no double list

Still open (next): AES-CBC→AEAD for large backups, recovery pin-required, ship secrets env, CI || true, TUI narrow layout, TG callback length.

## 0.9.455

- **P0** edge: one-shot TokenPickup after Approve — agent Enroll receives device token once; further bootstrap enrolls empty (H-1 without breaking onboarding)
- EnqueueCmd requires StatusApproved only (no empty-status enqueue)
- secondary pullBackup: Base name, .ndenc, temp+rename (no path traversal / no O_TRUNC wipe)
- agent backup download: clear WriteDeadline for large files
- Ship: no main/HEAD fallback when release tag missing
- release.sh: set -euo pipefail before preflight checks
- recovery: invalid ALLOW_CIDR fails arm; TLS failure does not fall back to HTTP unless TLS=0

## 0.9.454

- firewall: critical primary/secondary ufw rules use must(); Collect verifies mtls/878x role policy (H-2 residual)
- mtls CARolloverFinish: backup server cert, rollback CA+server on ensureServer failure; rollover state save error surfaced
- IssueClientFromNewCA: write .new then rename; revoke old serial only after successful promote

## 0.9.453

- edge: fail-closed on devices load (RegisterPending/Import/Export/SetSiteID); command queue does not wipe/reissue on corrupt IO
- enroll: recovery code must Consume successfully before Enroll (not bootstrap)
- NVR ingest: reject camera dir that resolves outside root via symlink
- statefile.SaveJSON: CreateTemp + 0600 + rename (no predictable .tmp name)
- vpn loadRegistry: migrate persist and EnsureDirs errors returned
- Ship: checkout release tag before HEAD/main; empty worktree is error
- operator Origin: parse URL, exact loopback host only
- release: agent arch builds and check-web-ui no longer `|| true`

## 0.9.452

- migrate-on-read for stores touched by product changes: VPN users (kind, sub_profile, policy defaults), edge devices, sites, custom policy presets, sub tokens, recovery list
- guests / locator devices / policy-presets → statefile (fail-closed load)
- host-baseline repair watches vpn_guests, vpn-devices, policy-presets

## 0.9.451

- statefile migration remainder: mtls pending/rollover, vpn-users registry, edge commands queue, notify alerts/known_chats, integrity manifest, reboot challenge load
- host-baseline repair scans expanded state path list

## 0.9.450

- LE: certbot pre/post/deploy hooks (free :80 on renew, restore redirect); doctor checks hooks, :80 listeners, cert expiry
- host-baseline apply/repair + stack baseline: RepairStateOnUpgrade (hooks + state JSON mode checks)
- internal/statefile: LoadJSON/SaveJSON; edge devices/recovery, vpn subtoken, sites, jobs, reboot-challenge migrated

## 0.9.449

- Enroll: check saveDevices errors (no silent success on disk failure)
- cli secondary: mtls.EnsureAll failure is fatal
- agent guest/build recovery HTTP servers: ReadHeaderTimeout set

## 0.9.448

- Security audit remainder:
  - R-10: jobs persist atomic + Start fails if cannot write; terminal status checks persist
  - vpn loadSub/saveSub: NotExist vs error; atomic save; N-12 refuse prune with empty live + tokens
  - sites GetStrict distinguishes missing vs store error; RSC respects load errors
  - CARolloverFinish: ordered backup/promote with rollback on rename failure
  - RotateClientForPush: load/save pending errors returned

## 0.9.447

- Security series 1 remainder (unified audit on tip 0.9.444 findings still open after 0.9.445–446):
  - R-1: loadDevices distinguishes missing vs corrupt; Enroll/Approve fail closed
  - R-2: BootstrapToken no longer falls back to legacy edge_token
  - R-6: DeviceIDFromAuth requires approved status
  - R-7: ConsumeRecoveryCode only succeeds after saveRecovery
  - R-3: serve exits if mtls.EnsureAll fails
  - R-9: /api/edge/commands method check

## 0.9.446

- Security remainder ([docs/SECURITY-REVIEW-2026-10-20.md](docs/SECURITY-REVIEW-2026-10-20.md)):
  - M-1: NVR camera_id validation + path containment
  - M-4: local release install/import requires SHA256SUMS
  - N-9: redirect + agent-plane HTTP timeouts
  - M-2: operator serve Host/Origin checks
  - M-6: ship checkout prefers refs/tags; SHIP_FROM_TAG / Ref=tag

## 0.9.445

- Security P0/P1 ([docs/SECURITY-REVIEW-2026-10-20.md](docs/SECURITY-REVIEW-2026-10-20.md)): H-1 enroll token, H-3 mTLS revoke fail-closed, H-2 firewall, N-10 OpenWrt pkgs, M-5 partial CA, M-7 tmp write, N-8 sub-token rename, M-3 download redirects, L-3 PEM nil-check

## 0.9.444

- edge: offline USB cache includes librt + libpthread (libuuid1 deps)

## 0.9.443

- edge: USB package list from operator.conf + Web textarea (A+B); defaults still built-in

## 0.9.442

- edge: OpenWrt USB cache uses libcomerr0 (not libcomerr2) + alternates

## 0.9.441

- edge: offline apk USB install (no WAN index fetch) + cache e2fsprogs deps (libuuid/libe2p/…)

## 0.9.440

- Docs D-1: RU pairs for surviving EN-only ops docs; OPEN_ITEMS/handoff stop line
- CLI: `netductor vpn sub-prune` [--dry-run] — drop orphan subscription tokens
- doctor (primary): WARN when orphan sub tokens remain

## 0.9.439

- N-1: recoveryHTTPClient returns error; both callers handle bad PEM
- N-2: incident.Collect always execs /usr/local/bin/netductor with timeout; tg exits 2 on unknown args
- N-3: /api/nvr/live clears WriteDeadline
- N-4: RenameSubToken; RevokeSubToken errors returned; disable still kills URL (documented)
- N-5: firewall.truncate trims before format.Truncate
- secreq.Eq on build-host and NVR tokens
- dropped no-caller dead funcs; restored PLAN-TG-MENU; dropped no-op lampac excludes


## 0.9.438

- R-1 step 6: DeployEdge moved to edge_run.go
- R-2: format.HTMLEsc / Truncate; agent and firewall use them. notify htmlEsc stays (cycle)


## 0.9.437

- R-1 step 5: session, git, and NVR API split into route-group files


## 0.9.436

- R-1 step 4: runDoctorNative setup only; checks in doctorRunChecks


## 0.9.435

- R-1 step 3: handleGitCB split into hub, build-host, ship, projects, repo


## 0.9.434

- R-1 step 2: TG handleCallback split into edge/tools/hubs/users/nodes. Behavior unchanged.


## 0.9.433

- R-1 step 1: edge API split into meta/templates/enroll/storage/agent/admin
- R-2: ShellQuote in internal/format; deploy and svcpaths use it
- F8: ufw delete via argv, not bash -c
- F7: httpx.NewServer on node serve (60s write). Op stream stays 2h


## 0.9.432

- Backup excludes build-backup-mirror (was 3.4G), stack binaries, lampac/ts
- R-2/S-4: internal/secreq.Eq; edge constEq and sub token compare use it


## 0.9.431

- F4: gofmt sweep (separate commit e3d8cd21)
- Backup excludes ci-cache, releases, agents, docker, lampac cache/data
- TG status/timeline: classic HTML unless the body needs rich controls
- Removed production-dead helpers (scpTo, PrepareEdgeOffline, PrunePeerBackups, edge.Harden, WaitVpsJob, and others with no test callers)
- Deleted stale plans: PLAN-REMOVE-TG-TOPICS, PLAN-TG-MENU, PLAN-EDGE-DEPLOY-HYBRID, YOUTUBE-ADS


## 0.9.430

- S-6: WireGuard private key to `wg pubkey` via stdin, not argv
- S-3: DR doc notes BACKUP_KEY.txt beside archives on secondary (accepted)
- Plans: PLAN-REFACTOR-R1-R8, PLAN-DEADCODE-F5, PLAN-DOCS-D1


## 0.9.429

- selfbin.CLI: never exec a go test binary (fork-bomb guard on incident, smoke, domain LE, op serve, profiles, backup-verify)


## 0.9.428

- S-1: revoke VPN sub-token on delete/disable; /sub/ rejects disabled users
- S-2: recovery pin present but not PEM fails closed
- S-8: Approve propagates saveDevices error
- Q-1: Web i18n keys edge_st_g_bak, rd_mode, rd_pos, result_hint
- Q-2: drop self-assignment in projects.go
- incident Collect: do not exec *.test (fork-bomb under go test)


## Unreleased (docs only, no version bump)

- Docs: full-complex static review report + plan (EN+RU): `docs/REVIEW-REPORT-2026-10-19.md`, `docs/REVIEW-PLAN-2026-10-19.md`
- Docs: AGENT_HANDOFF (EN+RU) restructured — current stop line v0.9.427 on top, history archived; OPEN_ITEMS follow-ups F1–F10; AGENTS.md baseline/Project State aligned to 0.9.427
- Code unchanged; automated tests were not run for this review

## 0.9.427

- Offline-prep: if SSH to primary :52222 fails, reuse cached edge_bootstrap_token instead of hard-fail
- Clearer hint when connection is closed (fail2ban / key / passphrase)


## 0.9.426

- OpenWrt USB packages: offline-first cache (no network on probe if offline-prep already filled cache; avoids downloads.openwrt.org EOF)
- Web Result dock: **Log** tab — job/deploy stdout streams there (option A); bottom Deploy log still mirrors
- Edge probe card: normalize job result `{facts, plan}` so DeviceCard renders again (not mixed text)

## 0.9.425

- Result dock: rubber layout; right mode = only ⋮ menu pinned top-right
- Position via select (no arrow icons); JSON no longer prints null
- op-cards / tables: overflow ellipsis, adaptive grid

## 0.9.424

- Result dock: icon toolbar (no truncated RU labels); position ⬇/➡/⬆ always works including right→top
- Sensible size when switching orientation; tooltips RU/EN

## 0.9.423

- Operate: map BTN sections to shell panels (overview to home); missing button wiring fixed
- Result dock: compact menu in right position; Raw no longer overflows
- Inline preview under clicked action only; human summary not raw JSON
- Unwrap nodeFetch envelope for tables/doctor; clear 401 session hint
- Stronger busy/ok/err feedback on op-link and primary

## 0.9.422

- Operate Result dock (VS Code–style): positions bottom/right/top, show/hide, drag-resize
- Modes: panel only / card inline only / both; History tab (local, last 40)
- Toggle: topbar icon + statusbar «result»; prefs in localStorage + operator.conf (`result_*`)
- Action busy/ok/err feedback on op-link clicks

## 0.9.421

- Web calm UI: no electric-blue primary fills; slate accent `#5a7a9a`
- Operate: actions grouped into op-cards (diagnostics / stack / updates / …) instead of button wall
- Edge USB day-2: same card layout
- Servers: control-plane topology (SP/PS/mTLS, uplink, PB/BP) on overview + per-card + detail

## 0.9.414

- Web UI skins A/B/C (Settings → UI skin; localStorage `nd_op_skin`); shared forms/jobs
- Web deploy: jobs-only path (no HTTP stream fallback for edge/fleet/primary/secondary/build-host)
- Header Jobs badge + running count; Deploy/Operate/Settings labels
- Interactive mockups at `/mockups` (A/B/TUI-structure)
- Deferred: Web terminal + file manager (OPEN_ITEMS)
- Docs: WEB-UI-REDESIGN status; OPEN_ITEMS deferred notes

## 0.9.413

- Thin-UI jobs: Phase 3 checklist audited (deploy=job, day-2=EnqueueCmd, fast=sync)
- Prometheus: netductor_jobs_running / netductor_jobs_failed_total
- OPEN_ITEMS stop line → 0.9.413


## 0.9.412

- SSH cancel: `CommandContext` + `deploy.WithSSHContext` on job runners (real kill of in-flight ssh/scp)
- TG: Doctor → `job start doctor.run`; Tools Jobs unchanged
- PLAN-THIN-UI-JOBS marked Phases 0–4 core done

## 0.9.411

- **All UIs:** job list + cancel
  - Web op: panel Operator jobs (refresh / running / cancel)
  - TUI: Jobs running/all + Jobs cancel form
  - CLI op: job list|cancel (already); node: `netductor job list|cancel`
  - TG Tools → Jobs (primary store) + cancel buttons for running
- Note: Mac edge/fleet jobs only on op serve; primary jobs (stack/doctor) on node

## 0.9.410

- CLI deploy (primary/secondary/fleet/edge) via **/v1/jobs**
- TUI menu: Jobs (running/all) against op serve
- Job runners: cooperative cancel via ctx (marks cancelled; SSH may finish current step)

## 0.9.409

- TUI wizards (primary/fleet/secondary/build/edge) via **/v1/jobs** + auto-start `serve`
- Web: jobMap includes primary/secondary/build-host; reattach running jobs from sessionStorage
- Dead in-process Deploy* on those TUI paths removed (runners on serve only)

## 0.9.408

- Thin UI jobs: internal/jobs; op /v1/jobs (edge.probe, edge.provision, edge.offline_prep, fleet, primary/secondary/buildhost); primary /api/jobs (stack.apply, doctor.run)
- Web: slow edge/fleet paths via job poll (sessionStorage job_id)
- CLI: netductor-op job list|get|wait|cancel

## 0.9.407

- Edge probe: fix BusyBox exit 127 aborting USB package install

## 0.9.406

- Edge probe: install OpenWrt USB mass-storage packages **before** collecting facts so card shows USB / Format / overlay

## 0.9.405

- Edge: install OpenWrt USB mass-storage packages from Mac cache (kmod-usb-storage, scsi, ext4, block-mount, e2fsprogs) via apk/opkg; offline-prep pre-seeds ramips/mt76x8
- Router needs no WAN; Mac downloads matching release/target/kernel packages

## 0.9.404

- Hygiene series after 0.9.403: dead-code layers 1–6, POST-REVIEW crumb removal, fleet sync timer cleanup
- DNS: custom list URL via API/CLI/TG (`dns add`, `/api/dns/custom`, TG ➕)
- Recovery HTTPS client: optional CA pin (`NETDUCTOR_RECOVERY_CA` or `/etc/netductor/secrets/recovery_peer.crt`); skip-verify only if unset
- R1 secondary builders: already closed in tree (WriteSecondarySingBox orchestrates only)
- opcatalog: dns-custom action

## 0.9.402

- Edge vpn_client: pre-check free space; install sing-box to USB data mount when internal flash too small; hard-fail (no fake success/reboot) if nowhere to put binary
- Prefer /usr/sbin when space allows; else /mnt/.../netductor/sing-box or /opt/netductor/sing-box + symlink

## 0.9.401

- TG: /start updates deletes user command + previous hub before opening Versions (no stacked menus)

## 0.9.400

- TG: /start updates and Tools→Updates open multi-component Versions hub (not legacy primary-only picker)

## 0.9.399

- EnsureAgentBinary: checksum mismatch on cache → re-fetch (not hard fail)

## 0.9.398

- Offline-prep: partial status when some agent/sing-box arches fail; UI warns and Retry downloads **only gaps** (cache kept)
- Manifest: missing_agents, partial, status; CLI exit 2 on partial;  full re-fetch

## 0.9.397

- TG: channel alert buttons open bot DM via deep link (`t.me/bot?start=updates`), not operator menu in the channel
- TG: hard-block callback menus when chat type is channel
- /start updates opens Updates screen in private chat

## 0.9.396

- Edge deploy: disable LuCI attended-sysupgrade check on Status (no popup loop)

## 0.9.395

- Edge vpn_client: sing-box + VLESS (from primary) at first boot when possible
- TG update alerts: inline buttons work in alerts channel
- ClearAlertPrefix for update:available:*

## 0.9.393

- Edge: guest nft no longer written to /etc/nftables.d (fw4 include broke firewall with "unexpected table")


## 0.9.374

- Build-host agent routes on :8789 (heartbeat/claim/work/result)
- Job routes require Bearer + host ownership; claim binds host to token
- BuildHostEnabled = online agent only (stale registry does not hard-fail)
- Online TTL unified to 3 min; build-host test is status probe; remove <id>


## 0.9.371

- Fleet: optional **build-host** (do_build) after primary/secondary
- TUI/Web/API: do_build + build_host/password/label; DeployBuildHost in FleetDeployWithReport

## 0.9.370

- Web i18n: clean Build-host tab labels (EN/RU), build_note RU
- TG Build-host card via T() (title/none/probe/online)
- PLAN-BUILD-HOST status through 0.9.369

## 0.9.369

- recover --from-build-host (same API as secondary recovery)
- Build agent: recovery_arm → :8790 temporary serve of peer backups
- CLI/API: build-host recovery-arm, backup-mirror; opcatalog entries
- Web i18n build-host RU/EN notes

## 0.9.368

- Build-host backup triangle phase: mirror + agent backup_pull over tunnel
- Heartbeat returns cmds; GET /api/build-host/backup/latest
- Web op Installer tab Build-host + POST /v1/build-host

## 0.9.367

- Agent uploads dist.tgz → primary job artifacts; publish unpacks it
- Build-host last_job / jobs_done; API waiting_build count
- build-host backup-mirror (stage primary backup for triangle)
- TG build-host shows jobs stats

## 0.9.366

- Build-host **agent job queue**: claim / work.tgz / result (no SSH CI)
- Ship parks test_build as waiting_external build_agent; agent completes → re-queue publish
- PB/BP tick in metrics + doctor; alerts both-down / pb-down

## 0.9.365

- Secondary agent plane **tunnel-only**: SP preferred, PS failover — never public :8789
- Build-host: build_core_url + PB/BP prefer; agent re-reads each tick
- PLAN updated

## 0.9.364

- PLAN-BUILD-HOST locked: op-only deploy, agent-only CI, tunnel-only B, PB/BP failover, multi-host, fleet last
- Node /api/build-host/deploy → 410; Web node deploy removed
- Agent SERVER= PB VIP 10.87.12.1:8789; real token at deploy
- Ship: no SSH CI when build-host enabled (agent queue next)

## 0.9.363

- Build-host deploy moved to **operator (Mac)** — not primary
- Wizard: Build-host VPS (creds → harden → agent → PB/BP tunnels)
- svc-paths PB/BP dual WG (10.87.12/13) analogous to SP/PS
- CLI: netductor-op deploy build-host; primary deploy removed

## 0.9.362

- Build-host **production deploy** from primary (`build-host deploy`)
- Agent ROLE=build + heartbeat API; SSH probe/alerts offline/disk/agent
- UI: CLI list/watch/deploy, API, TG Git→Build-host, Web deploy+status

## 0.9.361

- Optional **build-host** VPS: control on primary, test_build via SSH
- CLI `build-host show|set|test|enable|disable|clear`; API GET/POST `/api/git/build-host`
- `NETDUCTOR_BUILD_HOST_FALLBACK=1` to fall back to local CI on remote failure

## 0.9.360

- CI next: step timeout (45m), step duration, ship job log file
- Project CI secrets (`project secret set|list|rm`)
- Dependency ci-cache (GOCACHE/GOMODCACHE/npm) + prune 14d
- Auto-ship: on_push/on_tag/path_ignore; SHA256SUMS on publish
- TG 📜 job log; doctor ci-cache line

## 0.9.359

- Ship worker: async queue (API/TG enqueue; one VPS job at a time)
- build-all / BuildOnFetch → ship; opcatalog + i18n Ship
- Web: Ship jobs controls; hard-cut messaging on release ship / git build

## 0.9.358

- TG/Web parity: project **Ship** (not Build), ship jobs board, pause/resume/rm
- API: `/api/git/projects/ship`, `/api/git/projects/ship-jobs`

## 0.9.357

- S3/S4: project pause (after step), resume (no redo ok), complete --from
- S5: auto tag per project on ship; project next <name>
- S6: doctor INFO ship job counts + vps_slot

## 0.9.356

- S1: `netductor project ship <name>` — job + VPS slot + build + publish artifacts (host=mac → waiting_external)
- `project ship netductor` → update.Ship (releases matrix)
- `git project build` removed (use project ship)

## 0.9.355

- S0 PLAN-PROJECT-SHIP: ship job store (`ship-jobs/`), states, vps_slot, CLI `project jobs|job|job rm|next`
- `project ship` / pause / resume / complete → later phases

## 0.9.354

- release ship: tests → matrix → local releases (failed tests do not publish)
- release next / build without tag: auto patch bump
- git-artifacts prune (release prune-artifacts; keep 20 after ship)

## 0.9.353

- TG: device profiles list uses same table + # buttons pattern as Users

## 0.9.352

- CI runner: auto-rewrite `go test ./...` → sequential even if workflow YAML is old
- Same as 0.9.351: docker mem/cpu caps, local releases preferred

## 0.9.351

- CI: `go test -p 1 -parallel 1` (sequential packages); docker `--memory=1536m --cpus=1.5`, GOMAXPROCS=1 (protect VPS/bot from OOM)
- Updates: LatestReleaseTag / ListReleases prefer local `/var/lib/netductor/releases` when present
- Docs: VPS-first release + apply without GitHub

## 0.9.350

- TG: user profile card button rows ≤5 (Telegram limit) — menu no longer drops on profile select
- Web op: edge form prefill from operator config restored

## 0.9.340

- recover-drill: `netductor backup drill-mark` writes timestamp for digest age

- Incident timeline (alert/recovery) for API, TG, Web
- Clearer build_on_fetch UX + visible weekly CI cleanup policy
- Explicit operator session TTL; audit viewer notes; OIDC design
- Compact TG primary status card
- Backup verify status + recover-drill age in fleet digest
- Explicit DNS path note in doctor + Web
- Fast-test profile docs (~2 GiB); doctor VERSION vs GitHub tag; PLAN EN+RU policy


## 0.9.291

- Docs: Edge deploy hybrid marked done; migrate-from-GitHub runbook (SELFHOST-GIT + PLAN-INTERNAL-RELEASES)
- OPEN_ITEMS: edge hybrid closed (lab e2e remains)

## 0.9.290

- Alerts: drop channel:reality-total pages (scanner noise on :443); keep reality-sec only when uplink is down
- Metrics still record reality_invalid_total for channel status / history

## 0.9.289

- TG Status: no triple-button to same Updates screen; digest is fleet digest; Updates only under Tools
- Git project add parity: Web form (name/upstream/workflow), TG multi-step, opcatalog actions

## 0.9.288

- TG menu (PLAN-TG-MENU): five hubs Status / Users / Fleet / Tools / Operator
- Tools = day-2 only (no Status/Users/Fleet/Alerts clone); Stack under Operator
- Status: channel one-liner + digest/channel/versions nav

## 0.9.272

- Fix: stack apply / node upgrade flush TG alerts before CLI exit (were queued 25s and lost)
- nodes upgrade: start/done alerts to channel

## 0.9.271

- Edge agent: detect OpenWrt wifi-device radios via UCI (guest AP not hard-coded to radio0/Cudy)
- docs/OPENWRT-LAB.md — Mac/VPS lab for deploy e2e
- Git tags pruned to latest pair (ops)

## 0.9.270

- TG: remaining Topics buttons → Alerts; comments; unit tests for channel secret
- GitHub: prune historical releases (keep latest pair only)

## 0.9.269

- Remove TG private/forum topics entirely; alerts channel + flat admin DM only
- MigrateAwayFromTopics clears topics.json; Topics UI → Alerts channel screen

## 0.9.268

- TG: SR Config document + picker always get Menu/back keyboards; QR photo keeps access nav; flat-DM note without Media topic

## 0.9.267

- Test alert bypasses AlertOnce cooldown (ClearAlert rearm was blocking tests for 5m)

## 0.9.266

- Fix alerts-chat pick callback (was nested under test case → unknown action)
- Channel alerts: sendMessage only; surface TG API errors; test clears cooldown and reports result

## 0.9.265

- Flat DM menu when alerts channel is set or topics disabled: no message_thread_id to deleted Control topic; /menu force works in private chat

## 0.9.264

- TG alerts channel UX: Set → forward post or type id; list known chats (my_chat_member/channel_post/forward); pick buttons

## 0.9.263

- Configure alerts channel in all UIs: CLI `tg-alerts`, API `/api/telegram/alerts-chat`, TG Topics + `/alerts_chat`, op TUI field

## 0.9.262

- cleanup: brew-style `netductor cleanup [--apply]` (attempt, nd-sb temps, legacy WG); auto after stack apply; API + TG Tools; docs/CLEANUP.md
- notify: optional `telegram_alerts_chat_id` channel for alerts (no topics); skip hub re-pin when set; hub text simplified
- OPEN_ITEMS: dual-node smoke 0.9.261 marked done 2026-10-06
- docs: drop obsolete REVIEW-0.9.73/75, B3-B4, BACKBONE-WG; TG-ALERTS-CHANNEL.md

## 0.9.259

- Baseline snapshot at end of install/EnsureHostBaseline → /var/lib/netductor/baseline/{ss,units,dpkg,ufw}
- Firewall status in metrics + fleet digest; TG alert when !OK; /api/firewall/status requires session
- WEB-UI: Primary domain wording; CF proxy marked advanced

## 0.9.258

- R1 complete: secondary builders (inbounds, outbounds/svc, DNS, route) in secondary_builders.go; WriteSecondarySingBox orchestration-only

## 0.9.257

- R3: opcatalog Destructive/ByID helpers + tests for TG mutating actions
- R7b: POST /api/session/issue (loopback); op session issue via curl JSON not `vpn session hours` shell
- R11a: sing-box pinned tag (default v1.11.15); optional NETDUCTOR_SINGBOX_SHA256
- R12a: crane via download.Get; pin v0.22.1; optional NETDUCTOR_CRANE_SHA256
- R1: route_builders.go (primary inbounds + outbounds/route); secondary assembleSecondaryRouteRules

## 0.9.256

- R5: secondary provision password via NETDUCTOR_SSH_PASSWORD (not argv)
- R15: stack apply polls API health instead of fixed long sleeps
- R2: addons Versions single shape (items + compat lampac)
- R6: apiSessionGate middleware — /api requires session except agent/device paths

## 0.9.255

- Quick R batch: R11b secondary 10.88 rules before sniff; R12b no subscription artifacts; R13 vpn users native-only; R14 skip-verify needs CONFIRM; R4 local-cmd upgrade via SHA; R10 openssl pass via env; media preset respects ServiceEnabled

## 0.9.254

- R9: shared `internal/download` (https policy, size cap, optional SHA, atomic write); used by install, update, deploy EnsureAgentBinary, agent downloadFile

## 0.9.253

- R8: TG/CLI agent_update fills URL|sha256|confirm=yes from release SHA256SUMS + device arch (heartbeat reports arch)

## 0.9.252

- A: offline agent cache verifies against cached SHA256SUMS (SHA256SUMS / SHA256SUMS-<ver> next to agents); online EnsureAgentBinary writes sums into the cache dir

## 0.9.251

- F23: EnsureAgentBinary verifies SHA256SUMS after download (offline cache unchanged)

## 0.9.250

- F22: NVR IssueClipToken/RedeemClipToken enforce PathUnderRoot (defense in depth)

## 0.9.249

- F21: install httpDownload https-only + size cap; blocky checksums; telegram via DownloadReleaseAsset (SHA256SUMS)

## 0.9.248

- F20: agent_update/sysupgrade require SHA256; download http(s) only (http→private IP); size cap; TG no empty agent_update

## 0.9.247

- F19: operator session/issue — sanitize hours (1–168 digits) and SSH user/host; shorten token log

## 0.9.194

### Hoster monitoring / CM audit
- Expanded purge beyond zabbix: telegraf, datadog, salt-minion, puppet, chef, node_exporter, snmpd, nrpe, netdata, …
- Doctor: FAIL on active unwanted units; WARN on watched ports
- CLI: `netductor host-audit [--json] [--purge]`
- Script: `scripts/audit-hoster-agents.sh` (read-only; `--purge` optional)
- Remote secondary harden: broader unit/package remove list

## 0.9.193

### Security — host firewall (critical)

- New package `internal/firewall`: **always** try `apt install ufw`; on failure **iptables** `NETDUCTOR` chain fallback
- Role profiles: **primary** (SSH 52222, 443, redirect, 8444, ACME 80; 8789 only from SP/PS; deny 22/8788/10050) and **secondary** (52222, 443, 8445; deny 22/8787–8789/80/10050)
- Wired into primary `InstallHardening`, secondary `join`, remote SSH harden script
- **Doctor FAIL** if firewall backend missing/inactive/incomplete
- CLI: `netductor firewall status|apply|heal`
- API: `/api/firewall/status|apply|heal`
- TG: status card block + Tools → FW

Auto-heal only with `NETDUCTOR_FW_AUTOHEAL=1` or explicit `firewall heal --force` / API force=1 (default off — lockout risk).

## 0.9.192

### Hardening
- Idempotent **PurgeHostMonitoring**: disable/mask/remove `zabbix-agent*` packages; deny 10050/10051 if ufw present
- Called from primary `InstallHardening`, secondary `join`/`InstallRelay`, and remote SSH harden script (secondary provision)

### Ops / diagnostics
- Doctor `installed_version` uses `version.Running()` (binary + `/etc/netductor/VERSION`), refreshes stale `state/installed_version`
- Doctor zabbix check includes `zabbix-agentd`
- `stack status` role-aware: secondary reports sing-box + secondary-agent + SP/PS units (not primary API/TG)
- `scripts/collect-vps-state.sh`: `netductor-secondary-agent` unit, journal, host-monitoring section

## 0.9.190

- Edge guest first-boot: `--stage` (no network/wifi reload) so SSH survives until harden+reboot
- Guest UCI: modern `br-guest` device section; full path `/usr/sbin/netductor-agent`
- Network/guest stage failures abort deploy (no false "done")
- Probe arch: strip OpenSSH PQ/Warning noise from uname lines
- Log `psk:` / `join:` after guest stage

## 0.9.189

- Edge order: provision without harden → network/guest (new root pass) → harden → reboot
- SSH prefers password when set (fixes publickey denied after setPass)

## 0.9.188

- OpenWrt provision: no base64 on router — mTLS + root pass via ssh stdin
- Filter OpenSSH PQ warnings from edge error text

## 0.9.187

- operator token persisted in ~/.netductor/operator_token (no rotate on every restart)
- inject token into app.js; clearer 401 in Web stream

## 0.9.186

- OpenWrt/Dropbear: put agent via ssh stdin (no SFTP subsystem); scp -O fallback

## 0.9.185

- fix offline edge: keep PrimaryKey and load .pub for router harden; expand ~ on keys; skip packed client cert when device_id differs

## 0.9.184

- Edge offline pack: Web Prepare offline + Offline deploy checkbox; CLI deploy offline-prep
- Pack: bootstrap token, mTLS CA, optional device cert, all agent arches (refreshed on prep)

## 0.9.183

- Edge offline path:  + optional ; agent binary uses local cache if present (no re-download)
- Primary SSH only when token not provided

## 0.9.182

- DeployEdge: SSH to primary on **:52222** (day2SSHPort); router stays **:22** (factorySSHPort). Fixes bootstrap token read against hardened primary.

## 0.9.181

- WAN DNS field: space or comma; normalize to OpenWrt space-list; Web/TUI hints (recommend 1.1.1.1 9.9.9.9)

## 0.9.180

- Docs/handoff baseline → 0.9.179+
- Web guest: correct default-visible help; PSK/PIN hints (no open+captive placeholder)
- applyGuestOnEdge: template bind only with non-empty device_id (SSH remains first-boot SoT)
- Test: guestSSIDVisible matrix

## 0.9.179

- fix: agent `guest enable --hidden=1` was ignored after default-visible change (0.9.178)

## 0.9.178

- Guest SSID **visible by default** (Web/TUI/CLI/agent template); optional hidden
- Confirmed: DeployEdge SSH arch probe before agent download (already in 0.9.15x+)

## 0.9.177

- Web: password show/hide on page load (Installer + all forms; was Control-only)
- Web LuCI: prefill primary_key from Fleet/Settings; form load/save; device_id help EN/RU
- Release assets: agent linux-arm/arm64/amd64, op darwin+linux

## 0.9.154

## 0.9.155

### Edge
- Auto-detect router GOARCH via SSH before downloading agent (`auto` default)
- Release assets: agent linux `amd64` `arm64` `arm` `mipsle` `riscv64` (mipsle softfloat)
- Explicit decision: no opkg/apk feed; updates remain netductor stack


- **Edge Web/TUI parity:** 2.4+5 Wi‑Fi, LAN/DHCP, WAN static/pppoe, guest, **reboot** after provision
- `DeployEdge` optional reboot; CLI `--reboot`; EDGE-AGENT dry-run checklist

## 0.9.153

- **Force refresh versions:** `InvalidateReleaseCache` + TG 🔄 refresh GitHub, API `?force=1`, CLI `--refresh/--force`
- Web Control: Update status / Release list always force GitHub
- netductor-op update check|list `--refresh`

## 0.9.152

- **Hub force reset:** `/menu force|new|reset`, Topics → «Reset hub» / «Сброс меню» (`m:hub:reset`)
- On failed hub edit → ClearHubMsg + send new (chat clear / delete-for-all)
- ForceRecreateTopics clears hub; after recreate always forceHub into Control
- Topics status hint: vanished menu → `/menu force`
- Secondary `upgrade:vX`: GOARCH assets, curl HTTP code/size in CmdLog (no bare exit status 1)

## 0.9.151

- Bootstrap topics: 🎛 Control + 📎 Media
- Singleton Control hub (edit-first via hub_msg.json)
- QR/documents → Media topic; Access text stays on Control

## 0.9.150

- TG Versions hub (select primary/secondary → Apply)
- Subscription profiles secondary|primary|both (default secondary; operator both)
- docs: TG-MEDIA-TOPICS design

## 0.9.149

- **version.Running:** never prefer probe older than VERSION file; never prefer value older than process Release
- Fleet shows `ver dbg` (bin/file/self) when sources disagree
- `netductor version` fast path before config load

## 0.9.148

- **No spontaneous version changes:** secondary ignores `desired_release` self-update
- Primary no longer sends `desired_release` for auto-upgrade
- Stack apply does **not** queue secondary upgrade
- CLI/API update does not fan-out secondary upgrade
- Upgrade/rollback/heal only on explicit operator action

## 0.9.147

- stack heal: restore prev when prev > running (fixes 121/146 split)
- Stack HTML warns when prev newer
- TG Stack: Heal button

## 0.9.146

- Access card: never photo-edit on VLESS↔Sub; track msg id, always new card then delete old
- redirectBase: secrets/redirect_base fallback
- SR Config: DNS/conf host → IP-CIDR DIRECT; caption lists node IPs
- sendRichWithPhoto returns message_id

## 0.9.134

- **No auto-downgrade:** failed health keeps new binaries (stops 0.9.121 loop)
- **PromoteLastGood:** if prev > running, restore prev (fixes stuck 121 with prev 132)
- Refuse apply target older than running
- Clear stale attempt/ before each apply

## 0.9.133

- **Stack:** last-good `prev/` updated only after successful apply (no more rollback to ancient 0.9.121)
- Pre-apply snapshot in `attempt/` — auto-restore only from attempt on API hard-fail
- TG Rollback requires **confirm**

## 0.9.132

- TG Tools: route `m:cat:nodes|routers|sites|vpn` (aliases used in fleet hub)
- Expand `m:ops:*` sections; unknown callback shows data + Tools button

## 0.9.131

- Apply lock: TTL 8m; clear if unit dead after 45s; TG **Clear lock** button
- Prevents sticky "apply already in progress: v0.9.130"

## 0.9.130

- **Fix stack auto-rollback:** health no longer fails on telegram-bot alone (false fail → rollback to 0.9.121)
- `version.Running()` prefers **binary** over stale `/etc/netductor/VERSION` and self-heals the file
- snapshotPrev stores real running version, not compile const
- TG update: simpler systemd-run (no double restart race)

## 0.9.129

- TG Topics: visible on main menu + Tools; **Recreate topics** button (ForceRecreate)
- Topics screen: BotFather hint — Threaded Mode + **Disallow users to create topics** (stops «New thread» spam)
- Reconcile/Recreate bilingual labels

## 0.9.128

- **Version single source:** Fleet/Stack use `version.Running()` (prefer `/etc/netductor/VERSION`, then binary, then const)
- TG/node `version` CLI prints `internal/version.Release` (no stale `main.version` defaults like 0.9.121)
- `stack.Collect` syncs `current.json` to running version
- Removes dual-display bug (CLI 0.9.127 vs Fleet 0.9.121)

## 0.9.121

- Web Control: Stack status/apply/rollback/watchdog
- TG Fleet digest: stack table + core unit fails + Stack/WD buttons

## 0.9.120

- secondary agent upgrade:vX (node+agent from GitHub)
- stack apply queues secondary upgrade
- secondary-local-timer

## 0.9.119

- backup secondary-local / push-recovery / push-ssh
- recovery upload endpoint
- doctor stack-watchdog timer

## 0.9.119

- backup secondary-local (no primary API)
- recovery POST /recovery/upload + backup push-recovery
- backup push-ssh emergency offsite
- doctor WARN stack-watchdog.timer

## 0.9.118

- stack: pre-backup on apply; TG alerts on apply/rollback/watchdog; FormatHTML
- TG Tools → Stack status / Rollback / Watchdog
- stack-watchdog.timer in status matrix

## 0.9.117

- Topics: private DM support — createForumTopic is valid in private chats (BotFather Threaded Mode); send uses message_thread_id + direct_messages_topic_id
- stack: watchdog-install timer (5min); API stack routes fixed in tree; TG update apply → `stack apply`
- Docs: STACK note on private topics

## 0.9.116

- **stack** orchestrator: `netductor stack status|apply|rollback|watchdog` + API `/api/stack/*` + opcatalog
- Backup prune: only `.ndenc`/`.tar.gz`, keep newest N by stamp; secondary peers/core pruned after backup_pull
- Topics: clearer error if chat is private (forum topics need a group)

## 0.9.115

- **Backup:** exclude `var/lib/netductor/backups` (and registry/nvr/git) — stopped exponential tar growth
- **encryptFile:** >64MiB uses streaming openssl; small files keep AES-GCM
- **TG update apply:** runs `netductor update apply` in a **separate process** (no backup/download inside bot)
- **netductor-telegram-bot unit:** MemoryMax=512M MemoryHigh=384M
- TG HTTP body reads capped at 8MiB

## 0.9.75

- Remove enable paths for footgun env; ndconfig ignores obsolete keys (no Sanitize rewrite)

## 0.9.74

- recover: auto LE re-issue when DOMAIN+LE_EMAIL in conf (certs stay out of backup)
- persist LE_EMAIL on domain set --le
- SanitizeNetductorConf strips footgun keys; no LEGACY_ADMIN / public REDIRECT_LISTEN / recovery key fetch

## 0.9.72

- secondary geoip rule-set: download_detour **direct** (Reality uplink broke TLS to GitHub → sing-box crash loop)
- InstallRedirect: read REDIRECT_TLS_* from netductor.conf after recover; enable HTTPS :8443

## docs-i18n

- Sense-equivalent RU for all remaining product docs under docs/ru/
- EN headers link to RU pairs

## docs-prune

- Removed ~80 obsolete docs (reviews, test plans, duplicate runbooks, archive)
- Kept product + plans + handoff; RU only for operational pair

## 0.9.71-docs

- Live unattended recover marked passed
- EN+RU handoff/open items/recover; archive old reviews
- README → 0.9.71

## 0.9.71

- install.Run: continue on per-component errors (do not abort remaining list)
- recover: post-restore second pass for api/telegram/backup/vpn-users/… after secrets are on disk
- Fixes unattended recover stopped at vpn-users validation before tar restore

## 0.9.70

- Recover: always merge DefaultComponents (incl. telegram); backup COMPONENTS never sparse core
- InstallTelegram: real binary path, no self-symlink loop
- UFW :8789: SyncAgentAllowFromSecondaryRegistry after recover + heartbeat AllowAgentMTLSFromIP

## 0.9.69-debt

- Secondary verified :52222 + updated to 0.9.69
- NVR TG i18n remaining strings; Web residual labels
- OPEN_ITEMS: debts closed; owner-only recover/hardware

## 0.9.69-docs

- Formula: **real** sha256 for all op arches; never `:no_check` ([BREW.md](docs/BREW.md))
- Secondary day-2 SSH **:52222** + same Mac key — docs/FREEZE/PORTS/DEPLOY-MAC
- SHA256SUMS on release; linux-arm64 op asset
- i18n: NVR RTSP/enqueue prompts EN/RU

## 0.9.69

- B1: `netductor-op update` from GitHub Release (component op)
- B2: [RUNBOOK-INSTALL-RECOVER.md](docs/RUNBOOK-INSTALL-RECOVER.md) idempotent install/update/recover
- Formula track v0.9.69 (`sha256 :no_check` until CI fills digests; prefer op update)

## 0.9.68

- A5 doctor footgun env; cert expiry WARN already
- B1 agent desired_release self-update
- B3/B4 posture docs; i18n claim EN/RU; I18N.md

## 0.9.67

- OPCATALOG: intentional gaps documented; health all surfaces; metrics-hist +TUI

## 0.9.66

- B5: opcatalog surfaces web/tg/cli/tui + matrix (`netductor opcatalog`, docs/OPCATALOG.md)
- Metrics not on TG (Status only) — less duplication

## 0.9.65

- A6: `netductor update` downloads GitHub Release → `/usr/local/bin` (node|tg|agent)
- A2: DEPLOY-PARITY.md + Web fleet key_passphrase
- Recover drill documentation
- Freeze path: no /opt in update install

## 0.9.62

- **Backbone WG v1:** `internal/backbone`, CLI `netductor backbone init-primary|export|init-secondary|apply|status|conf`, doctor WARN/OK
- **Uplink mux A/B:** `NETDUCTOR_UPLINK_MUX` / `uplink_mux_mode` → on|off|h2mux; `netductor uplink-mux`
- Docs: BACKBONE-WG.md CLI section

## 0.9.61

- Docs: backbone WG/AWG design spike ([docs/BACKBONE-WG.md](docs/BACKBONE-WG.md)); OPEN_ITEMS + handoff next steps

## 0.9.60

- TG: lampac→addons; sites hub≠list; locations legend+tools back; git empty copy; sites vs locations hints

## 0.9.59

- TG: ⬅️ parent nav; NVR sub→hub; go2rtc; guest/loc/mtls/git/addons; drop Status fleet buttons

## 0.9.58

- TG: unified « Parent nav everywhere; sites parent fix; RSC table; sync/RU-exit i18n; addons key

## 0.9.57

- TG: navKeyboard « Parent; drop enroll from Nodes; Operator/Admin Mac; NVR/addons/RSC/lampac polish

## 0.9.56

- TG i18n: probes no double title, backup/DNS/tools RU strings

## 0.9.55

- formatProbes table; role core→primary display + List normalize
- docs/PATHS.md: /opt/netductor/bin is truth

## 0.9.54

- TG nav dedupe: Status slim; Tools ops-only; Nodes without Backup; ops redirects
- docs/TG-NAV.md

## 0.9.53

- Access UI: Core label → Primary; more static text cleanup

## 0.9.52

- Naming: primary/secondary (not core/relay) in TG, TUI, docs
- NormalizeRole: core→primary, relay→secondary
- Auto hostname nd-primary-* / nd-secondary-* (fix secondary applyHostname)
- VLESS fragment #nd-primary

## 0.9.51

- TG Probes/Secondary status via format.API (tables)
- NVR config/motion via format.API
- i18n tools key on main menu

## 0.9.50

- TG: nodes rename body picker; NVR labels i18n; help hub shortcuts

## 0.9.49

- TG hubs Fleet/Nodes/Routers/Sites/Tools: body actions, keyboard nav only
- SHARED-OPS: thin UI → one API method rule

## 0.9.48

- Shared SSH harden: hardening.DropInConf + RemoteHardenScript (primary+secondary)
- TG Operator/Addons: actions in body, keyboard nav only
- docs: SHARED-OPS.md

## 0.9.47

- Secondary harden-last now sets SSH **Port 52222** (parity with primary); was password-only on :22
- DeploySecondary credentials collect uses :52222 after harden

## 0.9.46

- TG: Status is single overview (host metrics + services + nodes + VPN); no separate Metrics menu
- Catalog metrics/health = web-only on TG surface
- Tools hub: drop duplicate DNS button
- docs: TG-SCREEN-INVENTORY.md

## 0.9.45

- TG template C: metrics as expanded key/value table
- docs/TG-UI-PATTERN.md: A/B/C templates + full screen matrix + audit checklist

## 0.9.44

- TG: metrics/status nested tables (mem/disk/load/services) not `{N keys}`
- Secondary uplink: mux padding + stream limits; agent TCP probe primary:443 + auto restart sing-box after 3 fails
- Primary: TG alert when secondary reports uplink_ok=false

## 0.9.43

- fix: doctorPrintf/doctorPrintln infinite recursion (stack overflow)

## 0.9.42

- Full code/security review: [docs/REVIEW-0.9.41.md](docs/REVIEW-0.9.41.md)
- Handoff / UI-PARITY / OPEN_ITEMS for next chat (no functional must-fix)

## 0.9.41

- TG: Guest VPN, Edge Guest Wi‑Fi, mTLS, Locations add, Updates, Sessions, Secondary — body actions (DNS/Users style)
- EN/RU labels for those screens; keyboard = navigation only

## 0.9.40

- Full code/security review: [docs/REVIEW-0.9.39.md](docs/REVIEW-0.9.39.md)
- Handoff + UI-PARITY for next chat; Installer i18n site/edge labels
- opcatalog: dns-set POST

## 0.9.39

- Web Control: EN/RU labels for DNS, backup schedule, Git forms
- TG Tools catalog: DNS section label; registry ensure/crane/stop/catalog rich HTML
- Docs handoff polish

## 0.9.38

- TG SSH hosts: delete/clear/forget in body; keyboard nav only
- TUI Ops: git repos/pipelines, registry status, domain show
- Docs: OPEN_ITEMS / TG-UI-PATTERN / handoff catch-up to 0.9.38

## 0.9.37

- TG catalog: section actions + 📄 JSON in message body; keyboard = nav only
- Web Git: init/log/show/artifact forms; fix log/show query params (`name`/`rev`)
- API git: accept `repo`/`path` aliases for log/show

## 0.9.36

- TUI Ops: DNS list/on/off/reload; backup list/schedule
- CLI: `netductor backup schedule` [set hour minute utc|local]
- TG NVR: lease host picker → table + numbered body buttons (nav keyboard only)

## 0.9.35

- Web Control: backup schedule POST form (hour/minute/UTC)
- Fix broken DNS section in app.js (0.9.34 syntax); dns-set via special handler
- audit: backup.schedule on POST

## 0.9.34

- CLI: `netductor dns list|on|off|reload`
- Web Control: DNS set form (id + on/off)

## 0.9.33

- API: /api/dns/lists|set|reload; /api/backup/schedule|list
- Web Control: DNS section; backup schedule/files buttons
- opcatalog: dns + backup schedule/list

## 0.9.32

- OPEN_ITEMS: CDN-XHTTP plan B (idea only)
- NVR TG: compact table + body actions (hub, cams, sites)
- Functional parity matrix draft

## 0.9.31

- TG: compact emoji legends under titles (Users, Nodes, Git)

## 0.9.30

- Users/VPN: table + numbered open; compact user hub emoji actions
- Nodes: numbered open; compact node card
- Git / Registry / SSH hosts: same TG-UI pattern

## 0.9.29

- TG UI pattern locked (docs/TG-UI-PATTERN.md): table=status, numbered/compact tg-button-row actions
- Apply pattern: backup list/schedule, pending edge, locations
- DNS already on pattern; edit-first refresh

## 0.9.28

- DNS: numbered compact buttons in one row; table keeps full names
- DNS refresh: edit in place, delete+send only as fallback

## 0.9.27

- DNS UI: compact table (status only) + link-style buttons with name inside (no extra labels)

## 0.9.26

- DNS: Telegram does not fire callbacks for tg-button inside <td>
- Layout: read-only table + Enable/Disable in tg-button-row under each list (works)
- Keyboard under message: nav only again

## 0.9.25

- TG DNS: fix no-op clicks — nil-safe callbacks, log data, delete+sendRich refresh
- DNS keyboard mirror toggles (safety if in-table callback not delivered)
- handleCallback: ack spinner before admin check

## 0.9.24

- TG: never classic-fallback when HTML has tables/tg-button (DNS in-table actions)
- apiPost treats Telegram ok:false as error so edit fails → delete+sendRich
- DNS menu: status as HTML; Back → Tools

## 0.9.23

- SNI health: Reality plain-TLS fail is OK; alert only if TCP dial fails

## 0.9.22

- format: dedicated Health / Metrics / Metrics-history renderers (no raw JSON primary view)
- TG catalog reply: editRich first, then sendRich

## 0.9.21

- format: robust smart tables for all catalog JSON (not only health)
- TG catalog: always sendRichMessage (avoid edit stripping tables)

## 0.9.20

- `internal/format`: API JSON → rich HTML (tables, emoji) for TG sendRichMessage
- Catalog actions: human view + 📄 JSON button (details/raw)
- Web Control: doctor structured table

## 0.9.19

- TG Tools: sections + actions from `internal/opcatalog` (same as Web)
- Execute via localhost session API (`m:op:<id>`)

## 0.9.18

- Structured doctor JSON (`CollectDoctor` / `/api/doctor`)
- `internal/opcatalog` + `GET /v1/catalog` (Web↔TG shared actions)
- Web Installer NVR tab (parity with TUI NVR wizard)
- Advanced Control hidden unless Settings → Show Advanced

## 0.9.17

- REVIEW-0.9.16 full code/security review; handoff for next chat
- Web Fleet/Primary: cf_proxy checkbox (parity with deploy API)
- OPEN_ITEMS / UI-PARITY refreshed

## 0.9.16

- WebUI full EN/RU labels
- /api/doctor /api/domain

## 0.9.15

- Web Fleet/Primary: Telegram checkbox → tg_token + tg_admin; server validates
- Deferred plans: PLAN-MIKROTIK-API-EDGE, PLAN-SITE-ROOMS; OPEN_ITEMS

## 0.9.14

- DeploySite + MikroTikAction in operator (shared backend)
- Web Installer: MikroTik site + manage; TUI wizard uses DeploySite
- UI-PARITY: OpenWrt + MikroTik on Web and TUI

## 0.9.13

- Architecture clarified: thin UIs share one operator/deploy backend; parity is the rule
- Web Installer: OpenWrt edge via POST /v1/edge (DeployEdge)
- Docs: UI-PARITY, ARCHITECTURE-OPERATOR, handoff corrected

## 0.9.12

- WebUI split: index.html + app.css + app.js (embed)
- CSP without unsafe-inline (style-src/script-src self only)
- Token via meta nd-token, not injected into JS

## 0.9.11

- TUI tools expanded: edge, NVR, git, registry, addons, mTLS, sites (remote CLI)
- TG Tools: probes, metrics, secondary, audit shortcuts
- Docs: WEB-UI-NOTES (index/CSP), UI-PARITY update

## 0.9.10

- Security hardening: sessionStorage for node session; stronger CSP/COOP; confirm Advanced POST
- Control section labels EN/RU

## 0.9.9

- WebUI Control: max operator-session API coverage + Advanced generic API
- EN/RU chrome; docs archive; UI-PARITY + handoff; SECURITY-REVIEW-0.9.9

## 0.9.8

- GET /v1/session/local — autofill node session from ~/.netductor/node_session
- WebUI EN/RU tabs; table row actions (VPN on/off, edge approve/deny)
- Extra POST: edge recovery, dhcp_static, mtls rotate, vpn enable/disable/revoke

## 0.9.7

- Control POST: NVR PTZ/recorder, git pipeline run, registry ensure/stop
- Tunnel path: prefer API direct if healthy; else SSH via VPN host when reachable, else public

## 0.9.6

- Control: full section coverage (VPN, Edge/guest, NVR, Git/Registry, Backup, Probes, …)
- Result: HTML tables + raw JSON toggle

## 0.9.5

- P4: managed SSH tunnel from WebUI (start/stop/status, auto on Control)
- P4: Issue session from Control → localStorage + ~/.netductor/node_session

## 0.9.4

- Node API-only by default: `/admin` returns 410 unless `NETDUCTOR_LEGACY_ADMIN_UI=1`
- Install no longer ships static admin UI by default

## 0.9.3

- Control P2: VPN users, nodes, edge pending/approve via `/v1/node` proxy (CORS-safe)
- `netductor-op session issue` — create node API session over SSH
- Node session token in WebUI Settings

## 0.9.2

- Locked: Mac client UI only; node = API server ([PLAN-MAC-CLIENT.md](docs/PLAN-MAC-CLIENT.md))
- WebUI: Installer | Control | Settings; tunnel CLI; Control hits node `/health`
- VPS `/admin` documented as legacy

## 0.9.1

- Operator WebUI: Fleet / Primary / Secondary / Credentials tabs, EN/RU, step chips, localStorage
- API: POST /v1/primary, /v1/secondary, /v1/credentials; GET /v1/meta

## 0.9.0

- Physical split: **netductor-op** (Mac) vs **netductor** node (VPS)
- Recovery: default bind **0.0.0.0 while armed** (DR); doctor WARN not FAIL
- Security: path/XFF/token hardening (see prior 0.8.98–0.9.0 commits)

## 0.8.98

- security(operator): token auth for /v1/fleet; host validation; credentials role sanitize
- operator serve: timeouts, deploy mutex, max body 1MiB, CLI --token

## 0.8.97

- `netductor operator serve` — localhost UI + `POST /v1/fleet` (loopback only)
- embed WebUI for FleetDeploy

## 0.8.96

- operator.Step + FleetDeployWithReport structured progress events

## 0.8.95

- internal/operator use-cases; CLI/TUI thin clients; deploy fleet

## Docs (post-0.8.94)

- [ARCHITECTURE-OPERATOR.md](docs/ARCHITECTURE-OPERATOR.md) — operator core + UI clients
- [OPERATOR-PLAN.md](docs/OPERATOR-PLAN.md) — phased checklist
- Handoff compressed; AGENTS.md v2.6 mandatory doc/checklist discipline

## 0.8.94

- Remove legacy huh deploy wizards; framed Setup only
- Wizard **Fleet**: primary+secondary one run (domain/LE/add-ons/TG → credentials on Mac)
- Primary wizard: lampac/git/TG toggles + LE parity with CLI

## 0.8.92

- Default: no CF proxy for multi-level `i.netductor.*` (grey + :8443)
- TUI `cf_proxy` default **no**; DOMAIN.md Universal SSL limit note

## 0.8.91

- --cf-proxy / TUI cf_proxy for Cloudflare orange on i.
- REDIRECT_BASE without :8443; Origin Rule → :8443

## 0.8.90

- Secondary RemoteJoin: no ancient FALLBACK 0.8.45 — download current release only
- docs/UI-PARITY.md rewritten for framed TUI + Mac-direct secondary
- DOMAIN.md: Cloudflare orange for `i.` is optional
- OpenWrt wizard labels FormT; stale huh-path comments fixed

## 0.8.89

- **credentials collect**: full dump of `/etc/netductor/secrets`, conf, LE, secondary devices.json → `~/.netductor/credentials/<role>-<host>-ts/`
- Port fallback 52222/22; CLI `--port`; secondary `--primary-key-passphrase` `--secondary-key`
- DEPLOY-MAC + handoff updated

## 0.8.88

- Redirect after LE: **:8443 only** (no public :80); certbot renew uses free :80
- Deploy automation: key chmod 600, empty .pub repair, REDIRECT_BASE :8443
- TUI≡CLI parity docs [DEPLOY-MAC.md](docs/DEPLOY-MAC.md); FormT le_email
- InstallRedirect prefers LE HTTPS unit

## 0.8.87

- LE redirect: `REDIRECT_BASE=https://i.<base>:8443` (port 443 is Reality, not LE)
- ufw allow 8443/tcp on tls le; docs DOMAIN.md Reality vs LE

## 0.8.86

- **Secondary deploy Mac-direct:** primary only `prepare-pack` (token/mTLS/bundle); **Mac SSHs to secondary** (no primary→secondary SSH)
- CLI: `netductor secondary prepare-pack --sni …`

## 0.8.85

- deploy: SSH_ASKPASS fallback without sshpass; CLI --domain-base/--le-email
- redirect HTTPS **:8443** (443 = Reality)
- secondary deploy defaults NETDUCTOR_SSH_PORT=52222 for primary
- agent logs failed ticks; NETDUCTOR_KEY_PATH for durable key path

## 0.8.84

- **Let's Encrypt:** `netductor tls le` + `domain set --le --email`
- Primary/TUI: optional LE email with domain base → auto certs + HTTPS redirect
- Redirect unit :80 + :443 when certs present

## 0.8.83

- `netductor domain set|show` — primary/vpn/redirect from `--base`
- Primary wizard optional domain base; deploy runs domain set on host
- Docs: DOMAIN.md

## 0.8.82

- Recovery arm: **HTTPS** self-signed by default (`RECOVERY_TLS=0` for HTTP)
- Credentials: keep last 5 files per role
- Handoff/docs: knock history trimmed; baseline 0.8.82

## 0.8.81

- Credentials: no file on SSH/secret failure; `latest-<role>.txt` symlink; iCloud warning
- CLI: `netductor credentials collect`
- Handoff: canonical baseline (no knock)
- TUI log marks credentials step

## 0.8.80

- Remove recovery port-knock; arm over SSH only
- After primary/secondary deploy: write `~/.netductor/credentials/*.txt` (BACKUP_KEY, RECOVERY_TOKEN, SSH)
- Docs: OPERATOR_CREDENTIALS.md

## 0.8.79

- Recovery: remove always-on legacy; 8-port **random** knock sequence per host (secrets file)
- CLI: `recovery knock-show` / `knock-regen`

## 0.8.78

- Recovery HTTP **off by default**; `netductor recovery arm|disarm|status`
- Port-knock sequence (default 41222→41223→41224) arms recovery for TTL
- Recovery token compare: constant-time
- Edge: TUN start failure → automatic socks@127.0.0.1 fallback

## 0.8.77

- Edge VPN default mode: **tun** (was socks/mixed)
- socks/mixed remains opt-in via template `vpn.mode` or explicit mode

## 0.8.76

- Recovery: **no backup key on the wire** by default; `--key` / `NETDUCTOR_BACKUP_KEY` required for recover-from-secondary
- Recovery: real CIDR allowlist, auth lockout (5 fails → 15m), optional `RECOVERY_SERVE_KEY=1`
- TUI: clamp log scroll (no vanishing lines past top)
- Edge mixed inbound: default `127.0.0.1:7890` (override `NETDUCTOR_EDGE_MIXED_LISTEN`)

## 0.8.75

- Telegram **fully optional**: removed from DefaultComponents / core primary install
- `netductor install telegram` installs binary + systemd unit (error if binary missing)
- Doctor: INFO skip when bot unit not present
- Formula + release assets include netductor-tg

## 0.8.74

- Primary deploy: **no** automatic lampac / git / registry (only core install)
- Git+Registry moved to Add-ons checklist (`git_registry` toggle)
- Primary progress checklist: core steps only
- TUI: disable mouse capture so terminal select/copy works (scroll: arrows / PgUp/PgDn)

## 0.8.73

- Primary wizard: removed TG token/admin (use Add-ons)

## 0.8.72

- Add-ons: target **any VPS** (host/user/key), not primary-only
- Telegram: token + admin id fields when installing bot
- Primary still has no Lampac yes/no (use Add-ons)

## 0.8.71

- Primary wizard: **removed** single Lampac yes/no (was confusing)
- Add-ons checklist is the only multi-select path (Мастер → Дополнения)
- Tools → same checklist

## 0.8.70

- TUI Add-ons: **checklist** [✓]/[ ] with Space toggle (not sequential yes/no batch)
- Ctrl+R from checklist installs selected only

## 0.8.69

- **Fix:** publish `netductor-tg-linux-*` on releases; InstallTelegram clearer fallback; primary deploy ensures bot after secrets
- **TUI Add-ons multi-select:** Lampac / Telegram / go2rtc (placeholder) each yes/no independently

- Plan: TUI Add-ons multi-select (per component), not all-or-nothing

## 0.8.67

- TUI deploy: **line-by-line stream** + **checklist/progress bar**
- Toggle views: **p** = progress/checklist, **o** = log (not both at once)

## 0.8.66

- **TUI integrity rule** locked (`docs/TUI-RULES.md`): no external huh/quit for operator flows
- Framed Output (header+border); Tools forms stay in-TUI; site/addons via wizard fields

## 0.8.65

- TUI Wizard: deploy runs **inside** framed UI (async log pane, no bare Output screen)

## 0.8.64

- SSH deploy: clear stale known_hosts before connect (fixes exit 255 after VPS reinstall)
- TUI wizard paste: accept long KeyMsg strings from terminals without bracketed-paste flag

## 0.8.63

- TUI Wizard fields: paste support (bracketed paste + Ctrl+V clipboard), multi-rune input

## 0.8.62

- TUI Wizard: menu-style split layout — short hints + right detail pane for all field steps

## 0.8.61

- TUI Wizard: multi-step fields **inside** framed UI (no external huh window)
- Primary / Secondary / OpenWrt / NVR / Add-ons collect data in-screen; result on Output

## 0.8.60

- go2rtc: WebRTC listen **127.0.0.1:8555** (was `:8555` all interfaces)
- PORTS.md: document go2rtc 1984/8554/8555 localhost-only

## 0.8.59

- TUI: all deploy wizards (Primary/Secondary/OpenWrt/MikroTik/NVR/Add-ons) only outside alt-screen
- Wizard tab: Add-ons entry; Tools site/mt/ssh forms exit to huh
- runWizardApply no longer invokes huh inside Bubble Tea

## 0.8.58

- TUI Wizard tab: exit alt-screen before huh Primary/Secondary/OpenWrt/… forms (fixes blank UI)

## Formula

- Homebrew: pin **0.8.57** with real SHA256 (no `:no_check`)

## Docs

- DEPLOY-WORKSTATION + handoff + OPEN_ITEMS locked to **v0.8.57** (TUI flow, recover e2e)

## 0.8.57

- recover: **peek operator pubkeys from backup tar before harden** (no required env)
- NETDUCTOR_OPERATOR_PUBKEY remains optional extra merge

## 0.8.56

- recover: inject operator pubkey before harden + reload sshd after restore
- PLAN-NVR: go2rtc two-way audio (`tapo://`)

## 0.8.55

- probes: api-health default **http://127.0.0.1:8787/health** (API is HTTP-only); migrate old https defaults

## 0.8.54

- TG: restore in-body rich `<tg-button>` (Access/hub); warn if REDIRECT_BASE unset
- install: EnsureClientProfiles (`nd-oc.conf`), EnsureDomainConfig, InstallRedirect
- Env: `NETDUCTOR_DOMAIN` / `NETDUCTOR_REDIRECT_BASE` / `NETDUCTOR_PUBLIC_HOSTNAME`

## 0.8.53

- TG: Access/User hub use **classic inline_keyboard** (no broken tg-button text glue)
- importRedirectURL returns empty without absolute http(s) base (fixes BUTTON_URL_INVALID)
- replyRichWithPhoto prefers sendPhoto + keyboard

## 0.8.52

- Backup: snapshot live hostname + operator **public** keys (`hostname.backup`, `operator_authorized_keys`)
- Recover: SkipHostname (no invented `nd-core-*`); restore hostname from backup; inject operator pubkeys (+ `NETDUCTOR_OPERATOR_PUBKEY` / `_FILE`)

## 0.8.51

- Remove SCP backup.offsite path (agent backup_pull only)
- TUI: current mode chip in header; clearer Mode tab; persist mode in tui.yaml; confirmation screen on change

## 0.8.50

- Fix: StartRecoveryServer in AgentLoop (was missing on secondary)
- Backup: TG alert when no online secondary for backup_pull
- Recovery: NETDUCTOR_RECOVERY_BIND / UFW / ALLOW_CIDR + rate limit

## 0.8.49

- DeployPrimary: after harden auto `NETDUCTOR_SSH_PORT=52222` for post-steps
- install lampac: Debian `docker-cli` + `docker.io`
- registry crane: correct GitHub asset names (Linux_x86_64, v0.22.1)
- hardening apt: git, wget
- primary deploy: registry+git bootstrap after install
- secondary ProvisionIn: SSH private key auth when password disabled

## 0.8.44

- TUI Setup: **Add-ons** (Lampac) + optional Lampac after primary deploy
- `deploy primary --with-lampac`; `deploy.RunOnPrimary`
- InstallLampac: ensure Docker (apt docker.io / get.docker.com), image fallbacks

## 0.8.43

- Deploy SSH: `NETDUCTOR_SSH_PORT` (default 22; use 52222 after harden)
- Primary binary download: curl **or** wget
- E2E: primary+secondary VPS path validated

## 0.8.42

- CLI `deploy edge`: full parity with TUI network/guest flags (LAN/DHCP/Wi‑Fi 2.4+5/WAN/pppoe)
- `deploy secondary --primary-key-passphrase`; edge/secondary password via NETDUCTOR_SSH_PASSWORD

## 0.8.41

- OpenWrt WAN: dhcp | static | **pppoe** (user/pass/service/AC)
- Wi‑Fi 2.4/5 separate fields; empty band inherits the other (EN/RU hints)
- edgeagent DesiredUCI dual-band + PPPoE

## 0.8.40

- OpenWrt wizard: optional LAN/DHCP/Wi‑Fi/WAN (dhcp|static) EN/RU → UCI on router + edge template
- edgeagent DesiredUCI: wan static, dhcp pool; ShellApply for first-boot

## 0.8.39

- Primary SSH key: optional passphrase in TUI (ask yes/no → phrase + confirm) and CLI `--key-passphrase`
- Key passphrase via SSH_ASKPASS (not argv); secondary/edge wizards can supply primary key phrase
- Form labels EN/RU for passphrase flow

## 0.8.38

- SSH: sshpass uses `SSHPASS` + `-e` (no password in argv)
- fleet provision-secondary: `--password-stdin` / `NETDUCTOR_SSH_PASSWORD`
- Deploy secondary pipes password on primary via stdin
- API `MaxBodyBytes` = 16 MiB, used by `withSecurity` + `readJSON`
- AGENT_HANDOFF baseline cleanup (stale 0.8.30 operator hints)

## 0.8.37

- Security: API systemd restart allowlist (`netductor-*` only)
- Docs/handoff baseline lock after isolated CI + deep review (no large refactor)

## 0.8.36

- Isolated CI: builds/tests in docker/podman by default (`netductor ci status|test|exec`)
- GHA-subset honors `container:`; auto image by go.mod/package.json/…
- Managed pipelines ci-run/go-test/oci-push use isolation; host keeps git+registry data only
- Doctor CI isolation check; seller guest-bot parked under Ideas

## 0.8.35

- Harden git artifact path reads (no `..` escape)
- Docs/handoff/OPEN_ITEMS baseline lock; security review snapshot
- Git+registry+GHA surface marked complete for thin self-host

## 0.8.34

- Git/registry UI parity: Admin+TG workflow, artifacts; catalog tags; registry htpasswd auth
- Pipeline/workflow logs under state/git-artifacts; doctor checks git root + registry
- Sample pipeline `ci-run` (Go/Node/Rust/Python/Make); multi-language docs

## 0.8.33

- TUI: single deploy path (tab Wizard → confirm → huh deploy wizards); drop dead field forms
- NVR in tab wizard targets; GitHub Actions–subset is the only CI YAML target (no GitLab dual-DSL)

# Changelog

## 0.9.0

- security(recovery): default bind 127.0.0.1; non-loopback requires ALLOW_CIDR
- security: SHA-256 constant-time token compares (secondary/edge/recovery)
- doctor: FAIL if recovery :8790 on 0.0.0.0

- security: Content-Disposition filename sanitization; BackupPath rejects control chars
- security: rate-limit / clientIP trust XFF only with NETDUCTOR_TRUST_PROXY=1
- chore: agent version 0.9.0; registerSecondaryAPI rename (drop relay naming)

- **Physical split:** `cmd/netductor-op` (deploy/TUI/operator serve only) vs `cmd/netductor` (node plane only)
- No shared “fat” binary; operator day-2 uses SSH → `netductor` on primary

- **Binary split:** `netductor-op` (operator / Mac) vs `netductor` node (`netductor-linux-*` on VPS)
- Link-time `main.binaryRole=operator|node|all`; command surfaces gated
- Deploy / secondary provision download **node** assets only (never operator binary onto VPS)
- Release workflow emits both asset families; docs + OPERATOR-PLAN Phase 4


## 0.8.99

- security(deploy): release version / agent arch charset; https-only edge ServerURL
- security(deploy): scp uses `--` before paths; token compare via SHA-256

- TUI wizard: all deploy targets via operator (EdgeFromFields, RunRemote); step matcher for Fleet events

- security(operator): stricter ValidHost/ValidUser; DeployEdge use-case
- security(operator): token charset + CSP/XFO on operator serve
- security(deploy): ssh `--` before user@host
- CLI/TUI edge via operator core only


## 0.8.32

- GHA-subset workflow runner: parse `.github/workflows/*.yml`, execute `run:` steps only
- `netductor git workflow <repo> [path]`; post-receive prefers workflow if `NETDUCTOR_GIT_WORKFLOW=1` or path set
- Example workflow in docs; CHANGELOG rebuilt for 0.8.4–0.8.31

## 0.8.31

- Local OCI registry: `registry status|ensure|stop|crane|catalog` + API/Admin/TG
- Pipeline sample `oci-push`; post-receive auto-runs `NETDUCTOR_GIT_PIPELINE`
- Docs SELFHOST-GIT / OPEN_ITEMS

## 0.8.30

- Git: `git delete`, sample `go-test` pipeline, docs/handoff sync
- Thin model: no Forgejo by default

## 0.8.29

- Git pipelines: CLI/API/Admin tab/TG

## 0.8.28

- Thin `netductor git` CLI + SELFHOST-GIT clarification

## 0.8.27

- SSH listen **52222** + fail2ban; SELFHOST-GIT facts

## 0.8.26

- Guest cmd allowlist + TG wait `cmd_result`; certs in TG; docs cleanup

## 0.8.25

- Guest wizard, PNG QR, TG/Admin edge guest control

## 0.8.24

- Guest nft MAC gate, captive DNAT, VPN bypass, DHCP

## 0.8.23

- Guest Wi-Fi allow-list, captive+desk, default grant 10m

## 0.8.22

- Session test, SMTP host, TUI EchoMode, wizard SA4006

## 0.8.21

- SA4000 doctor, dead code removal, HTTP status constants

## 0.8.20

- `go vet` clean; `internal/version` single pin; remove dead TG code & 0.7 install URLs

## 0.8.19

- Full review docs; primary upgrade uses `deploy.Release` (no v0.7 base64)

## 0.8.18

- Admin i18n complete pass, UI parity matrix, mTLS rollover UI

## 0.8.17

- Residual risk mitigations, secondary callbacks, pure-Go upgrade

## 0.8.16

- cli18n full doctor body + VPN CLI strings EN/RU
- Deep code/security/UI review docs

## 0.8.15

- cli18n CLI, `StartAgentPlane`, relay→secondary UI, mTLS in all UIs

## 0.8.14

- mTLS auto-push + grace; CA dual-trust rollover; cli18n

## 0.8.13

- mTLS revoke/rotate; cert expiry doctor; plane IP ban after rate limits

## 0.8.12

- Agent plane rate limit (180/min/IP); FormT deploy labels
- CI version-pin step

## 0.8.11

- Split `netductor-agent` (nvr_cmds / openwrt_net / agent_ops)
- `scripts/check-version-pins.sh`; FormT; docs pin

## 0.8.10

- Mac TUI tab wizard = full deploy wizards; `deploy.Release` pin

## 0.8.9

- TUI deploy/forms/sites RU/EN; TG NVR i18n

## 0.8.8

- Drop IP allowlist; admin EN/RU expanded

## 0.8.7

- Edge NAT-friendly agent plane (mTLS only); admin EN/RU tabs

## 0.8.6

- Agent allowlist experiment (later dropped for edge); UI/docs i18n

## 0.8.5

- Edge mTLS on agent plane `:8789`

## 0.8.4

- mTLS-only secondary agent plane, auto certs, UFW lock

## 0.8.31

- Local OCI registry: `netductor registry status|ensure|stop|crane|catalog` + API/Admin/TG
- Pipeline sample `oci-push`; post-receive auto-runs `NETDUCTOR_GIT_PIPELINE`
- Docs SELFHOST-GIT / OPEN_ITEMS

## 0.8.30

- mTLS: revoke list + VerifyPeerCertificate; rotate/list CLI & API
- Doctor: cert expiry WARN (≤30 days); revoked client flags
- Agent plane: temporary IP ban after repeated rate limits
- Docs: MTLS.md, OPEN_ITEMS (hardware e2e remaining)

## 0.8.30 — 2026-09-20

### Finish remaining refactor items
- Agent plane `:8789` IP rate limit (180 req/min; plain emergency 60)
- `FormT` expanded; deploy wizard labels use dictionary
- CI: `scripts/check-version-pins.sh` step (via GitHub connector)

## 0.8.11 — 2026-09-20

### Refactor
- Split `netductor-agent` into `nvr_cmds.go`, `openwrt_net.go`, `agent_ops.go`
- `scripts/check-version-pins.sh` + CI step (no stale v0.7 in operator paths)
- `FormT` dictionary for shared TUI form labels
- Docs/bootstrap/TG/admin download pins → current release

## 0.8.10 — 2026-09-20

### Mac workstation deploy scenario
- Bubbletea **tab wizard** now delegates to full `wizardPrimary/Secondary/OpenWrt` (same as Setup menu) — no more local-only install on laptop
- `internal/deploy.Release` single pin for binary/agent downloads

## 0.8.9 — 2026-09-20

### TUI / TG i18n
- Deploy wizards, VPN forms, sites, SSH hosts, menu items: full RU/EN via TT()
- TG NVR buttons use i18n dict

## 0.8.8 — 2026-09-20

### Remove agent IP allowlist
- Firewall: deny `:8788`, allow `:8789` (mTLS only) — no IP allowlist
- Edge behind NAT and secondary both use client certificates

### Admin UI i18n
- Tabs and main section labels EN/RU

## 0.8.7 — 2026-09-20

### Agent plane policy (edge + secondary)
- **Edge behind NAT:** no IP allowlist (WAN changes); security = mTLS client cert on `:8789`
- **Secondary:** public IP recorded in `agent_allowlist` (inventory)
- Default ufw: `:8789` open + mTLS; `NETDUCTOR_AGENT_ALLOWLIST_STRICT=1` for secondary-only lock
- Removed edge enroll/heartbeat auto-allowlist

### UI i18n
- Admin tabs + core labels EN/RU

## 0.8.6 — 2026-09-20

### Security default: agent allowlist
- `:8789` **allowlist-only** by default (empty list = closed; no world-open)
- `netductor agent-allowlist list|add|apply`
- Secondary/edge provision and edge enroll/heartbeat add source IPs
- Doctor WARN/FAIL on empty allowlist or world-open 8789

### UI / i18n
- TUI/wizard Primary URL → https://IP:8789
- Admin recovery strings EN/RU; relay one-liner → secondary v0.8.5+
- TG secondary download pin updated

## 0.8.5 — 2026-09-20

### Edge mTLS control plane
- Agent plane `:8789` serves edge + NVR device APIs (alongside secondary)
- `netductor-agent` loads `/etc/netductor-agent/mtls` client certs; forces `https://host:8789`
- DeployEdge / workstation: issue per-device cert on primary, install on router, never plain `:8787`
- Edge control independent of site VPN

## 0.8.4 — 2026-09-20

### Security (secondary agent plane)
- mTLS-only agent plane on **:8789** (plain :8788 off unless `NETDUCTOR_PLAIN_AGENT=1`)
- Certs auto-generated: `EnsureAll` at install/serve; per-node `EnsureClientFor` at secondary provision
- Client material installed in-band over provision SSH session
- UFW: deny 8788; allow 8789; restrict to secondary IP after provision
- Doctor FAIL if 8788 exposed or mTLS missing

### Edge (documented plan)
- Control plane must stay reachable **without** site VPN (mTLS public :8789) so router is not lost when VPN is down

## 0.8.3 — 2026-09-20

### Security / SSH
- Operator Mac pubkey on secondary, OpenWrt, MikroTik after first password bootstrap; password auth disabled where supported
- Primary `install` no longer generates `/root/.ssh/id_ed25519` when `authorized_keys` already has a key
- Secondary: `--operator-pubkey` (workstation deploy passes Mac `.pub`)

### Transport (documented)
- Admin API remains localhost `:8787` (SSH tunnel from Mac)
- Secondary agent: prefer mTLS `:8789`; plain `:8788` legacy
- Edge: prefer HTTPS or VPN path; plain public HTTP is token-only, not confidential

### Docs
- ARCHITECTURE, EDGE-AGENT, DEPLOY-WORKSTATION, AGENT_HANDOFF updated

## 0.8.1-docs — 2026-09-20

### Docs
- Align operator guides to **v0.8.1**: bootstrap default, INSTALL/DEPLOY/RUNBOOK/RELEASES/BOOTSTRAP (+ ru)
- CLI naming: `relay` → `secondary` / `fleet provision-secondary` in FLEET, RUNBOOK, RELAY, VPN-USERS, README
- EDGE-REINSTALL: recovery codes documented as implemented (LAN :7879)
- FLEET: secondary sync; drop lab IP placeholder in remote example

## 0.8.1 — 2026-09-20

### Security hardening
- Edge recovery: prefer private IP bind, LAN-only clients, optional SERVER_PIN
- Self-update verifies release SHA256SUMS
- TG: edge approve/deny callbacks; mask device_token in chat
- serve: EnsureAll mTLS before agent plane; SNI fallback api.vk.me
- Doctor warns on CLAIM_FIRST

## 0.8.0 — 2026-09-20

### Highlights
- **Secondary-only** RU entry (legacy relay naming removed)
- Edge **LAN recovery** (`:7879/netductor-recovery`), recovery codes, register/set-site
- NVR/Tapo Go port foundations; RU-direct for gosuslugi stack
- Primary self-update from **GitHub Releases** (TG Tools → Updates); agents **manual** `agent_update`
- Docs EN+RU parity pass; review `docs/REVIEW-2026-09-20.md`

### Breaking / cleanup
- CLI `relay` alias removed → `secondary`
- State path only `secondary/`
- API only `/api/secondary/*`

## 0.7.39-dev

- UI: edge recovery/register/set-site/export in TG, Admin, TUI + API

## 0.7.38-dev

- Edge recovery codes + LAN page :7879; register/export/import/set-site; CONTROL_ONLY

## 0.7.37-dev

- Rename remaining Relay* identifiers to Secondary*; docs EDGE-REINSTALL

## 0.7.36-dev

- Remove legacy relay naming: paths, CLI, API, roles — secondary only

See git history for earlier 0.7.x / 0.5.x notes.
## 0.8.93

- LE: strip :port from redirect hostname for certbot
- docs example p.nd / i.nd / s.nd
