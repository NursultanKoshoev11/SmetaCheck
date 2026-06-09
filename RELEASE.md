# SmetaCheck KG Release Notes

## Current release status

This repository is not ready for public production release yet.

It is ready for infrastructure smoke testing:
- Docker Compose starts core services.
- API exposes health endpoint.
- Frontend has a landing page.
- Worker and bot processes start.

## Blockers before public release

1. Real upload endpoint must save the file and create a database record.
2. Worker must process queued estimates.
3. XLSX