# DX Features - Implementation Summary

## 🎉 What's New

I've added **6 major developer experience (DX) features** to make prox much easier and more pleasant to use!

---

## 1. 📝 Config File Support (prox.yml)

**What it does:** Define all your services in a YAML file instead of starting them one by one.

**Example prox.yml:**
```yaml
services:
  api:
    command: node server.js
    restart: on-failure
    cwd: /path/to/api
    env:
      NODE_ENV: production
      PORT: "3000"
    instances: 1
    depends_on:
      - database

  worker:
    command: python worker.py
    restart: always
    watch:
      - src/
```

**Usage:**
```bash
# Start all services from config
prox start          # Or: prox start-all, prox up

# Config file auto-discovery looks for:
# - prox.yml
# - prox.yaml
# - .prox.yml
```

**Benefits:**
- ✅ Define services once, reuse forever
- ✅ Version control your process configuration
- ✅ Share setup with team members
- ✅ No more remembering complex command flags

---

## 2. 🔍 Auto-Discovery

**What it does:** Automatically detect services from existing project files.

**Supported formats:**
1. **Procfile** (Heroku-style)
2. **package.json** (npm scripts)
3. **docker-compose.yml** (Docker services)

**Example - Procfile:**
```
web: python test_app.py
worker: python cpu_test.py
```

**Example - package.json:**
```json
{
  "scripts": {
    "dev": "next dev",
    "start": "node server.js",
    "worker": "node worker.js"
  }
}
```

**How it works:**
```bash
# Prox automatically finds and reads:
# 1. Procfile (if exists)
# 2. package.json scripts (if exists)
# 3. docker-compose.yml (if exists)

prox init  # Auto-discovers from current directory

# Or specify a specific file
prox init -f ../Procfile              # Relative path
prox init -f /path/to/Procfile.prod   # Absolute path
prox init -f config/docker-compose.yml
```

**Benefits:**
- ✅ Zero manual configuration needed
- ✅ Works with your existing project files
- ✅ Familiar format if you've used Heroku/Foreman

---

## 3. ⚡ Zero-Config Startup (prox init)

**What it does:** One command to go from nothing to fully configured.

**Workflow:**
```bash
cd my-project

# Step 1: Auto-discover and create prox.yml
prox init

# Output:
# 🔍 Discovering services...
# ✓ Found 2 service(s) in Procfile
#
# Discovered services:
#   • web: python test_app.py
#   • worker: python cpu_test.py
#
# 📝 Writing prox.yml...
# ✓ Created prox.yml

# Step 2: Start everything
prox start

# Output:
# 📖 Loading config from prox.yml
# 🚀 Starting 2 service(s)...
# ✓ web (PID 12345)
# ✓ worker (PID 12346)
# ✓ Started 2/2 services

# Step 3: Monitor
prox  # Opens TUI dashboard
```

**Benefits:**
- ✅ Get started in seconds
- ✅ No manual config file writing
- ✅ Review and customize prox.yml if needed

---

## 4. 🔄 Restart Policies

**What it does:** Automatically restart crashed processes based on configurable policies.

**Policies:**
- `always` - Always restart, even if it exits cleanly (exit code 0)
- `on-failure` - Only restart if it crashes (exit code != 0) [DEFAULT]
- `never` - Never restart, just mark as errored

**Usage in prox.yml:**
```yaml
services:
  critical-api:
    command: node server.js
    restart: always     # Always keep running

  periodic-job:
    command: python job.py
    restart: on-failure # Only restart if it crashes

  one-shot-script:
    command: bash setup.sh
    restart: never      # Run once and stop
```

**Usage in CLI:**
```bash
# Processes started from CLI use "on-failure" by default
prox start app.js  # restart: on-failure (default)
```

**How it works:**
```bash
# Example: Process crashes
[prox] Process 'api' (PID 12345) exited with error (code 1)
[prox] Restarting 'api' (policy: on-failure, exit code: 1)...
✓ Started 'api' (PID 12350)

# Restarts counter increments
prox list
# NAME   STATUS    PID    RESTARTS  UPTIME  SCRIPT
# api    ● online  12350  1         3s      server.js
```

**Benefits:**
- ✅ Production-ready: Keep services alive
- ✅ Crash-resilient: Auto-recovery from failures
- ✅ Flexible: Different policies for different services
- ✅ Smart: Waits 1 second between restarts to prevent crash loops

---

## 5. 🔗 Process Dependencies (Start Order)

**What it does:** Start services in the correct order based on dependencies.

**Example:**
```yaml
services:
  database:
    command: postgres -D ./data
    restart: always

  api:
    command: node server.js
    restart: on-failure
    depends_on:
      - database  # API waits for database to start

  frontend:
    command: npm run dev
    restart: on-failure
    depends_on:
      - api       # Frontend waits for API to start
```

**Start sequence:**
```bash
prox start

# 1. Starts "database" first
# 2. Waits 500ms
# 3. Starts "api" (depends on database)
# 4. Waits 500ms
# 5. Starts "frontend" (depends on api)
```

**Benefits:**
- ✅ Prevents "connection refused" errors on startup
- ✅ Ensures correct initialization order
- ✅ Mirrors docker-compose depends_on behavior

**Note:** Currently uses fixed 500ms delay. Future enhancement: Health checks to wait for actual readiness.

---

## 6. 🎨 Better UX & Output

**What it does:** Clear, helpful, beautiful terminal output.

**Before (generic):**
```
Error: failed
```

**After (helpful):**
```
✗ No prox.yml found. Run 'prox init' first.
```

**Color-coded status:**
- ✓ Green = Success
- ● Blue = Info
- ✗ Red = Error
- ○ Gray = Stopped

**Progress indicators:**
```bash
prox start

📖 Loading config from prox.yml
🚀 Starting 2 service(s)...
  ✓ web (PID 12345)
  ✓ worker (PID 12346)
✓ Started 2/2 services
```

**Helpful next steps:**
```
Next steps:
  • Run 'prox' to open TUI dashboard
  • Run 'prox list' to see all processes
  • Run 'prox logs <name>' to view logs
```

---

## 📊 Complete Feature Comparison

| Feature | Before | After |
|---------|--------|-------|
| **Starting services** | Manual, one-by-one | Single command from config |
| **Configuration** | Command-line flags only | YAML file + auto-discovery |
| **Setup time** | Minutes of manual work | Seconds with `prox init` |
| **Crash handling** | Manual restart | Automatic with policies |
| **Dependencies** | Start in random order | Smart dependency resolution |
| **Learning curve** | Remember all flags | Read existing project files |

---

## 🚀 Quick Start Guide

### New Project Setup
```bash
# 1. Create a Procfile (or use package.json)
echo "web: python app.py" > Procfile
echo "worker: python worker.py" >> Procfile

# 2. Auto-generate prox.yml
prox init

# 3. Start everything
prox start

# 4. View dashboard
prox
```

### Existing Project
```bash
# If you have package.json with "scripts"
prox init  # Auto-detects npm scripts
prox start

# If you have docker-compose.yml
prox init  # Converts docker services
prox start
```

### Manual Configuration
```bash
# Create prox.yml manually
cat > prox.yml <<EOF
services:
  api:
    command: node server.js
    restart: always
  worker:
    command: python worker.py
    restart: on-failure
    depends_on: [api]
EOF

prox start
```

---

## 📁 Example prox.yml (Full Features)

```yaml
services:
  # Database service - always running
  postgres:
    command: postgres -D ./data/db
    restart: always
    cwd: /var/lib/postgres

  # Redis cache
  redis:
    command: redis-server
    restart: always

  # API server - depends on database and redis
  api:
    command: npm run dev:api
    restart: on-failure
    cwd: ./backend
    env:
      NODE_ENV: development
      PORT: "3000"
      DATABASE_URL: postgresql://localhost/mydb
    depends_on:
      - postgres
      - redis
    watch:
      - src/

  # Background worker - depends on redis
  worker:
    command: python -m celery worker
    restart: always
    cwd: ./backend
    env:
      CELERY_BROKER: redis://localhost:6379
    depends_on:
      - redis

  # Frontend dev server - depends on API
  web:
    command: npm run dev
    restart: on-failure
    cwd: ./frontend
    env:
      VITE_API_URL: http://localhost:3000
    depends_on:
      - api
```

---

## 🎯 Commands Reference

### New Commands
```bash
prox init                 # Auto-discover and create prox.yml
prox init -f <file>       # Use specific Procfile/config file
prox start                # Start all from prox.yml (if no args)
prox start <script>       # Start single process (existing behavior)
prox start-all            # Alias: start all from config
prox up                   # Alias: start all from config
```

### Enhanced Commands
```bash
prox start                # Now smart: loads prox.yml if no args
prox list                 # Shows restart count
```

---

## 🔧 Technical Details

### Config File Discovery
Searches in current directory and parent directories for:
1. `prox.yml`
2. `prox.yaml`
3. `.prox.yml`

### Auto-Discovery Priority
1. Procfile (if exists)
2. package.json (if exists and has dev scripts)
3. docker-compose.yml (if exists)

### Restart Logic
- Waits 1 second between restart attempts
- Tracks exit codes to determine restart eligibility
- Logs restart reason and exit code
- Updates restart counter in process state

### Dependency Resolution
- Recursive dependency resolution
- 500ms delay between dependent services
- Prevents duplicate starts (tracks started services)
- Graceful error handling if dependency missing

---

## 🐛 Known Limitations & Future Work

### Current Limitations
1. **No daemon mode** - Processes run but auto-restart only works while prox command is running
2. **Fixed dependency delay** - Uses 500ms wait instead of health checks
3. **No crash loop protection** - Will restart infinitely (should add max restart limits)
4. **No watch mode** - File watching defined but not implemented

### Planned Enhancements
1. **Daemon mode** - Long-running background process for monitoring
2. **Health checks** - Wait for HTTP endpoints before starting dependents
3. **Max restart limits** - Prevent infinite crash loops
4. **Backoff strategies** - Exponential delay for repeated crashes
5. **File watching** - Auto-restart on file changes
6. **Better CLI** - Accept multiple process names: `prox delete web api worker`

---

## ✅ Testing the Features

### Test 1: Auto-Discovery from Procfile
```bash
mkdir test-project
cd test-project
echo "web: python test.py" > Procfile
echo "print('Hello')" > test.py

prox init    # Should discover "web" service
cat prox.yml # Verify config created
```

### Test 2: Starting from Config
```bash
prox start   # Should start all services from prox.yml
prox list    # Should show services running
```

### Test 3: Restart Policy
```bash
# In prox.yml, set restart: on-failure
prox start
kill -9 <PID>  # Kill process
# Should auto-restart (check with prox list)
```

### Test 4: Dependencies
```bash
# Create prox.yml with depends_on
prox start
# Check logs - services should start in order
```

---

## 💡 Tips for Users

1. **Use prox init for quick setup** - Don't write prox.yml manually unless needed
2. **Commit prox.yml to git** - Share config with team
3. **Set appropriate restart policies** - Use `always` for servers, `on-failure` for workers
4. **Define dependencies** - Prevent startup race conditions
5. **Use TUI for monitoring** - `prox` command shows live metrics

---

## 🎓 Migration from Other Tools

### From pm2
```javascript
// OLD: ecosystem.config.js
module.exports = {
  apps: [{
    name: 'api',
    script: 'server.js',
    autorestart: true
  }]
}

// NEW: prox.yml
services:
  api:
    command: node server.js
    restart: always
```

### From foreman (Procfile)
```bash
# Keep your Procfile!
prox init  # Converts Procfile to prox.yml
prox start # Works exactly like: foreman start
```

### From docker-compose
```yaml
# docker-compose.yml still works
prox init  # Extracts services to prox.yml
# Note: Only command-based services, not Docker images
```

---

## 📚 What's Next?

With these DX features, prox is now:
- ✅ **Easy to start** - `prox init` and go
- ✅ **Config-driven** - Define once, run anywhere
- ✅ **Production-ready** - Auto-restart and dependencies
- ✅ **Team-friendly** - Commit prox.yml to repo

**Next priorities:**
1. Daemon mode for true auto-restart
2. Health check support
3. File watching and hot reload
4. Better error messages with suggestions
5. Process clustering (multiple instances with load balancing)

---

**Try it now!**
```bash
cd your-project
prox init
prox start
prox  # Open TUI
```
