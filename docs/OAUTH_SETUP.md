# Google and Telegram authentication setup

SmetaCheck uses server-side OpenID Connect. The same button handles both registration and login: the first successful authorization creates the local account, and later authorizations open the existing account.

## Public URLs

The production example assumes:

- Frontend: `https://app.smetacheck.kg`
- API: `https://api.smetacheck.kg`
- Google callback: `https://api.smetacheck.kg/v1/auth/google/callback`
- Telegram callback: `https://api.smetacheck.kg/v1/auth/telegram/callback`

Callback URLs must match exactly. Production callbacks must use HTTPS.

## Google

1. Create or select a project in Google Cloud Console.
2. Configure the OAuth consent screen.
3. Create an OAuth 2.0 Client ID with application type **Web application**.
4. Add the exact authorized redirect URI:

```text
https://api.smetacheck.kg/v1/auth/google/callback
```

5. Put the generated client ID and client secret in the production `.env`:

```env
AUTH_GOOGLE_ENABLED=true
GOOGLE_OIDC_CLIENT_ID=your-client-id
GOOGLE_OIDC_CLIENT_SECRET=your-client-secret
GOOGLE_OIDC_REDIRECT_URL=https://api.smetacheck.kg/v1/auth/google/callback
```

SmetaCheck requests only `openid email profile`. Google accounts are accepted only when Google returns a verified email.

## Telegram

1. Open `@BotFather` in Telegram.
2. Select the bot used for SmetaCheck.
3. Open **Bot Settings → Web Login**.
4. Add the allowed origin:

```text
https://app.smetacheck.kg
```

5. Add the exact redirect URI:

```text
https://api.smetacheck.kg/v1/auth/telegram/callback
```

6. Copy the generated OIDC Client ID and Client Secret into the production `.env`:

```env
AUTH_TELEGRAM_ENABLED=true
TELEGRAM_OIDC_ISSUER=https://oauth.telegram.org
TELEGRAM_OIDC_CLIENT_ID=your-client-id
TELEGRAM_OIDC_CLIENT_SECRET=your-client-secret
TELEGRAM_OIDC_REDIRECT_URL=https://api.smetacheck.kg/v1/auth/telegram/callback
```

The Telegram issuer is fixed to `https://oauth.telegram.org`. SmetaCheck uses PKCE with SHA-256 and sends the client credentials through HTTP Basic authentication when exchanging the authorization code.

## Shared security settings

```env
OAUTH_HTTP_TIMEOUT=15s
OAUTH_STATE_TTL=10m
COOKIE_SECURE=true
PUBLIC_BASE_URL=https://app.smetacheck.kg
API_PUBLIC_BASE_URL=https://api.smetacheck.kg
ALLOWED_ORIGINS=https://smetacheck.kg,https://www.smetacheck.kg,https://app.smetacheck.kg
```

The implementation validates a database-backed one-time `state`, a browser-bound state cookie, PKCE, the OIDC token signature, issuer, audience, expiration and nonce. Login sessions are stored in Secure HttpOnly cookies.

## Deploy

After updating `.env`:

```bash
git pull --ff-only origin main
docker compose -f docker-compose.production.yml up -d --build api frontend
docker compose -f docker-compose.production.yml logs --tail=200 api
```

## Verification checklist

1. Open the login page in a private browser window.
2. Complete Google login with a new account and confirm a SmetaCheck account is created.
3. Log out and repeat Google login; confirm the same account is reused.
4. Repeat both steps with Telegram.
5. Cancel each provider flow and confirm the login page displays a safe error message.
6. Confirm `/v1/auth/me` returns the authenticated user after each successful flow.
7. Confirm another browser cannot reuse an old callback URL.
8. Confirm the API logs do not contain tokens, authorization codes or client secrets.

Never commit real client secrets to GitHub.
