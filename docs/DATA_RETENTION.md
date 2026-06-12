# Data retention and deletion

SmetaCheck stores uploaded estimate files, generated reports, parsed estimate rows, findings, comparison results and batch-analysis metadata.

## Comparison uploads

Files submitted to `POST /v1/estimates/compare` are temporary processing inputs. They are written to the operating system temporary directory with owner-only permissions and are deleted after comparison, including error paths. Only the resulting comparison payload and original display names are stored in PostgreSQL.

## Single estimate uploads

A file submitted to `POST /v1/estimates/upload` is retained in the upload storage together with its generated report until the owner deletes the estimate or a separate retention policy removes it.

The owner can delete an estimate through:

```http
DELETE /v1/estimates/{estimate_id}
```

A successful deletion returns HTTP `204 No Content` and removes:

- the owner-scoped estimate row;
- estimate items and findings through database cascades;
- cached AI reports through database cascades;
- the generated report file;
- the uploaded file when it is owned directly by the estimate.

The API records an `estimate.deleted` audit event. A user cannot delete an estimate owned by another user; the response is indistinguishable from a missing estimate.

## Batch-derived estimates

Batch uploads own their source files through `analysis_batch_files`. Deleting a derived estimate removes the estimate and generated report, but preserves the shared batch source file. The batch retention process removes that file after `BATCH_FILE_RETENTION_DAYS`.

This avoids deleting a file that is still part of a retained batch record.

## Batch retention

Completed and failed batch source files are eligible for deletion after:

```env
BATCH_FILE_RETENTION_DAYS=30
```

The worker runs cleanup at:

```env
BATCH_CLEANUP_INTERVAL=6h
```

Set retention according to customer agreements and applicable legal requirements. Setting retention to `0` disables automatic batch-file deletion and should not be used accidentally in production.

## Operational controls

Before public launch:

1. publish the retention period in the privacy notice and customer terms;
2. define retention for compare results, audit logs and account data;
3. monitor upload-volume usage and cleanup failures;
4. test deletion with direct uploads and batch-derived estimates;
5. include deletion and retention behavior in backup/restore procedures;
6. document whether deleted data remains in encrypted backups until backup expiration.

Deleting live data does not immediately remove it from historical backups. Backup retention and restore access must be documented separately.
