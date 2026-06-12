#!/bin/sh
set -eu
umask 077

COMPOSE_FILE=${COMPOSE_FILE:-docker-compose.production.yml}
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-smetacheck}
BACKUP_DIR=${BACKUP_DIR:-/var/backups/smetacheck}
RETENTION_DAYS=${BACKUP_RETENTION_DAYS:-14}
REQUIRE_ENCRYPTION=${BACKUP_REQUIRE_ENCRYPTION:-true}
ENCRYPTION_PASSWORD_FILE=${BACKUP_ENCRYPTION_PASSWORD_FILE:-}
BACKUP_S3_URI=${BACKUP_S3_URI:-}
TIMESTAMP=$(date -u +%Y%m%dT%H%M%SZ)

case "$RETENTION_DAYS" in
  ''|*[!0-9]*) echo "BACKUP_RETENTION_DAYS must be a non-negative integer" >&2; exit 1 ;;
esac

for command_name in docker sha256sum tar; do
  command -v "$command_name" >/dev/null 2>&1 || {
    echo "Required command is missing: $command_name" >&2
    exit 1
  }
done

test -f "$COMPOSE_FILE" || {
  echo "Compose file not found: $COMPOSE_FILE" >&2
  exit 1
}

mkdir -p "$BACKUP_DIR"
BACKUP_ROOT=$(cd "$BACKUP_DIR" && pwd -P)
CURRENT_ROOT=$(pwd -P)
case "$BACKUP_ROOT" in
  ''|/) echo "Refusing to use unsafe backup directory: $BACKUP_ROOT" >&2; exit 1 ;;
esac
if [ "$BACKUP_ROOT" = "$CURRENT_ROOT" ] || { [ -n "${HOME:-}" ] && [ "$BACKUP_ROOT" = "$HOME" ]; }; then
  echo "Refusing to use the current directory or home directory as BACKUP_DIR" >&2
  exit 1
fi

if [ "$REQUIRE_ENCRYPTION" = "true" ]; then
  command -v openssl >/dev/null 2>&1 || {
    echo "openssl is required when BACKUP_REQUIRE_ENCRYPTION=true" >&2
    exit 1
  }
  [ -n "$ENCRYPTION_PASSWORD_FILE" ] && [ -s "$ENCRYPTION_PASSWORD_FILE" ] || {
    echo "BACKUP_ENCRYPTION_PASSWORD_FILE must point to a non-empty protected file" >&2
    exit 1
  }
fi

WORK_DIR="$BACKUP_ROOT/.partial-$TIMESTAMP-$$"
FINAL_DIR="$BACKUP_ROOT/$TIMESTAMP"
[ ! -e "$FINAL_DIR" ] || {
  echo "Backup target already exists: $FINAL_DIR" >&2
  exit 1
}
mkdir "$WORK_DIR"
cleanup_partial() {
  rm -rf "$WORK_DIR"
}
trap cleanup_partial EXIT HUP INT TERM

VOLUME_PREFIX=$COMPOSE_PROJECT_NAME
UPLOADS_VOLUME="${VOLUME_PREFIX}_uploads"
REPORTS_VOLUME="${VOLUME_PREFIX}_reports"

echo "Creating PostgreSQL backup..."
docker compose -f "$COMPOSE_FILE" exec -T postgres sh -c \
  'pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" --format=custom --compress=6 --no-owner --no-acl' \
  > "$WORK_DIR/postgres.dump"
[ -s "$WORK_DIR/postgres.dump" ] || {
  echo "PostgreSQL backup is empty" >&2
  exit 1
}

echo "Creating upload volume backup..."
docker run --rm \
  -v "$UPLOADS_VOLUME:/source:ro" \
  -v "$WORK_DIR:/backup" \
  alpine:3.20 sh -c 'cd /source && tar -czf /backup/uploads.tgz .'

echo "Creating report volume backup..."
docker run --rm \
  -v "$REPORTS_VOLUME:/source:ro" \
  -v "$WORK_DIR:/backup" \
  alpine:3.20 sh -c 'cd /source && tar -czf /backup/reports.tgz .'

if [ "$REQUIRE_ENCRYPTION" = "true" ]; then
  echo "Encrypting backup files..."
  for source_file in postgres.dump uploads.tgz reports.tgz; do
    openssl enc -aes-256-cbc -pbkdf2 -iter 200000 -salt \
      -in "$WORK_DIR/$source_file" \
      -out "$WORK_DIR/$source_file.enc" \
      -pass "file:$ENCRYPTION_PASSWORD_FILE"
    rm -f "$WORK_DIR/$source_file"
  done
fi

(
  cd "$WORK_DIR"
  sha256sum postgres.dump* uploads.tgz* reports.tgz* > SHA256SUMS
)
chmod -R go-rwx "$WORK_DIR"

mv "$WORK_DIR" "$FINAL_DIR"
trap - EXIT HUP INT TERM

if [ -n "$BACKUP_S3_URI" ]; then
  command -v aws >/dev/null 2>&1 || {
    echo "aws CLI is required when BACKUP_S3_URI is configured" >&2
    exit 1
  }
  S3_TARGET=${BACKUP_S3_URI%/}/$TIMESTAMP/
  echo "Uploading backup to $S3_TARGET"
  aws s3 cp "$FINAL_DIR/" "$S3_TARGET" --recursive --only-show-errors
fi

find "$BACKUP_ROOT" \
  -mindepth 1 -maxdepth 1 -type d \
  -name '20??????T??????Z' \
  -mtime "+$RETENTION_DAYS" \
  -exec rm -rf -- {} +

echo "Backup created successfully: $FINAL_DIR"
