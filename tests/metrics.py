#!/usr/bin/env python3

import datetime
import random
import time

events = [
    "CPU high: 85%",
    "Memory low: 20%",
    "Disk write: 5MB/s",
    "Network ping: 50ms",
    "Error: Timeout",
    "Info: Connected",
    "Warning: High temp"
]

while True:
    timestamp = datetime.datetime.now().strftime("%Y-%m-%d %H:%M:%S")
    event = random.choice(events)
    print(f"{timestamp} - {event}")
    time.sleep(1)
