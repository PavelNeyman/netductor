# False-positive TCP probe results for remote secondaries

**Status:** Confirmed code defect; not fixed  
**Identified:** 2026-11-06  
**Affected component:** Primary metrics collector, `internal/probes`  
**Scope:** Dynamically generated TCP probes to enrolled secondary public addresses

## Summary

A remote TCP probe can be reported as successful even when the connection to the remote host timed out. The fallback in `probeTCP()` checks whether the *local Primary* has a listener on the same port. If such a listener exists, the remote probe is changed to `ok=true`.

This makes the metric unsuitable as evidence that the remote Secondary is reachable.

## Code path

In `cmd/netductor/collect.go`, `runCollect()` creates a TCP probe for every enrolled secondary with a non-empty `PublicIP`:

```go
p := map[string]any{
    "name": "secondary-" + d.PublicIP,
    "type": "tcp",
    "host": d.PublicIP,
    "port": 443,
    "timeout": 5,
}
res := probes.Run(map[string]any{"probes": []any{p}})
live = append(live, res...)
```

The probe is implemented in `internal/probes/run.go`, function `probeTCP()`:

1. It calls `net.DialTimeout()` for the requested host and port.
2. If the dial succeeds, it returns `ok=true`.
3. If the dial fails, it calls `ssListening(port, false)`.
4. `ssListening()` executes local `ss -tlnH sport = :<port>`.
5. If any local TCP listener exists on that port, `probeTCP()` returns `ok=true`, with the original failed dial duration and `via="ss"`.

The fallback does not verify that the requested host is local. It therefore conflates a local listening socket with reachability of a remote endpoint.

## Confirmed Primary-side condition

On 2026-11-06, the following command was run on Primary:

```sh
ss -lntp '( sport = :443 )'
```

It showed a listener on `*:443` owned by `sing-box` (PID 755).

Consequently, when the remote dial to Secondary B (`195.140.147.20:443`) times out, the fallback sees the local Primary listener on port 443 and reports success.

## Observed monitoring anomaly

During the 2026-11-06 connectivity incident, history snapshots contained results for `secondary-195.140.147.20` with:

- `ok=true`
- `ms` approximately 5000–5003 ms

The five-second duration is consistent with the configured TCP dial timeout. The success flag is explained by the local-listener fallback, not by a successful connection to Secondary B.

The independent channel/path checks use a separate TCP probe implementation that returns failure when its dial fails; those checks reported TCP 443 unreachable during the outage. They are more reliable evidence of remote reachability than the affected `probes` result.

## Impact

- The `probes` history can contain false-positive success for remote TCP targets.
- Probe-failure alerting can be suppressed because `evaluateSimpleAlerts()` treats `ok=true` as success and clears the corresponding alert.
- Dashboards and incident analysis that rely on this probe may incorrectly suggest that a Secondary endpoint is reachable.
- The `ms` field can show a timeout-length duration alongside `ok=true`, which is an important diagnostic signature.

This defect does not establish the root cause of the 2026-11-06 network interruption. It explains why this particular probe series was misleading.

## Required correction (not applied)

The remote TCP probe must report the result of the remote dial only. A local-listener check must not turn a failed remote connection into success.

If local listener health is required, it should be represented by a separate, explicitly local probe with a distinct name and semantics.

A regression test should cover this case:

- arrange for a local listener on port 443;
- probe an unreachable remote address on port 443 with a short timeout;
- assert that the result remains `ok=false` and contains the dial error, regardless of the local listener.

## Change status

This document records the finding only. No source code, tests, runtime configuration, or server state were changed as part of documenting it.
