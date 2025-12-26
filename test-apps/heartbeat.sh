#!/bin/bash

echo "Shell script starting... PID: $$"
echo "Current time: $(date)"

counter=0
while true; do
    counter=$((counter + 1))
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] Heartbeat #$counter"
    sleep 3

    # Check if we should exit
    if [[ -f "/tmp/stop_script" ]]; then
        echo "Stop signal received, shutting down..."
        rm -f /tmp/stop_script
        exit 0
    fi
done