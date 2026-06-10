# SmetaCheck KG — Production deployment guide

This guide prepares a real server deployment. Do not deploy with placeholder secrets.

## 1. Server requirements

Recommended minimum for MVP launch:

- Ubuntu 22.04 or 24.04
- 2 vCPU
- 4 GB RAM
- 40 GB SSD
- Docker + Docker Compose plugin
- Nginx or Caddy reverse proxy
- Domain with DNS access

## 2. DNS records

Point these records to the server IP:

```text
smetacheck.kg      A  SERVER_IP
www.smetacheck.kg  A  SERVER_IP
app.smetacheck.kg  A  SERVER_IP
api.smetacheck.kg  A  SERVER_IP
```

## 3. Prepare server

```bash
sudo apt update
sudo apt install -y git curl ca-certificates nginx certbot python3-certbot-nginx postgresql-client
curl -fsSL https://get.docker.com | sudo sh
sudo usermod -aG docker $USER
newgrp docker
```

## 4. Clone project

```bash
sudo mkdir -p /opt/smetacheck
sudo chown -R $USER:$USER /opt/smetacheck
git clone https://github.com/NursultanKoshoev11/SmetaCheck.git /opt/smetacheck
cd /opt/smetacheck
```

## 5. Create production env

```bash
cp .env.production.example .env
nano .env
```

Replace every placeholder:

- `POSTGRES_PASSWORD`
- `DATABASE_URL`
- `JWT_SECRET`
- `TELEGRAM_WEBHOOK_SECRET`
- domain values
- `NEXT_PUBLIC_API_BASE`

Generate secrets:

```bash
openssl rand -base64 48
openssl rand -hex 48
```

## 6. Create storage directories

```bash
sudo mkdir -p /var/lib/smetacheck/uploads /var/lib/smetacheck/reports /var/backups/smetacheck
sudo chown -R $USER:$USER /var/lib/smetacheck /var/backups/smetacheck
```

## 7. Build and start

```bash
docker compose --env-file .env up -d --build
```

Check containers:

```bash
docker compose ps
docker compose logs -f api
```

## 8. Apply database schema

```bash
set -a
source .env
set +a
psql "$DATABASE_URL" -f db/migrations/001_initial_schema.sql
```

## 9. Configure Nginx

```bash
sudo cp deploy/nginx/smetacheck.conf /etc/nginx/sites-available/smetacheck.conf
sudo ln -sf /etc/nginx/sites-available/smetacheck.conf /etc/nginx/sites-enabled/smetacheck.conf
sudo nginx -t
sudo systemctl reload nginx
```

## 10. Enable HTTPS

Use Certbot after DNS is ready:

```bash
sudo certbot --nginx -d smetacheck.kg -d www.smetacheck.kg -d app.smetacheck.kg
sudo certbot --nginx -d api.smetacheck.kg
```

Then test renewal:

```bash
sudo certbot renew --dry-run
```

## 11. Health checks

```bash
curl https://api.smetacheck.kg/health
curl https://api.smetacheck.kg/ready
```

Open:

```text
https://smetacheck.kg
https://app.smetacheck.kg/upload
https://app.smetacheck.kg/reports
https://app.smetacheck.kg/compare
```

## 12. Backups

Make script executable:

```bash
chmod +x scripts/backup.sh
```

Run manually:

```bash
APP_DIR=/opt/smetacheck BACKUP_DIR=/var/backups/smetacheck scripts/backup.sh
```

Add cron:

```bash
crontab -e
```

Add:

```cron
15 2 * * * APP_DIR=/opt/smetacheck BACKUP_DIR=/var/backups/smetacheck /opt/smetacheck/scripts/backup.sh >> /var/log/smetacheck-backup.log 2>&1
```

## 13. Production smoke test

1. Open `/upload`.
2. Upload a small XLSX or CSV estimate.
3. Open `/dashboard`.
4. Open `/reports`.
5. Download report.
6. Open `/compare`.
7. Compare two versions.
8. Check API logs.
9. Check backups.

## 14. Do not launch publicly until these are done

- Use real secrets, no placeholders.
- HTTPS works.
- Backups work.
- Parser tested with real estimate files.
- User-specific storage is moved to PostgreSQL.
- Auth protects uploaded estimate history.
- Upload storage is backed up.
- Monitoring is enabled.
