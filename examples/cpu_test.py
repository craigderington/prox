#!/usr/bin/env python3
"""Test app that generates CPU and memory usage for metrics testing"""
import time
import sys
import math

print("CPU Test App Started - Generating load...")
sys.stdout.flush()

# Allocate some memory
data = [0] * (10 * 1024 * 1024)  # ~40MB

counter = 0
while True:
    # Do some CPU-intensive work
    for i in range(100000):
        _ = math.sqrt(i) * math.sin(i) * math.cos(i)

    counter += 1
    if counter % 10 == 0:
        print(f"Iteration {counter} - CPU load active")
        sys.stdout.flush()

    time.sleep(0.1)  # Brief pause
