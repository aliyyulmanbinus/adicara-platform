# Manajemen Profil User

Modul `profile` memungkinkan pengguna melihat dan mengubah data identitasnya sendiri. Semua rute memakai `Authorization: Bearer <access token>`; usecase-nya memanggil modul `auth` untuk membaca dan mengubah baris di tabel `users`.

## `GET /v1/me`

`200 {"data": {"user": {"id", "email", "username", "name"}}}`. `name` bernilai `null` sampai pernah diisi. Respons ini sengaja tidak memuat `role_id` dan `status` (yang ada pada respons login/register).

Error: `401 unauthorized` (token tidak ada, salah, atau kedaluwarsa), `404 not_found` (akun sudah tidak ada).

## `PATCH /v1/me`

Body `{"name": "..."}`. Nama dipangkas spasinya dan tidak boleh kosong. `200 {"data": {"id", "email", "username", "name"}}`.

> Bentuk respons berbeda dengan GET: `PATCH` mengembalikan pengguna langsung di bawah `data`, sedangkan `GET` membungkusnya dalam `data.user`.

Error: `400 invalid_json`, `400 invalid_input` (nama kosong), `401 unauthorized`, `404 not_found`. Saat ini hanya `name` yang bisa diubah; email dan username tidak.
