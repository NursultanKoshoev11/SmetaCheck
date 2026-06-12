#!/bin/sh
set -eu
umask 077

CONFIG_FILE=${BACKUP_CONFIG_FILE:-.env}

read_env_value() {
  key=$1
  [ -f "$CONFIG_FILE" ] || return 0
  awk -v key="$key" '
    {
      line=$0
      sub(/\r$/, "", line)
      pattern="^[[:space:]]*" key "[[:space:]]*="
      if (line ~ pattern) {
        sub(pattern "[[:space:]]*", "", line)
        first=substr(line,1,1)
        last=substr(line,length(line),1)
        if ((first=="\"" && last=="\"") || (first=="\047" && last=="\047")) {
          line=substr(line,2,length(line)-2)
        }
        print line
        exit
      }
    }
  ' "$CONFIG_FILE"
}

CONFIG_COMPOSE_PROJECT_NAME=$(read_env_value COMPOSE_PROJECT_NAME)
CONFIG_ENCRYPTION_PASSWORD_FILE=$(read_env_value BACKUP_ENCRYPTION_PASSWORD_FILE)

COMPOSE_FILE=${COMPOSE_FILE:-docker-compose.production.yml}
COMPOSE_PROJECT_NAME=${COMPOSE_PROJECT_NAME:-${CONFIG_COMPOSE_PROJECT_NAME:-smetacheck}}
BACKUP_SOURCE=${BACKUP_SOURCE:-}
RESTORE_CONFIRM=${RESTORE_CONFIRM:-}
ENCRYPTION_PASSWORD_FILE=${BACKUP_ENCRYPTION_PASSWORD_FILE:-${CONFIG_ENCRYPTION_PASSWORD_FILE:-}}

[ "$RESTORE_CONFIRM" = "RESTORE_SMETACHECK" ] || {
  echo "Set RESTORE_CONFIRM=RESTORE_SMETACHECK to acknowledge this destructive operation" >&2
  exit 1
}
[ -n "$BACKUP_SOURCE" ] && [ -d "$BACKUP_SOURCE" ] || {
  echo "BACKUP_SOURCE must point to an existing backup directory" >&2
  exit 1
}
test -f "$COMPOSE_FILE" || {
  echo "Compose file not found: $COMPOSE_FILE" >&2
  exit 1
}

for command_name in docker sha256sum tar awk; do
  command -v "$command_name" >/dev/null 2>&1 || {
    echo "Required command is missing: $command_name" >&2
    exit 1
  }
done

SOURCE_ROOT=$(cd "$BACKUP_SOURCE" && pwd -P)
[ -f "$SOURCE_ROOT/SHA256SUMS" ] || {
  echo "SHA256SUMS is missing from backup" >&2
  exit 1
}
(
  cd "$SOURCE_ROOT"
  sha256sum -c SHA256SUMS
)

WORK_DIR=$(mktemp -d "${TMPDIR:-/tmp}/smetacheck-restore.XXXXXX")
cleanup_restore() {
  rm -rf "$WORK_DIR"
}
trap cleanup_restore EXIT HUP INT TERM

copy_or_decrypt() {
  base_name=$1
  if [ -f "$SOURCE_ROOT/$base_name" ]; then
    cp "$SOURCE_ROOT/$base_name" "$WORK_DIR/$base_name"
    return
  fi
  if [ -f "$SOURCE_ROOT/$base_name.enc" ]; then
    command -v openssl >/dev/null 2>&1 || {
      echo "openssl is required to decrypt this backup" >&2
      exit 1
    }
    [ -n "$ENCRYPTION_PASSWORD_FILE" ] && [ -s "$ENCRYPTION_PASSWORD_FILE" ] || {
      echo "BACKUP_ENCRYPTION_PASSWORD_FILE must point to the correct password file" >&2
      exit 1
    }
    openssl enc -d -aes-256-cbc -pbkdf2 -iter 200000 \
      -in "$SOURCE_ROOT/$base_name.enc" \
      -out "$WORK_DIR/$base_name" \
      -pass "file:$ENCRYPTION_PASSWORD_FILE"
    return
  fi
  echo "Backup file is missing: $base_name or $base_name.enc" >&2
  exit 1
}

copy_or_decrypt postgres.dump
copy_or_decrypt uploads.tgz
copy_or_decrypt reports.tgz

UPLOADS_VOLUME="${COMPOSE_PROJECT_NAME}_uploads"
REPORTS_VOLUME="${COMPOSE_PROJECT_NAME}_reports"

echo "Stopping application services..."
docker compose -f "$COMPOSE_FILE" stop api worker frontend

echo "Restoring PostgreSQL database..."
docker compose -f "$COMPOSE_FILE" exec -T postgres sh -c \
  'pg_restore -U "$POSTGRES_USER" -d "$POSTGRES_DB" --clean --if-exists --no-owner --no-acl --exit-on-error' \
  < "$WORK_DIR/postgres.dump"

echo "Restoring uploads volume..."
docker run --rm \
  -v "$UPLOADS_VOLUME:/target" \
  -v "$WORK_DIR:/backup:ro" \
  alpine:3.20 sh -c 'find /target -mindepth 1 -maxdepth 1 -exec rm -rf -- {} + && tar -xzf /backup/uploads.tgz -C /target'

echo "Restoring reports volume..."
docker run --rm \
  -v "$REPORTS_VOLUME:/target" \
  -v "$WORK_DIR:/backup:ro" \
  alpine:3.20 sh -c 'find /target -mindepth 1 -maxdepth 1 -exec rm -rf -- {} + && tar -xzf /backup/reports.tgz -C /target'

echo "Starting application services..."
docker compose -f "$COMPOSE_FILE" up -d api worker frontend caddy

echo "Restore completed. Run scripts/smoke-production.sh immediately."
