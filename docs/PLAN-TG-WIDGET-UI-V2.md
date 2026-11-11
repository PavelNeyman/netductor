# TG Widget UI v2 (shell A+B)

## Constraints
- HTML ≤32KiB, zlib ≤4KiB; chat_data ≤1024B; height ≤600px; no scroll; no network
- Tabs/segments = our buttons, not Telegram native controls
- Charts = canvas or CSS bars from compact numbers in chat_data

## IA
| Tab | Content |
|-----|---------|
| Status | health rows + bars + channel/netmap/incident |
| Users | list → card (toggles, quota, QR via chat) |
| Fleet | nodes, secondaries, updates hub |
| Tools | edge/media/data/git/mtls/presets |
| Operator | stack, backup, session, doctor |

## Updates hub (`updates`)
Segments: **Primary | Secondary | Edge**
- Primary: version, floor, stack apply entry
- Secondary: online/version list, upgrade act
- Edge: agent versions / pending

## Visual patterns
- Top tab bar (always)
- Segment control on multi-layer screens
- Toggle-looking acts (primary = on)
- Key/value rows; CSS progress bars
- Optional sparkline later (canvas, ≤24 samples)

## Rollout
1. Shell CSS/JS + tabs
2. Updates hub
3. Formatters (no raw JSON dumps)
4. Toggle polish + i18n
5. Sparkline on status (optional)
