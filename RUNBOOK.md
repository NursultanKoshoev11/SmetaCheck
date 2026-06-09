# SmetaCheck KG runbook

1. Copy `.env.example` to `.env`.
2. Set strong production secrets.
3. Start core: docker compose up -d --build
4. Start all: docker compose -f docker-compose.yml -f docker-compose.worker.yml -f docker-compose.frontend.yml up -d --build
5. API health URL: http://localhost:8080/health
