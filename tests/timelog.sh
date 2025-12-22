#!/bin/bash

# Simulate dynamic system logs with timestamps and random events

events=("CPU high: 85%" "Memory low: 20%" "Disk write: 5MB/s" "Network ping: 50ms" "Error: Timeout" "Info: Connected" "Warning: High temp")

while true; do
  timestamp=$(date +"%Y-%m-%d %H:%M:%S")
  event=${events[$RANDOM % ${#events[@]}]}
  echo "$timestamp - $event"
  sleep 1
done
