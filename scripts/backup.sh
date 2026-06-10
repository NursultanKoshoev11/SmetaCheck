#!/usr/bin/env bash
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/smetacheck}"
RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-14}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
APP_DIR="${APP_DIR:-/opt/smetacheck}"

mkdir -p "$BACKUP_DIR"

if [ ! -f "$APP_DIR/.env" ]; then
  echo "Missing $APP_DIR/.env" >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "$APP_DIR/.env"
set +a

DB_FILE="$BACKUP_DIR/postgres_$STAMP.sql.gz"
UPLOADS_FILE="$BACKUP_DIR/uploads_$STAMP.tar.gz"
REPORTS_FILE="$BACKUP_DIR/reports_$STAMP.tar.gz"

echo "Creating database backup: $DB_FILE"
docker compose --env-file "$APP_DIR/.env" -f "$APP_DIR/docker-compose.yml" exec -T postgres pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB" | gzip > "$DB_FILE"

if [ -d "${UPLOAD_DIR:-/var/lib/smetacheck/uploads}" ]; then
  echo "Creating uploads backup: $UPLOADS_FILE"
  tar -czf "$UPLOADS_FILE" -C "$(dirname "$UPLOAD_DIR")" "$(basename "$UPLOAD_DIR")"
fi

if [ -d "${REPORT_DIR:-/var/lib/smetacheck/reports}" ]; then
  echo "Creating reports backup: $REPORTS_FILE"
  tar -czf "$REPORTS_FILE" -C "$(dirname "$REPORT_DIR")" "$(basename "$REPORT_DIR")"
fi

find "$BACKUP_DIR" -type f -mtime +"$RETENTION_DAYS" -delete

echo "Backup completed at $STAMP"
