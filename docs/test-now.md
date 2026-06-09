# Test now

Status: API smoke test.

Run:

1. Copy env example to .env.
2. Start docker compose.
3. Open http://localhost:8080/health.

Auth contract endpoints:

POST /v1/auth/register
POST /v1/auth/login

Expected result: HTTP 202 with status ready.

Not ready yet:

- real user creation
- JWT session
- Google token verification
- Telegram signature verification
- upload pipeline
