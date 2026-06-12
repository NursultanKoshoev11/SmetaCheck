# Production backup and restore

SmetaCheck production backups include:

- PostgreSQL custom-format dump;
- uploaded estimate files;
- generated reports;
- SHA-256 checksums for every backup artifact.

Backups are created by `scripts/backup-production.sh`. Restores are performed by `scripts/restore-production.sh`.

## Required production setup

Create a directory outside the repository:

```bash
sudo install -d -m 700 /var/backups/smetacheck
```

Create a backup encryption password file. Keep a protected copy outside the production server. Losing this password makes encrypted backups unrecoverable.

```bash
sudo install -d -m 700 /etc/smetacheck
umask 077
openssl rand -base64 48 | sudo tee /etc/smetacheck/backup-password >/dev/null
sudo chmod 600 /etc/smetacheck/backup-password
```

Configure `.env`:

```env
BACKUP_DIR=/var/backups/smetacheck
BACKUP_RETENTION_DAYS=14
BACKUP_REQUIRE_ENCRYPTION=true
BACKUP_ENCRYPTION_PASSWORD_FILE=/etc/smetacheck/backup-password
BACKUP_S3_URI=s3://your-private-backup-bucket/smetacheck
```

`BACKUP_S3_URI` is optional, but a production backup must be copied away from the application server. Configure the AWS CLI with credentials that can write only to the backup location. Enable object versioning and server-side encryption on the bucket.

The scripts read only their supported backup settings from `.env`; they do not execute or source the file.

## Create a backup

Run from the repository directory:

```bash
./scripts/backup-production.sh
```

A completed backup is written to a UTC timestamp directory such as:

```text
/var/backups/smetacheck/20260612T120000Z/
```

The script first writes into a hidden partial directory and publishes the final directory only after the dump, volume archives, encryption and checksums succeed.

## Scheduling

Run at least daily. Example root cron entry:

```cron
15 2 * * * cd /opt/smetacheck && ./scripts/backup-production.sh >> /var/log/smetacheck-backup.log 2>&1
```

Add external monitoring for the exit status and the age of the newest successful backup. A backup older than the agreed RPO must trigger an alert.

## Restore

A restore overwrites the production database, upload volume and report volume. Confirm that the selected backup is correct and take a fresh backup before restoring whenever the current system is still accessible.

```bash
BACKUP_SOURCE=/var/backups/smetacheck/20260612T120000Z \
RESTORE_CONFIRM=RESTORE_SMETACHECK \
./scripts/restore-production.sh
```

The restore script:

1. verifies SHA-256 checksums;
2. decrypts artifacts into a temporary protected directory;
3. stops API, worker and frontend services;
4. restores PostgreSQL with `pg_restore --clean --if-exists`;
5. replaces the upload and report volumes;
6. restarts the application services.

Immediately run:

```bash
./scripts/smoke-production.sh
```

Then manually verify login, estimate upload, batch analysis, report download and a known historical estimate.

## Restore drill

Perform a restore drill at least monthly on an isolated server or isolated Docker project. Do not wait for an incident to test the backup.

Record:

- backup timestamp;
- restore start and finish time;
- database row counts;
- upload/report object counts;
- smoke-test result;
- any errors and remediation.

## RPO and RTO

Before public launch, choose and document:

- **RPO:** maximum acceptable data loss, for example 24 hours for daily backups;
- **RTO:** maximum acceptable recovery time, measured by a real restore drill.

The scripts provide the mechanism, but they do not prove that backups are scheduled, uploaded offsite, monitored or restorable. Those controls must be configured on the production infrastructure.
