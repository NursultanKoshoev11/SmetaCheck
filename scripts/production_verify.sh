#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "== SmetaCheck production verification =="

echo "1) Go tests"
go test ./...

echo "2) Frontend dependencies and build"
cd frontend
if [ ! -d node_modules ]; then
  npm install
fi
npm run build
cd "$ROOT_DIR"

echo "3) Docker compose syntax"
docker compose config >/dev/null

echo "4) Required production files"
test -f .env.production.example
test -f db/migrations/001_initial_schema.sql
test -f docs/PRODUCTION_DEPLOYMENT.md
test -f scripts/backup.sh
test -f scripts/smoke_test.sh

echo "5) Production placeholder scan"
if grep -R "REPLACE_WITH\|change_me\|smetacheck_change_me" .env.production.example .env.example >/dev/null 2>&1; then
  echo "Templates contain placeholders as expected. Make sure server .env does not contain placeholders."
fi

echo "✅ Static production verification passed."
echo "Next: run API and execute API_BASE=http://localhost:8080 ./scripts/smoke_test.sh"
