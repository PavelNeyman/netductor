# Media / QR topics (design)

## Goal
Keep the **Control** menu topic as a single editable hub. Large payloads (QR photos, SR config documents, long apply logs) should not permanently sit under the menu.

## Options

| Option | Behavior | Pros | Cons |
|--------|----------|------|------|
| **A. Ephemeral in Control** | Send photo/doc in menu topic; delete after N min or when user taps Back | Simple | Still flashes in menu thread |
| **B. Topic `📎 Media`** | All QR/files go to dedicated topic; menu only gets a one-line “sent to Media” | Menu stays clean | User switches topic to open QR |
| **C. Topic per kind** | `QR`, `Files`, `Logs` | Clear | Topic sprawl |
| **D. Alerts-style** | QR only on demand in Media; Access screen stays text+links in Control | Best for “static menu” | Extra tap for image |

**Recommendation:** **B** — one `📎 Media` bootstrap topic + Control hub. Access card in Control keeps **links + client buttons**; QR image optionally “show QR → Media topic”. Documents (SR Config) always to Media.

## Hub singleton (related)
- Store `hub_msg_id` + `hub_thread_id` (Control).
- Navigation = `editMessage`; prompts = edit same hub; never `send` a second menu.
