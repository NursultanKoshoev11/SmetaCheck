#!/bin/sh
set -eu

COMPOSE_FILE=${COMPOSE_FILE:-docker-compose.production.yml}
APP_URL=${APP_URL:-https://app.smetacheck.kg}
API_URL=${API_URL:-https://api.smetacheck.kg}
RELEASE_SHA=${RELEASE_SHA:-$(git rev-parse HEAD)}
STATE_DIR=${STATE_DIR:-/var/lib/smetacheck/releases}

mkdir -p "$STATE_DIR"
PREVIOUS_SHA=$(git rev-parse HEAD)
printf '%s\n' "$PREVIOUS_SHA" > "$STATE_DIR/previous-sha"
printf '%s\n' "$RELEASE_SHA" > "$STATE_DIR/pending-sha"

./scripts/backup-production.sh

git fetch --all --tags
git checkout --detach "$RELEASE_SHA"

docker compose -f "$COMPOSE_FILE" build --pull
docker compose -f "$COMPOSE_FILE" up -d --remove-orphans

APP_URL="$APP_URL" API_URL="$API_URL" ./scripts/smoke-production.sh
printf '%s\n' "$RELEASE_SHA" > "$STATE_DIR/current-sha"
rm -f "$STATE_DIR/pending-sha"
echo "Production deployment completed: $RELEASE_SHA"
