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

echo "1) Health"
curl -fsS "$API_BASE/health" >/dev/null

echo "2) Register"
REGISTER_RESPONSE="$(curl -fsS -X POST "$API_BASE/v1/auth/register" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"full_name\":\"Smoke Test\"}")"
echo "$REGISTER_RESPONSE" | grep -q 'token'

echo "3) Login"
LOGIN_RESPONSE="$(curl -fsS -X POST "$API_BASE/v1/auth/login" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")"
echo "$LOGIN_RESPONSE" | grep -q 'token'

echo "4) Upload estimate"
UPLOAD_RESPONSE="$(curl -fsS -X POST "$API_BASE/v1/estimates/upload" -F "file=@$BASE_FILE")"
echo "$UPLOAD_RESPONSE" | grep -q 'score'
ESTIMATE_ID="$(printf '%s' "$UPLOAD_RESPONSE" | sed -n 's/.*"id":"\([^"]*\)".*/\1/p')"
if [ -z "$ESTIMATE_ID" ]; then
  echo "Cannot parse estimate id from upload response" >&2
  echo "$UPLOAD_RESPONSE" >&2
  exit 1
fi

echo "5) List estimates"
curl -fsS "$API_BASE/v1/estimates" | grep -q "$ESTIMATE_ID"

echo "6) AI summary"
curl -fsS "$API_BASE/v1/ai/estimate-summary/$ESTIMATE_ID" | grep -q 'executive_brief'

echo "7) Compare estimates"
COMPARE_RESPONSE="$(curl -fsS -X POST "$API_BASE/v1/estimates/compare" -F "base=@$BASE_FILE" -F "new=@$NEW_FILE")"
echo "$COMPARE_RESPONSE" | grep -q 'delta_total'

echo "✅ Smoke test passed against $API_BASE"
