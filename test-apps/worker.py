#!/usr/bin/env python3

import time
import os
import sys


def main():
    print(f"Python worker starting... PID: {os.getpid()}", flush=True)

    counter = 0
    while True:
        counter += 1
        print(
            f"[{time.strftime('%Y-%m-%d %H:%M:%S')}] Processing item #{counter}",
            flush=True,
        )

        # Simulate some work
        time.sleep(2)

        # Check for graceful shutdown signal
        try:
            # In a real app, you might check for signals here
            pass
        except KeyboardInterrupt:
            print("Shutting down gracefully...", flush=True)
            sys.exit(0)


if __name__ == "__main__":
    main()
