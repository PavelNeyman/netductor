**EN** · [RU](ru/AUDIT-ARTIFACTS.md)

# Audit and review artifacts — repository policy

## Purpose

Keep code reviews, security audits, test runs, release checks, production validation, and engineering decisions as versioned, reproducible files in the repository. Chat messages are not the system of record.

## Storage layout

- `docs/audits/` — canonical English audit reports and templates.
- `docs/ru/audits/` — Russian versions with the same findings, evidence, severity, status, and recommendations.
- `docs/UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md` and its RU pair — living implementation checklist.
- Use dated names with a short scope and base commit where practical, for example:
  - `docs/audits/2026-10-28-update-paths-bf54587.md`
  - `docs/ru/audits/2026-10-28-update-paths-bf54587.md`

If a report is too large for one file, split it by subsystem and add an index/overview linking all parts. Keep related EN/RU paths parallel.

## Required contents for each audit/report

1. **Metadata:** date/time and timezone, repository, exact base commit SHA, release/tag if relevant, author/tooling if relevant, and scope.
2. **Question and method:** what was checked, paths/files/commands inspected, and explicit exclusions.
3. **Environment:** OS/toolchain/dependencies for executed tests; distinguish source inspection from tests actually run.
4. **Results:** summary, confirmed findings, hypotheses, unknowns, and items requiring live-fleet/operator validation.
5. **Finding records:** stable ID, severity, affected component/path, preconditions, impact, evidence, remediation, regression test, status, and links to issues/PRs.
6. **Verification:** exact commands, exit status, test counts/output summaries, and what was not run. Do not claim a test passed unless it was executed.
7. **Decision and follow-up:** accepted/rejected recommendations with rationale, open risks, next action, owner when known, and links to implementation plan/issues.
8. **Language parity:** EN and RU reports carry the same material findings and operational details. Update both in the same change.

## Evidence and secret handling

- Never commit passwords, API/bearer/recovery tokens, private keys, session cookies, seed material, complete production configs, user UUIDs, or other live credentials.
- Redact sensitive hostnames/IPs and identifiers when they are not necessary. Use stable placeholders consistently (for example, `PRIMARY_A`, `SECONDARY_B`, `<REDACTED_TOKEN>`).
- Prefer a minimal sanitized excerpt and a source path/line/commit over a full command dump. Do not attach raw `env`, secret stores, private keys, unredacted configs, or unrestricted diagnostic bundles.
- Public keys and private keys are different; still consider whether any identifier or key material can be linked to a live deployment before publishing it.
- If sensitive data is accidentally committed, do not merely delete it in a later commit: treat it as disclosed, notify the operator, rotate/revoke as appropriate, and follow the repository's incident procedure.

## Progress and status discipline

- Check off plan items only when their acceptance criteria are met and the report/PR/test evidence is linked.
- Use explicit statuses such as `confirmed`, `not reproduced`, `hypothesis`, `blocked`, `deferred`, and `fixed — awaiting verification`.
- A code change is not proof that the defect is fixed. Link the regression test and record its actual result.
- A static code review does not prove production behavior. Label live-fleet checks separately.
- Keep historical reports immutable in meaning. If later code changes invalidate a finding, add a dated follow-up or update the status with a link to new evidence; do not silently rewrite what was known at the time.

## Recommended workflow

1. Create an audit report from [the template](audits/TEMPLATE.md).
2. Record the exact base commit before inspection.
3. Collect only necessary, sanitized evidence.
4. Verify every finding against source and/or a reproducible test.
5. Save the EN and RU report pair under the paths above.
6. Link the report from the relevant plan checkbox, issue/PR, and documentation index.
7. After remediation, add verification evidence and update the finding status; do not mark the plan complete until acceptance criteria pass.

## Related documents

- [Update-system implementation plan](UPDATE-SYSTEM-IMPLEMENTATION-PLAN.md)
- [Audit template](audits/TEMPLATE.md)
- [RU policy](ru/AUDIT-ARTIFACTS.md)
- [RU template](ru/audits/TEMPLATE.md)
