# Post-review 0.9.318

- **F1** `GET /api/nvr/clip` exempt from session gate (one-shot token share); mint still session + PathUnderRoot
- **F3** explicit `requireSession` on release GET (local, detail, git-tags, build-status)
- **F4** doctor warns if `go2rtc.yaml` / `nvr/secrets.json` group/other readable
