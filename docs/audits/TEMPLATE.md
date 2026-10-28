**EN** · [RU](../ru/audits/TEMPLATE.md)

# [Audit title]

> Status: draft / in progress / final  
> Report date: YYYY-MM-DD  
> Repository: `PavelNeyman/netductor`  
> Base commit: `<full SHA>`  
> Release/tag: `<version or n/a>`  
> Scope: <subsystems, paths, entry points>  
> Language pair: [RU report](../ru/audits/<matching-file>.md)

## 1. Executive summary

- Overall assessment:
- Critical/high-risk conclusion:
- What is confirmed:
- What remains unknown or requires live-fleet validation:

## 2. Scope and method

- Questions to answer:
- Files/paths inspected:
- Commands/tools used:
- Explicit exclusions and limitations:

## 3. Environment and verification status

| Check | Exact command | Environment/toolchain | Result | Evidence |
|---|---|---|---|---|
| Example | `go test ./...` | Go version, OS | NOT RUN | reason |

Only mark a check PASS if the command was actually executed and its result was observed. Distinguish source inspection, tests, staging, and live-fleet verification.

## 4. Findings

Repeat this block for each finding.

### [ID] Short title

- **Status:** confirmed / hypothesis / not reproduced / blocked / fixed-awaiting-verification / verified-fixed
- **Severity:** critical / high / medium / low / informational
- **Affected component/path:**
- **Preconditions:**
- **Impact:**
- **Evidence:** source path + line/commit, sanitized output, test name or reproducible steps
- **Confidence and caveats:**
- **Recommendation/remediation:**
- **Regression test:**
- **Verification result:**
- **Related issue/PR/plan item:**

## 5. UI and localization review (when applicable)

- Telegram card/keyboard template reused:
- Callback authorization, stale callback, pagination/back behavior:
- Web UI design-system components reused:
- Localization source and keys checked:
- EN/RU semantic parity:
- Automated checks and outcomes:

## 6. Security and data handling

- Secrets/identifiers redacted:
- Trust boundaries and integrity checks:
- Permissions and authorization:
- Logging/telemetry exposure:
- Rollback and recovery safety:

## 7. Decisions and follow-up

| Action | Decision/status | Rationale/evidence | Owner/issue | Next step |
|---|---|---|---|---|

## 8. Open questions and limitations

- [ ] Question/blocker — what evidence or operator action is needed:
- [ ] Live-fleet check not performed — reason and required validation:

## 9. Reproduction notes

List the exact safe commands, required non-secret prerequisites, expected results, and cleanup steps. Do not include credentials or production secrets.

## 10. Change log

- YYYY-MM-DD: initial report, base commit `<SHA>`.
- YYYY-MM-DD: follow-up evidence/status update, linked commit/PR.

## Completion checklist

- [ ] Base commit and scope are exact.
- [ ] Findings are classified as facts, hypotheses, or unknowns.
- [ ] Commands and tests are reported honestly with results.
- [ ] Secrets and sensitive identifiers are redacted.
- [ ] RU counterpart is updated with equivalent detail.
- [ ] Related plan/issue/PR links are present.
