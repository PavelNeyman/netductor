# Edge Agent identity (Phase 4)

## Decision

| Item | Choice |
|------|--------|
| Logical component ID | `agent` (`ComponentAgent`) — already in update contract |
| Binary name | **Keep** `netductor-agent` |
| Paths | **Keep** `/usr/sbin/netductor-agent` (fallback `/usr/bin/…`), config `/etc/netductor-agent/` |
| Rename | **Not** in this phase — no migration required for rename |

Credentials (TOKEN, mTLS under `/etc/netductor-agent/mtls`) must never be copied from another device or rewritten by a binary-only update.

## Install / runtime

| Piece | Location |
|-------|----------|
| Binary | `/usr/sbin/netductor-agent` preferred |
| Config | `/etc/netductor-agent/config` (`SERVER`, `TOKEN`, …) |
| mTLS | `/etc/netductor-agent/mtls/` |
| Init | procd `/etc/init.d/netductor-agent` when packaged |
| Version | compile-time `version.Release`; `netductor-agent version`; heartbeat payload field `agent` |

## Update protocol

1. Primary/operator builds `URL\|sha256\|confirm=yes` via `AgentUpdateArg(tag, arch)` (arch from device heartbeat/facts — **not** guessed).
2. Edge command `agent_update` downloads, verifies SHA-256, checks ELF, stages `.new`, keeps `.prev`, replaces binary.
3. Without `confirm=yes` the update is refused.
4. Enrollment/credentials are **not** part of `agent_update` — binary only.

## Architecture matrix (published assets)

| GOARCH | Asset | Notes |
|--------|-------|-------|
| amd64 | `netductor-agent-linux-amd64` | |
| arm64 | `netductor-agent-linux-arm64` | |
| arm | `netductor-agent-linux-arm` | GOARM=7 class; armv6 **rejected** by `NormalizeEdgeGOARCH` |
| mipsle | `netductor-agent-linux-mipsle` | Cudy TR1200 etc.; big-endian `mips` **rejected** |
| riscv64 | `netductor-agent-linux-riscv64` | |

`NormalizeEdgeGOARCH` maps probe strings (`mipsel_24kc`, `aarch64`, …) or fails closed.

## Compatibility / migration matrix

| From | To | Action |
|------|-----|--------|
| Any prior `netductor-agent` on same paths | Newer tag, same arch | `agent_update` with confirm; config/mtls untouched |
| Wrong arch binary | — | Refuse via arch normalize / ELF run failure |
| Path rename (future) | — | **Blocked** until staged migration written and tested |

## Health / identity proof

- Heartbeat carries `agent` version string and device id used by primary registry.
- Update success on edge is local (binary replaced); primary still needs heartbeat to confirm **running** version on that instance.
