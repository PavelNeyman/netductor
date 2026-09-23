# TUI integrity rule

**Locked:** the workstation TUI must stay **self-contained**.

1. Never exit alt-screen / Bubble Tea to run huh, external prompts, or shell wizards for operator actions.
2. Never leave a bare terminal for results — use **framed** log panes (header + border + help chips).
3. Deploy, Tools, Ops, forms: collect input with in-TUI fields; show results inside the same window.
4. `tea.Quit` is only for explicit **Quit** / Ctrl+C.
5. Do not add new `runTUI` post-quit huh hooks.

See also AGENT_HANDOFF.
