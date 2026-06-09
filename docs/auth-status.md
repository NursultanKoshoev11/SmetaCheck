# Auth Status

Implemented:
- auth user migration
- email/password DTO
- bcrypt password hashing
- email registration service contract
- Google login DTO and provider interface
- Telegram login DTO and verifier shell

Blocked before production:
- JWT issuing is not wired
- HTTP auth endpoints are not wired
- Google ID token verification is not implemented
- Telegram verifier must be replaced with full HMAC validation
- auth storage SQL methods must be added locally

Release gate: do not enable public login until all blocked items are closed.
