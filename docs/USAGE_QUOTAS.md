# Production usage quotas

SmetaCheck enforces server-side quotas per user account. The checks are performed atomically in PostgreSQL and cannot be bypassed by hiding or modifying frontend controls.

## Default limits

```env
QUOTA_MONTHLY_UPLOAD_FILES=100
QUOTA_MONTHLY_UPLOAD_MB=2048
QUOTA_MONTHLY_AI_JOBS=200
QUOTA_MONTHLY_BATCHES=50
QUOTA_STORAGE_MB=4096
```

The monthly period starts at 00:00 UTC on the first day of each month.

- `QUOTA_MONTHLY_UPLOAD_FILES` counts files accepted by single and batch uploads.
- `QUOTA_MONTHLY_UPLOAD_MB` counts accepted upload bytes during the month.
- `QUOTA_MONTHLY_AI_JOBS` counts external AI provider calls. Deterministic `rules` analysis does not consume this quota.
- `QUOTA_MONTHLY_BATCHES` counts accepted batch jobs.
- `QUOTA_STORAGE_MB` limits currently retained upload bytes.

Comparison files are processed in temporary storage and do not consume retained-storage quota.

## Reservation behavior

Upload and batch handlers reserve quota before writing persistent files. If file saving, parsing, report generation or PostgreSQL persistence fails, the reservation is rolled back.

Storage quota is released when:

- a directly uploaded estimate is deleted successfully;
- a retained batch source file is removed by the batch cleanup worker.

Monthly upload and job counters are not reduced when the user deletes successful work. They measure monthly consumption, not current storage.

## API response

When a limit is exceeded, the API returns HTTP `429 Too Many Requests`:

```json
{
  "error": "usage quota exceeded",
  "resource": "storage_bytes",
  "limit": 4294967296,
  "used": 4200000000,
  "requested": 200000000
}
```

The response also includes:

```text
X-Quota-Resource
X-Quota-Limit
X-Quota-Used
```

Authenticated clients can read current usage at:

```http
GET /v1/account/usage
```

## External AI behavior

Before an external AI provider is called, SmetaCheck checks the report cache. A cached report does not consume an additional AI job. When the AI quota is exhausted, SmetaCheck returns deterministic rule-based analysis with a warning instead of making a paid external request.

## Operations

Monitor:

- accounts approaching storage limits;
- repeated HTTP 429 responses;
- failed reservation rollbacks;
- mismatch between `account_storage_usage` and actual retained files;
- monthly external AI consumption and provider invoices.

The migration seeds storage counters from existing estimate and batch file paths. Before public launch, compare the seeded totals with filesystem usage and document any intentional differences such as generated reports, which are not currently counted toward upload storage quota.
