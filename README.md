# SmetaCheck KG

SmetaCheck KG is a production-oriented platform for preliminary construction estimate checks in Kyrgyzstan.

The current MVP scope is intentionally simple and testable: upload a real estimate file, store it safely, generate review metadata, create a downloadable report file, and show the result in the web dashboard.

## Current services

- Go API service
- Next.js web dashboard
- Go background worker service
- Go Telegram bot service
- PostgreSQL database for future durable schema-backed data
- Redis queue/cache for future async processing
- Docker Compose deployment files

## Working MVP flow

- `POST /v1/estimates/upload` uploads a file and creates estimate metadata.
- `GET /v1/estimates` returns uploaded estimate history.
- `GET /v1/estimates/{id}` returns one estimate.
- `GET /v1/estimates/{id}/report` downloads the generated report text file.
- `/upload` in the web app uploads files to the API.
- `/dashboard` and `/reports` read real data from the API.

## Production preparation already included

- API health endpoint: `GET /health`
- API readiness endpoint: `GET /ready`
- HTTP server timeouts
- Request body size limit based on `MAX_UPLOAD_MB`
- CORS whitelist based on `ALLOWED_ORIGINS`
- Security headers
- Request ID header
- Panic recovery middleware
- Production placeholder secret validation
- Separate Dockerfiles for API, worker, and Telegram bot
- Docker Compose healthchecks and persistent volumes
- GitHub Actions CI for Go checks and Docker builds

## Local development quick start

```bash
cp .env.local.example .env
mkdir -p data/uploads data/reports
go run ./cmd/api
```

In another terminal:

```bash
cd frontend
cp .env.example .env.local
npm install
npm run dev
```

Open:

```text
http://localhost:3000
```

For a LAN device, set `frontend/.env.local` like this:

```text
NEXT_PUBLIC_API_BASE=http://192.168.1.2:8080
```

Then restart the frontend dev server.

## Docker quick start

```bash
cp .env.example .env
# edit .env and replace all production secrets before APP_ENV=production deployment
docker compose up --build
```

Check the API:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
curl http://localhost:8080/v1/estimates
```

## Required before real public launch

- Replace text-file metadata storage with PostgreSQL-backed tables.
- Implement real Excel/PDF row-level parsing.
- Implement deterministic estimate checks for quantities, units, duplicate rows, totals, and suspicious prices.
- Replace generated TXT report with PDF/Excel export.
- Add real authentication to protect user-specific estimate history.
- Configure HTTPS through a reverse proxy or cloud load balancer.
- Configure database backups and uptime monitoring.

See [`docs/PRODUCTION_CHECKLIST.md`](docs/PRODUCTION_CHECKLIST.md) before deploying.
