#!/bin/bash
# /usr/local/bin/outpost-idle-stop

CONFIG="/var/lib/outpost/idle-stop-minutes"
ACTIVITY="/var/lib/outpost/last-activity"
NOW=$(date +%s)

# idle-stop disabled?
[[ -f "$CONFIG" ]] || exit 0
MINUTES=$(cat "$CONFIG")
[[ "$MINUTES" =~ ^[0-9]+$ && "$MINUTES" -gt 0 ]] || exit 0

# someone using the box? reset idle clock
# interactive sessions with a TTY (real SSH work)
if who | awk '{print $2}' | grep -q '^pts'; then
  echo "$NOW" > "$ACTIVITY"
  exit 0
fi

# first run: start counting from now
[[ -f "$ACTIVITY" ]] || { echo "$NOW" > "$ACTIVITY"; exit 0; }

LAST=$(cat "$ACTIVITY")
[[ "$LAST" =~ ^[0-9]+$ && "$LAST" -gt 0 && "$LAST" -le "$NOW" ]] || exit 0
IDLE_SEC=$((NOW - LAST))
LIMIT_SEC=$((MINUTES * 60))

if (( IDLE_SEC >= LIMIT_SEC )); then
  /usr/bin/logger -t outpost-idle-stop "idle for ${IDLE_SEC}s (limit ${LIMIT_SEC}s), shutting down"
  /usr/sbin/shutdown -h now
fi