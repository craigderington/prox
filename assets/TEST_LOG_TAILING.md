# Testing Live Log Tailing

## Setup

The `chatty_app.py` is already running and generates a new log line every second:

```bash
./prox list
# Should show 'chatty' as online
```

## Test Live Log Tailing in TUI

1. **Launch the TUI:**
   ```bash
   ./prox
   ```

2. **Select the chatty process:**
   - Use ↑/↓ to select "chatty"
   - Press `ENTER` to open monitor view

3. **Watch the logs auto-scroll:**
   - Top-right panel shows live logs
   - Follow mode is ON by default (shows `[FOLLOW]` in green)
   - Every 1 second, new log entries should appear
   - Logs should auto-scroll to bottom

## Controls

- **Tab** - Switch between panels
- **f** - Toggle follow mode on/off
- **↑/↓** - Manual scroll (disables follow mode)
- **g** - Jump to top of logs
- **G** - Jump to bottom of logs (re-enables follow)
- **Esc** - Back to dashboard

## What You Should See

```
╭─ ▶ Logs: chatty [FOLLOW] ──────────────────╮
│ [INFO] 11:01:56 - Log entry #206           │
│ [INFO] 11:01:58 - Log entry #208           │
│ [ERROR] 11:01:59 - Sample error #209       │  ← New lines
│ [INFO] 11:02:00 - Log entry #210           │    appearing
│ [INFO] 11:02:01 - Log entry #211           │    every 1s
│                                             │
╰─────────────────────────────────────────────╯
```

## Test Follow Mode Toggle

1. Press **Tab** to focus the Logs panel (you'll see "▶ Logs")
2. Press **↓** to manually scroll (follow mode turns OFF)
3. Notice new logs appear but don't auto-scroll
4. Press **G** to jump to bottom (follow mode turns ON)
5. New logs auto-scroll again

## Current Implementation

- **Poll interval:** 1 second
- **Max log lines:** 10,000 (capped to prevent memory issues)
- **Initial load:** Last 100 lines
- **Tailing:** Continuously checks for new lines every second
- **Auto-scroll:** Only when follow mode is ON

## Stop Testing

```bash
./prox stop chatty
```
