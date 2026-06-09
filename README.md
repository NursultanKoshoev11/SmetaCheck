# SmetaCheck KG

SmetaCheck KG is a production-oriented platform for preliminary construction estimate checks in Kyrgyzstan.

This repository contains:

- Go API service
- Go background worker for estimate analysis and report generation
- Go Telegram bot service
- Next.js web dashboard
- PostgreSQL schema and Docker deployment files

The first production scope is simple and safe: upload a real estimate file, run deterministic checks, generate a human-readable report, and keep a full history for authenticated users.
