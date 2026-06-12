#!/bin/sh
set -eu

APP_URL=${APP_URL:-https://app.smetacheck.kg}
API_URL=${API_URL:-https://api.smetacheck.kg}
TEMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/smetacheck-smoke.XXXXXX")
cleanup_smoke() {
  rm -rf "$TEMP_DIR"
}
trap cleanup_smoke EXIT HUP INT TERM

command -v curl >/dev/null 2>&1 || {
  echo "curl is required" >&2
  exit 1
}

check() {
  name=$1
  url=$2
  expected=$3
  body_file=$4
  shift 4
  code=$(curl --silent --show-error --max-time 20 --output "$body_file" --write-out '%{http_code}' "$@" "$url")
  if [ "$code" != "$expected" ]; then
    echo "$name failed: expected HTTP $expected, got $code" >&2
    cat "$body_file" >&2 || true
    exit 1
  fi
  echo "$name: HTTP $code"
}

check "Frontend" "$APP_URL/" 200 "$TEMP_DIR/frontend.body" --location
check "API health" "$API_URL/health" 200 "$TEMP_DIR/health.body"
check "API readiness" "$API_URL/ready" 200 "$TEMP_DIR/ready.body"
grep -q '"schema_version"' "$TEMP_DIR/ready.body" || {
  echo "API readiness response does not include schema_version" >&2
  exit 1
}

check "Authentication providers" "$API_URL/v1/auth/providers" 200 "$TEMP_DIR/providers.body"
grep -q '"providers"' "$TEMP_DIR/providers.body" || {
  echo "Authentication providers response is invalid" >&2
  exit 1
}

check "Unauthenticated session" "$API_URL/v1/auth/me" 401 "$TEMP_DIR/auth-me.body"
check "Disallowed CORS origin" "$API_URL/v1/auth/providers" 403 "$TEMP_DIR/cors.body" \
  -H 'Origin: https://attacker.invalid'

frontend_headers=$(curl --silent --show-error --max-time 20 --head "$APP_URL/")
api_headers=$(curl --silent --show-error --max-time 20 --head "$API_URL/health")

assert_header() {
  name=$1
  headers=$2
  pattern=$3
  echo "$headers" | grep -qi "$pattern" || {
    echo "$name is missing required header matching: $pattern" >&2
    exit 1
  }
}

assert_header "Frontend" "$frontend_headers" '^strict-transport-security:'
assert_header "Frontend" "$frontend_headers" '^x-content-type-options: nosniff'
assert_header "Frontend" "$frontend_headers" '^x-frame-options: deny'
assert_header "API" "$api_headers" '^strict-transport-security:'
assert_header "API" "$api_headers" '^x-content-type-options: nosniff'
assert_header "API" "$api_headers" '^cache-control: no-store'

if echo "$frontend_headers$api_headers" | grep -qi '^server:'; then
  echo "Server header must not be exposed" >&2
  exit 1
fi

echo "Production smoke test passed"
