# SMTP setup for registration emails

SmetaCheck sends email verification and password reset messages through SMTP.

## Required production variables

```env
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_TLS_MODE=starttls
SMTP_TIMEOUT=15s
SMTP_USERNAME=your-smtp-username
SMTP_PASSWORD=your-smtp-password
SMTP_FROM=no-reply@your-domain.example
SMTP_FROM_NAME=SmetaCheck KG
```

Use `SMTP_TLS_MODE=starttls` for port `587` and `SMTP_TLS_MODE=implicit` for port `465`.

The production API now refuses to start when SMTP username or password is missing. This prevents registration from appearing available while verification mail is unable to authenticate.

## Verification after deployment

1. Restart the API after updating `.env`.
2. Register with a new email address.
3. Confirm the API logs contain no `verification email delivery failed` entry.
4. Check Inbox and Spam for the message.
5. Open the verification link and confirm login succeeds.

Do not commit real SMTP credentials to GitHub.
