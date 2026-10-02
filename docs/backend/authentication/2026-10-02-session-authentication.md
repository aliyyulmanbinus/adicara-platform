# Session Authentication

Date: 2026-10-02

## Endpoints

- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `GET /api/v1/me`

## Security model

Passwords are hashed with bcrypt and never returned or logged. Registration accepts passwords between 12 and 128 characters. Login failures use one generic `invalid_credentials` response.

Successful registration and login issue a 256-bit random opaque session token in the `adicara_session` cookie. The cookie is HttpOnly, SameSite=Lax, scoped to `/`, and Secure whenever `APP_ENV=production` or `COOKIE_SECURE=true`. PostgreSQL stores only the SHA-256 token hash.

A separate random CSRF token is stored as a hash in the session row and mirrored in the readable `adicara_csrf` cookie. Clients must echo it through `X-CSRF-Token` for logout and every authenticated state-changing request. The backend compares hashes in constant time. The CSRF token grants no authentication without the HttpOnly session cookie.

Nginx limits login and registration attempts more aggressively than general API traffic. Expired sessions are rejected by the repository query. A future maintenance job should periodically delete expired rows.

## Configuration

- `APP_ENV=production` forces Secure cookies.
- `COOKIE_SECURE=true` enables Secure cookies outside production.
- Session lifetime is currently 30 days in backend configuration.
