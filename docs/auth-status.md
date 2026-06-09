# Auth Status

Implemented:
- user auth migration
- email/password DTO
- bcrypt password hashing
- email registration service contract
- Google login DTO and provider interface
- Telegram login DTO and temporary verifier shell

Not production complete yet:
- JWT issuing is not wired
- HTTP auth endpoints are not wired
- Google id token verification is not implemented
- Telegram payload verification needs full HMAC check
- storage SQL methods for auth were blocked by connector and must be added locally

Release gate