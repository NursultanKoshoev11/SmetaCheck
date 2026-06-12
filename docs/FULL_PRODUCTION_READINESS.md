# SmetaCheck KG — Full Production Release Gates

A public paid release is allowed only after every P0 item below has evidence attached to the release ticket.

## P0 release blockers

- [ ] Privacy Policy and Terms are published and linked from every public page.
- [ ] The current legal-document version is stored when the user accepts registration and upload processing.
- [ ] Account deletion removes active sessions, estimates, batch files, comparisons, AI reports, and live storage objects.
- [ ] Upload, batch, and compare endpoints use the same document validation pipeline.
- [ ] XLSX/XLSM archives have entry-count, expanded-size, path, and compression-ratio limits.
- [ ] XLSM is disabled by default and macro handling is documented.
- [ ] PDF active content is rejected and every uploaded file is scanned before parsing.
- [ ] Billing webhooks are authenticated, idempotent, and linked to account quotas.
- [ ] Reports are generated on the server in PDF and XLSX formats.
- [ ] Uptime, application errors, queue depth, disk, certificate expiry, AI spend, and backup age have alerts.
- [ ] Encrypted backups run on a schedule and are copied outside the application server.
- [ ] A restore drill has measured RPO and RTO evidence.
- [ ] E2E covers registration, login, upload, batch, compare, report, quotas, billing, deletion, and logout.
- [ ] Deployment performs backup, migration, health checks, smoke tests, and automatic rollback.
- [ ] DNS, HTTPS, SMTP, OAuth, firewall, production secrets, and public smoke tests are verified.

## Mandatory evidence

Store the following for every production release:

1. Commit SHA and container image digests.
2. Successful CI and security workflow links.
3. Database migration output.
4. Production smoke-test output.
5. Latest backup timestamp and offsite object reference.
6. Latest restore-drill date, duration, row counts, and file counts.
7. Monitoring dashboard screenshot and test-alert result.
8. Payment test transaction and verified webhook event.
9. Sample PDF and XLSX reports.
10. Approval from the product owner and production operator.

## External configuration that cannot be completed from source code alone

The repository can provide code and scripts, but the following require access to external accounts or the production server:

- DNS records and domain ownership;
- TLS certificate issuance on the real host;
- SMTP credentials and delivery-domain verification;
- Google and Telegram OAuth applications;
- payment-provider merchant account and webhook secret;
- firewall and cloud-network rules;
- offsite backup bucket and credentials;
- monitoring notification channels;
- real restore drill on isolated infrastructure.

Do not mark an external item complete without test output or a provider-side confirmation record.
