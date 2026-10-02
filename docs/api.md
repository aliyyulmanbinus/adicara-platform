# API

Kontrak kanonis berada di [`contracts/openapi.yaml`](../contracts/openapi.yaml).

## Endpoint saat ini

- `GET /healthz` — liveness proses API; tidak memeriksa database.
- `GET /readyz` — readiness API dan PostgreSQL.
- `GET /api/v1/public/invitations/{slug}` — undangan yang berstatus `published`.
- `POST /api/v1/auth/register`, `/login`, `/logout` dan `GET /api/v1/me` — session cookie.
- `/api/v1/invitations` — CRUD undangan milik pengguna dan publish/unpublish.
- `/api/v1/invitations/{id}/guests` — CRUD tamu milik undangan.
- `POST /api/v1/public/invitations/{slug}/rsvp` — RSVP berbasis token tamu.

Respons error konsisten:

```json
{
  "error": {
    "code": "not_found",
    "message": "published invitation was not found"
  }
}
```

API tidak mengaktifkan CORS karena browser mengaksesnya melalui origin yang sama. Endpoint privat memakai session cookie HttpOnly dan header CSRF pada operasi yang mengubah state. Kontrak request, respons, error, dan security scheme lengkap berada di OpenAPI.
