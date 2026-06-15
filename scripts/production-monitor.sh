#!/usr/bin/env sh
set -eu
APP_DIR="${APP_DIR:-/opt/smetacheck}"
READY_URL="${READY_URL:-https://api.smetacheck.kg/ready}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/smetacheck}"
MAX_BACKUP_AGE_HOURS="${MAX_BACKUP_AGE_HOURS:-24}"
DISK_WARN_PERCENT="${DISK_WARN_PERCENT:-80}"
failures=""
add_failure(){ failures="${failures}\n- $1"; }
if ! curl -fsS --max-time 10 "$READY_URL" >/dev/null; then add_failure "API readiness check failed: $READY_URL"; fi
if [ -d "$APP_DIR" ]; then cd "$APP_DIR"; docker compose ps --status running >/dev/null 2>&1 || add_failure "Docker Compose status check failed in $APP_DIR"; fi
disk_use="$(df -P / | awk 'NR==2 {gsub(/%/,"",$5); print $5}')"
if [ -n "$disk_use" ] && [ "$disk_use" -ge "$DISK_WARN_PERCENT" ]; then add_failure "Root disk usage is ${disk_use}%"; fi
if [ -d "$BACKUP_DIR" ]; then
  newest="$(find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d -printf '%T@\n' 2>/dev/null | sort -nr | head -n 1)"
  if [ -z "$newest" ]; then add_failure "No backup directories found in $BACKUP_DIR"; else
    now="$(date +%s)"; newest_i="$(printf '%.0f' "$newest")"; age="$(((now-newest_i)/3600))"
    if [ "$age" -gt "$MAX_BACKUP_AGE_HOURS" ]; then add_failure "Newest backup is ${age}h old"; fi
  fi
else add_failure "Backup directory does not exist: $BACKUP_DIR"; fi
if [ -n "$failures" ]; then printf 'SmetaCheck monitor failed:%s\n' "$failures" >&2; exit 1; fi
printf 'SmetaCheck monitor passed\n'
