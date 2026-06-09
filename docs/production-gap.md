# Production gap

Added now:
- API DB pool helper.
- User lookup helper.
- User create helper.
- Session token helper.
- HTML report renderer.
- Admin user/file/error row models.

Still not fully production:
- Email handlers need DB-backed JSON flow.
- JWT access and refresh token persistence.
- Google ID token verification.
- Telegram signature verification.
- Upload handler to DB queue.
- Worker queue processing.
- PDF renderer integration.
- Real email and Telegram delivery.
- Admin SQL queries.

Reason: some auth, SQL and file-write payloads were blocked or truncated by the GitHub connector. Do not mark public launch ready until these are implemented and tested locally.
