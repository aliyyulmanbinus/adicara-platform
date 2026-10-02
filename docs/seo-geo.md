# SEO dan GEO

## Fondasi saat ini

- Bahasa dokumen `id`.
- Title, description, canonical, Open Graph, dan Twitter metadata dirender di server.
- Homepage memuat Organization JSON-LD yang hanya mendeskripsikan informasi terlihat.
- Sitemap dibuat oleh integrasi resmi Astro.
- `robots.txt` dibuat saat build berdasarkan `PUBLIC_SITE_URL`.
- Route `/i/*` dikecualikan dari sitemap.
- Halaman autentikasi placeholder diberi `noindex` dan dikecualikan dari sitemap.
- Undangan memakai `noindex, nofollow` kecuali pemilik secara eksplisit mengaktifkan `allow_indexing`.
- Informasi penting tersedia sebagai HTML tanpa JavaScript browser.

## Prinsip konten

Konten berbahasa Indonesia harus menjawab pertanyaan nyata, tidak memakai statistik palsu, keyword stuffing, testimonial buatan, atau klaim ranking AI. Structured data harus sesuai dengan konten yang terlihat.

## Konfigurasi

Set `PUBLIC_SITE_URL` ke origin HTTPS kanonis saat build produksi. Nilai ini menentukan canonical URL, sitemap, dan referensi sitemap di robots.
