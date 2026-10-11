export type AuthAction = 'login' | 'register';

const byCode: Record<string, string> = {
  invalid_input: 'Lengkapi semua data yang diperlukan.',
  invalid_json: 'Permintaan tidak valid. Muat ulang halaman lalu coba lagi.',
  invalid_email: 'Format email tidak valid.',
  invalid_username:
    'Username harus 3–30 karakter dan hanya berisi huruf, angka, titik, atau garis bawah.',
  weak_password:
    'Kata sandi minimal 8 karakter dan harus mengandung huruf serta angka.',
  password_too_long:
    'Kata sandi terlalu panjang. Gunakan maksimal 72 karakter.',
  email_taken: 'Email sudah terdaftar. Silakan masuk.',
  username_taken: 'Username sudah dipakai. Pilih username lain.',
  account_inactive: 'Akun Anda tidak aktif. Hubungi dukungan Adicara.',
  rate_limited: 'Terlalu banyak percobaan. Tunggu sebentar lalu coba lagi.',
  unsupported_media_type:
    'Permintaan tidak valid. Muat ulang halaman lalu coba lagi.',
};

export const SERVICE_UNAVAILABLE =
  'Layanan sedang tidak dapat dijangkau. Silakan coba lagi sebentar lagi.';

/**
 * Turns a backend error into the Indonesian text shown in the form. Raw backend
 * messages are English and may change, so they are never shown to the user.
 */
export function authErrorMessage(
  status: number,
  code: string | undefined,
  action: AuthAction,
): string {
  if (status >= 500) return SERVICE_UNAVAILABLE;
  if (code === 'unauthorized' && action === 'login') {
    return 'Email/username atau kata sandi salah.';
  }
  const known = status === 429 ? byCode.rate_limited : code && byCode[code];
  if (known) return known;

  return 'Permintaan belum berhasil. Periksa kembali data Anda.';
}
