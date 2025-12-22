#!/usr/bin/env bash

set -u

SERVICES=("scheduler" "worker-α" "worker-β" "io-daemon" "net-agent")
LEVELS=("INFO" "DEBUG" "WARN" "ERROR")
START_TIME=$(date +%s)
ITER=0

rand() {
  echo $((RANDOM % $1))
}

while true; do
  ((ITER++))

  svc=${SERVICES[$(rand ${#SERVICES[@]})]}
  lvl=${LEVELS[$(rand ${#LEVELS[@]})]}

  now=$(date +"%Y-%m-%d %H:%M:%S.%3N")
  uptime=$(( $(date +%s) - START_TIME ))

  cpu=$(printf "%.1f" "$(echo "scale=2; (RANDOM%800)/10" | bc)")
  mem=$((RANDOM % 4096 + 128))
  jobs=$((RANDOM % 42))

  msg="heartbeat"
  [[ $lvl == "WARN" ]] && msg="latency spike detected"
  [[ $lvl == "ERROR" ]] && msg="task failed, retry queued"

  printf '%s | %-9s | %-5s | uptime=%ss cpu=%s%% mem=%sMB jobs=%s msg="%s"\n' \
    "$now" "$svc" "$lvl" "$uptime" "$cpu" "$mem" "$jobs" "$msg"

  sleep "$(printf "0.%02d" $((RANDOM % 40 + 5)))"
done

