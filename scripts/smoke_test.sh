#!/usr/bin/env bash
set -euo pipefail

API_BASE="${API_BASE:-http://localhost:8080}"
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

EMAIL="test+$(date +%s)@smetacheck.local"
PASSWORD="TestPassword123"
BASE_FILE="$TMP_DIR/base.csv"
NEW_FILE="$TMP_DIR/new.csv"

cat > "$BASE_FILE" <<'CSV'
Наименование,Ед,Количество,Цена,Сумма
Кирпич,шт,1000,12,12000
Цемент,мешок,20,450,9000
Песок,м3,5,1200,6000
CSV

cat > "$NEW_FILE" <<'CSV'
Наименование,Ед,Количество,Цена,Сумма
Кирпич,шт,1200,12,14400
Цемент,мешок,20,450,8500
Арматура,кг,300,65,19500
CSV

echo "1) Health and PostgreSQL readiness"
curl -fsS "$API_BASE/health" >/dev/null
curl -fsS "$API_BASE/ready" | grep -q 'postgresql'

echo "2) Protected endpoint rejects anonymous request"
STATUS="$(curl -sS -o /dev/null -w '%{http_code}' "$API_BASE/v1/estimates")"
test "$STATUS" = "401"

echo "3) Register user in PostgreSQL"
REGISTER_RESPONSE="$(curl -fsS -X POST "$API_BASE/v1/auth/register" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"full_name\":\"Smoke Test\"}")"
echo "$REGISTER_RESPONSE" | grep -q 'token'

echo "4) Login and extract JWT"
LOGIN_RESPONSE="$(curl -fsS -X POST "$API_BASE/v1/auth/login" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
TOKEN="$(printf '%s' "$LOGIN_RESPONSE" | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')"
if [ -z "$TOKEN" ]; then
  echo "Cannot parse JWT from login response" >&2
  echo "$LOGIN_RESPONSE" >&2
  exit 1
fi
AUTH_HEADER="Authorization: Bearer $TOKEN"

echo "5) Verify current user"
curl -fsS "$API_BASE/v1/auth/me" -H "$AUTH_HEADER" | grep -q "$EMAIL"

echo "6) Upload estimate to PostgreSQL-backed account"
UPLOAD_RESPONSE="$(curl -fsS -X POST "$API_BASE/v1/estimates/upload" -H "$AUTH_HEADER" -F "file=@$BASE_FILE")"
echo "$UPLOAD_RESPONSE" | grep -q 'score'
ESTIMATE_ID="$(printf '%s' "$UPLOAD_RESPONSE" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')"
if [ -z "$ESTIMATE_ID" ]; then
  echo "Cannot parse estimate id" >&2
  echo "$UPLOAD_RESPONSE" >&2
  exit 1
fi

echo "7) List only authenticated user's estimates"
curl -fsS "$API_BASE/v1/estimates" -H "$AUTH_HEADER" | grep -q "$ESTIMATE_ID"

echo "8) Load detailed estimate and AI summary"
curl -fsS "$API_BASE/v1/estimates/$ESTIMATE_ID" -H "$AUTH_HEADER" | grep -q 'findings'
curl -fsS "$API_BASE/v1/ai/estimate-summary/$ESTIMATE_ID" -H "$AUTH_HEADER" | grep -q 'executive_brief'

echo "9) Download authenticated report"
curl -fsS "$API_BASE/v1/estimates/$ESTIMATE_ID/report" -H "$AUTH_HEADER" | grep -q 'SmetaCheck'

echo "10) Compare estimates and save result"
COMPARE_RESPONSE="$(curl -fsS -X POST "$API_BASE/v1/estimates/compare" -H "$AUTH_HEADER" -F "base=@$BASE_FILE" -F "new=@$NEW_FILE")"
echo "$COMPARE_RESPONSE" | grep -q 'delta_total'

echo "✅ PostgreSQL smoke test passed against $API_BASE"
