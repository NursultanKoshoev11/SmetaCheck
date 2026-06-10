# SmetaCheck KG Production Checklist

Use this checklist before deploying SmetaCheck KG to a public production server.

## 1. Secrets and environment

- [ ] Copy `.env.example` to `.env` on the server.
- [ ] Replace `POSTGRES_PASSWORD` with a strong password.
- [ ] Replace `JWT_SECRET` with at least 64 random characters.
- [ ] Replace `TELEGRAM_WEBHOOK_SECRET` with a strong random secret.
- [ ] Set `TELEGRAM_BOT_TOKEN` only if the bot is used.
- [ ] Set `ALLOWED_ORIGINS` to the real production domains only.
- [ ] Do not commit `.env` or real secrets.

## 2. Database and storage

- [ ] Confirm PostgreSQL starts successfully.
- [ ] Confirm Redis starts successfully.
- [ ] Confirm upload and report directories use persistent Docker volumes.
- [ ] Add backup process for PostgreSQL volume.
- [ ] Add backup process for generated reports if reports must be retained.

## 3. API checks

- [ ] `GET /health` returns HTTP 200.
- [ ] `GET /ready` returns HTTP 200.
- [ ] Production startup fails when placeholder secrets are used.
- [ ] CORS allows only the production frontend domains.
- [ ] Upload endpoints enforce `MAX_UPLOAD_MB` once upload API is implemented.

## 4. CI/CD

- [ ] GitHub Actions CI passes on `main`.
- [ ] Go formatting check passes.
- [ ] `go vet ./...` passes.
- [ ] `go test ./...` passes.
- [ ] API, worker, and bot Docker images build successfully.

## 5. Reverse proxy and TLS

- [ ] Put API behind Nginx, Caddy, Traefik, or a cloud load balancer.
- [ ] Enable HTTPS with a valid certificate.
- [ ] Redirect HTTP to HTTPS.
- [ ] Set upload body size in reverse proxy to match `MAX_UPLOAD_MB`.
- [ ] Do not expose PostgreSQL or Redis publicly.

## 6. Monitoring

- [ ] Capture container logs.
- [ ] Add uptime monitoring for `/ready`.
- [ ] Add disk usage alerts for Docker volumes.
- [ ] Add database backup success/failure alerts.

## 7. Product readiness

- [ ] Implement estimate upload endpoint.
- [ ] Implement estimate parsing.
- [ ] Implement deterministic estimate checks.
- [ ] Implement report generation.
- [ ] Implement user estimate history.
- [ ] Implement frontend dashboard or remove dashboard from release scope.
