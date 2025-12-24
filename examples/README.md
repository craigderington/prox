# Prox Examples

Test the new DX features with these examples.

## Quick Test

```bash
cd simple-app
../../prox init   # Discovers services from Procfile
../../prox start  # Starts all services
../../prox list   # Check status
```

## Available Examples

### simple-app/
Basic Procfile example with web + worker processes.

**Files:**
- `Procfile` - Heroku-style process definition

**Try it:**
```bash
cd simple-app
../../prox init
../../prox start
```

## What to Test

1. **Auto-discovery**: `prox init` reads Procfile automatically
2. **Config creation**: Creates `prox.yml` from discovered services
3. **Start all**: `prox start` launches everything from config
4. **Restart policies**: Kill a process, watch it auto-restart
5. **Dependencies**: Check start order in logs
6. **TUI**: `../../prox` to see live metrics with graphs

## Expected Output

```bash
$ prox init
🔍 Discovering services...
✓ Found 2 service(s) in Procfile

Discovered services:
  • web: python ../../test_app.py
  • worker: python ../../cpu_test.py

📝 Writing prox.yml...
✓ Created prox.yml
```
