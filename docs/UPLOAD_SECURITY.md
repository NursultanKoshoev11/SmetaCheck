# SmetaCheck KG upload security requirements

This document defines the minimum upload-security rules required before public paid release.

## Required behavior

All file-ingestion endpoints must use the same validation pipeline:

- single estimate upload;
- batch upload;
- compare upload;
- raw PDF AI analysis.

The validation must run before parsing, AI submission, persistence, or report generation.

## Allowed formats

Default production formats:

- `.xlsx` — allowed;
- `.csv` — allowed;
- `.pdf` — allowed only after PDF signature and active-content checks;
- `.xlsm` — disabled by default.

`XLSM` may be enabled only with `ALLOW_XLSM_UPLOADS=true` after documenting macro handling and customer risk communication.

## Required checks

### Generic checks

- Enforce `MAX_UPLOAD_MB` before reading the full body.
- Sanitize file names and never trust user-provided paths.
- Validate extension and file signature together.
- Reject files with mismatched signature and extension.
- Reject binary content submitted as CSV.
- Store uploads outside the repository and outside the web root.
- Log validation failures with request ID and user ID, without logging file contents.

### XLSX / XLSM checks

Office files are ZIP containers. Validate the archive before calling the Excel parser:

- maximum ZIP entry count: `MAX_XLSX_ZIP_ENTRIES`;
- maximum expanded size: `MAX_XLSX_EXPANDED_MB`;
- maximum compression ratio: `MAX_XLSX_COMPRESSION_RATIO`;
- reject absolute paths, `..` path traversal, empty paths, and unsafe entry names;
- reject XLSM unless `ALLOW_XLSM_UPLOADS=true`.

### PDF checks

Before AI analysis:

- require `%PDF-` file signature;
- reject active content markers such as JavaScript actions, launch actions, embedded files, rich media, and automatic open actions;
- scan the file with the configured malware scanner before sending to any AI provider;
- show the selected AI provider and model in the report.

### Malware scanning

Before public launch, configure one of:

- local ClamAV daemon inside the private backend network;
- managed object-storage malware scanning;
- CI/CD upload quarantine workflow.

A file must not be parsed or submitted to AI until scan status is clean.

## Production environment variables

Recommended defaults:

```env
ALLOW_XLSM_UPLOADS=false
MAX_XLSX_ZIP_ENTRIES=2000
MAX_XLSX_EXPANDED_MB=200
MAX_XLSX_COMPRESSION_RATIO=100
MALWARE_SCAN_REQUIRED=true
MALWARE_SCANNER_URL=tcp://clamav:3310
```

## Release evidence

For every production release, store:

1. unit or integration test output for allowed and rejected files;
2. sample rejected XLSM evidence;
3. sample rejected ZIP-bomb-style XLSX evidence;
4. sample rejected active-content PDF evidence;
5. malware scanner healthcheck output;
6. production smoke-test output after deployment.
