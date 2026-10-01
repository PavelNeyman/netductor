# Control + Media topics (implemented 0.9.151)

## Bootstrap topics
| Key | Name | Purpose |
|-----|------|---------|
| menu | 🎛 Control | Singleton operator hub (edit-in-place) |
| media | 📎 Media | QR photos, SR Config documents |
| alerts / warnings / service / updates | … | Notifications only |

## Hub singleton
- State: `/var/lib/netductor/tg/hub_msg.json` (`message_id`, `thread_id`)
- `reply()` always prefers editing that message
- `/menu` and main navigation do not spawn a second hub when hub id is known

## Media
- `sendRichWithPhoto` / `sendDocumentFile` attach **media** thread
- Access: text card stays on Control; QR is sent to Media

## Client tips
1. BotFather: Threaded Mode ON, disallow user-created topics
2. Open **🎛 Control** for the menu — avoid the aggregate «All / General» view
3. After upgrade: Topics → Recreate if menu/media missing

## Limits
Telegram may still show cross-topic items in «All». Perfect one-message-only chats are impossible with media; goal is one **control** message + ephemeral media in another topic.
