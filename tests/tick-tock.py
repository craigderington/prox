#!/usr/bin/env python3

import sys
import time
import random
from datetime import datetime

PROCS = [
    "indexer",
    "event-loop",
    "gc-worker",
    "disk-writer",
    "api-gateway"
]

STATES = ["idle", "running", "blocked", "restarting"]
LEVELS = ["INFO", "DEBUG", "WARN", "ERROR"]

start_time = time.time()
tick = 0

def log(proc, level, state, msg, extra=""):
    ts = datetime.now().strftime("%Y-%m-%d %H:%M:%S.%f")[:-3]
    uptime = int(time.time() - start_time)
    sys.stdout.write(
        f"{ts} | {proc:<12} | {level:<5} | state={state:<10} | up={uptime:>4}s | {msg} {extra}\n"
    )
    sys.stdout.flush()

while True:
    tick += 1
    proc = random.choice(PROCS)
    state = random.choice(STATES)
    level = random.choices(
        LEVELS, weights=[60, 25, 10, 5], k=1
    )[0]

    cpu = round(random.uniform(0.2, 88.0), 1)
    mem = random.randint(64, 8192)
    qlen = random.randint(0, 128)

    msg = "tick"
    if level == "WARN":
        msg = "queue depth rising"
    elif level == "ERROR":
        msg = "unexpected exit, respawning"

    log(
        proc,
        level,
        state,
        msg,
        f"cpu={cpu}% mem={mem}MB q={qlen}"
    )

    time.sleep(random.uniform(0.08, 0.45))

