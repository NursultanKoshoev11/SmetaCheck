#!/bin/sh
set -eu

APP_URL=${APP_URL:-https://smetacheck.kg}
API_URL=${API_URL:-https://api.smetacheck.kg}

check() {
  name=$1
  url=$2
  expected=$3
  code=$(curl --silent --show-error --location --output /tmp/smetacheck-smoke-body --write-out '%{http_code}' "$url")
  if [ "$code" != "$expected" ]; then
    echo "$name failed: expected HTTP $expected, got $code" >&2
    cat /tmp/smetacheck-smoke-body >&2 || true
    exit 1
  fi
  echo "$name: HTTP $code"
}

check "Frontend" "$APP_URL/" 200
check "API health" "$API_URL/health" 200
check "API readiness" "$API_URL/ready" 200

headers=$(curl --silent --show-error --head "$APP_URL/")
echo "$headers" | grep -qi '^strict-transport-security:' || { echo "Missing HSTS header" >&2; exit 1; }
echo "$headers" | grep -qi '^x-content-type-options: nosniff' || { echo "Missing nosniff header" >&2; exit 1; }

echo "Production smoke test passed"
