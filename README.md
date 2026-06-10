# SmetaCheck KG

SmetaCheck KG is a production-oriented platform for preliminary construction estimate checks in Kyrgyzstan.

The first production scope is intentionally simple and safe: upload a real estimate file, run deterministic checks, generate a human-readable report, and keep a full history for authenticated users.

## Current services

- Go API service
- Go background worker service
- Go Telegram bot service
- PostgreSQL database
- Redis queue/cache
- Docker Compose deployment files

The Next.js web dashboard is part of the product roadmap. Keep it out of the production release scope until the frontend code and Docker deployment are added.

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

## Quick start

```bash
cp .env.example .env
# edit .env and replace all production secrets
docker compose up --build
```

Check the API:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

## Required before real public launch

- Implement estimate upload endpoint.
- Implement estimate parser.
- Implement deterministic estimate checks.
- Implement report generation.
- Implement authenticated estimate history.
- Add frontend dashboard or keep the first release API-only.
- Configure HTTPS through a reverse proxy or cloud load balancer.
- Configure database backups and uptime monitoring.

See [`docs/PRODUCTION_CHECKLIST.md`](docs/PRODUCTION_CHECKLIST.md) before deploying.
