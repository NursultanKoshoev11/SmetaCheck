#!/bin/sh
set -eu

COMPOSE_FILE=${COMPOSE_FILE:-docker-compose.production.yml}
BACKUP_DIR=${BACKUP_DIR:-./backups}
RETENTION_DAYS=${BACKUP_RETENTION_DAYS:-14}
TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)
TARGET="$BACKUP_DIR/$TIMESTAMP"

mkdir -p "$TARGET"

docker compose -f "$COMPOSE_FILE" exec -T postgres sh -c \
  'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --format=custom --no-owner --no-acl' \
  > "$TARGET/postgres.dump"

docker run --rm \
  -v smetacheck_uploads:/source:ro \
  -v "$(pwd)/$TARGET:/backup" \
  alpine:3.20 sh -c 'cd /source && tar -czf /backup/uploads.tgz .'

docker run --rm \
  -v smetacheck_reports:/source:ro \
  -v "$(pwd)/$TARGET:/backup" \
  alpine:3.20 sh -c 'cd /source && tar -czf /backup/reports.tgz .'

sha256sum "$TARGET"/* > "$TARGET/SHA256SUMS"
find "$BACKUP_DIR" -mindepth 1 -maxdepth 1 -type d -mtime "+$RETENTION_DAYS" -exec rm -rf {} +
echo "Backup created: $TARGET"
