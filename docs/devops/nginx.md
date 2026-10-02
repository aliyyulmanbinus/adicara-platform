# Nginx

Nginx adalah pintu masuk same-origin:

- `/api/*`, `/healthz`, `/readyz` → backend.
- `/_astro/*` → frontend dengan cache immutable.
- route lain → frontend.

Konfigurasi menambahkan compression, body-size limit, rate limit API, dan security headers dasar. CSP saat ini mengizinkan inline style/script karena Astro dapat menghasilkan markup tersebut; kebijakan perlu diperketat setelah inventaris asset produksi tersedia.

TLS diasumsikan diterminasi oleh proxy host Sumopod. Bila Nginx container menjadi terminator TLS langsung, tambahkan konfigurasi sertifikat dan redirect HTTP secara terpisah.

