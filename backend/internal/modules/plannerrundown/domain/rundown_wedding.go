package domain

// The wedding agendas of Wedding_Planner.pdf p. 14 (06 BE 5.2 and 5.3): 11 for the lamaran, 8 for the akad and 14 for the
// resepsi, 33 in all, with the text of the PDF (the capitals of agendas 17-22 and "Iring-iring" are the only edits).
// TestWeddingTemplateMatchesSpec reads the tables of the spec and fails when this file drifts. Open agendas have no
// End: agenda 11 of the lamaran ("10.40 - selesai") and agenda 9 of the resepsi ("09.15").

// WeddingLamaran is the rundown of the lamaran session.
var WeddingLamaran = []TemplateAgenda{
	{Key: "wedding.lamaran_01", ScopeKey: "lamaran", Start: "09:00", End: "09:10", Name: "Pembukaan", PIC: "MC", Description: "Penyambutan keluarga CMP dan mempersilahkan duduk", Highlights: "- Keluarga besar pihak pria hadir\n- Persiapan menyambut keluarga besar pihak pria\n- Pembukaan oleh MC\n- Mempersilahkan duduk"},
	{Key: "wedding.lamaran_02", ScopeKey: "lamaran", Start: "09:10", End: "09:25", Name: "Sambutan Penyerahan CMP", PIC: "Perwakilan Keluarga CMP", Description: "Sambutan Penyerahan Keluarga CMP", Highlights: "- Kata Sambutan\n- Menyampaikan maksud & tujuan kedatangan\n- Meyakinkan keluarga CMW"},
	{Key: "wedding.lamaran_03", ScopeKey: "lamaran", Start: "09:25", End: "09:40", Name: "Sambut Penerimaan CMW", PIC: "Perwakilan Keluarga CMW", Description: "Sambutan Penerimaan Keluarga CMW", Highlights: "- Kata Sambutan\n- Mengucapkan terima kasih atas kedatangan\n- Memberikan tanggapan\n- Menyerahkan jawaban ke CMW"},
	{Key: "wedding.lamaran_04", ScopeKey: "lamaran", Start: "09:40", End: "09:50", Name: "Pemanggilan dan perkenalan CMP", PIC: "CMP", Description: "Perkenalan CMP", Highlights: "- MC mempersilahkan CMP berdiri untuk mengutarakan secara langsung\n- Meminta izin kpd orangtua\n- Meminta izin kpd orang tua CMW\n- Mengutarakan kpd CMW"},
	{Key: "wedding.lamaran_05", ScopeKey: "lamaran", Start: "09:50", End: "10:00", Name: "Pemanggilan dan perkenalan CMW", PIC: "CMW", Description: "Perkenalan CMW", Highlights: "- MC mempersilahkan CMW masuk untuk menjawab secara langsung (tepok pundak)\n- Mengucapkan terima kasih\n- Meminta izin kpd orangtua\n- Mengutarakan kpd CMW"},
	{Key: "wedding.lamaran_06", ScopeKey: "lamaran", Start: "10:00", End: "10:10", Name: "Perkenalan 2 Keluarga", PIC: "Perwakilan Keluarga CMP & CMW", Description: "Perkenalan 2 keluarga yang akan diwakilkan oleh salah satu keluarga calon mempelai"},
	{Key: "wedding.lamaran_07", ScopeKey: "lamaran", Start: "10:10", End: "10:20", Name: "Menceritakan kisah cinta calon mempelai", PIC: "CMP & CMW", Description: "Pertanyaan dari MC"},
	{Key: "wedding.lamaran_08", ScopeKey: "lamaran", Start: "10:20", End: "10:25", Name: "Pemanggilan kedua ibu mempelai & penyematan cincin", PIC: "Kedua Ibu CMP & CMW", Description: "Ibu calon mempelai untuk penyerahan cincin dilanjutkan dengan penyematan cincin"},
	{Key: "wedding.lamaran_09", ScopeKey: "lamaran", Start: "10:25", End: "10:35", Name: "Games with calon mempelai", Description: "Sesi Games (Shoe Game & Games untuk Tamu)"},
	{Key: "wedding.lamaran_10", ScopeKey: "lamaran", Start: "10:35", End: "10:40", Name: "DOA"},
	{Key: "wedding.lamaran_11", ScopeKey: "lamaran", Start: "10:40", Name: "Ramah Tamah - penutup", PIC: "MC", Description: "Foto bersama & Makan siang"},
}

// WeddingAkadResepsi is the rundown of the akad (agendas 1-8) and the resepsi (9-22) sessions; the keys run 01-22 as in the PDF.
var WeddingAkadResepsi = []TemplateAgenda{
	{Key: "wedding.rundown_01", ScopeKey: "akad", Start: "07:00", End: "07:20", Name: "Persiapan", PIC: "CMP & CMW", Description: "Penghulu datang dan melakukan pemeriksaan kelengkapan persyaratan administrasi, kelengkapan dan kesiapan calon pengantin, serta kehadiran wali dan saksi."},
	{Key: "wedding.rundown_02", ScopeKey: "akad", Start: "07:21", End: "07:25", Name: "Pembukaan acara", PIC: "MC", Description: "Pembawa acara melakukan pembukaan acara akad nikah."},
	{Key: "wedding.rundown_03", ScopeKey: "akad", Start: "07:26", End: "07:35", Name: "Pembacaan ayat suci", PIC: "Qori", Description: "Pembacaan ayat-ayat kitab suci."},
	{Key: "wedding.rundown_04", ScopeKey: "akad", Start: "07:36", End: "07:45", Name: "Khutbah", PIC: "Ustadz/Kyai", Description: "Kutbah pernikahan dan pengumuman hasil kelengkapan persyaratan."},
	{Key: "wedding.rundown_05", ScopeKey: "akad", Start: "07:46", End: "07:50", Name: "Ijab Kabul", PIC: "Penghulu", Description: "Ijab kabul yang dipimpin oleh penghulu dan dilakukan dan dilakukan oleh wali nikah."},
	{Key: "wedding.rundown_06", ScopeKey: "akad", Start: "07:51", End: "08:00", Name: "Doa bersama", PIC: "Kyai/penghulu", Description: "Doa bersama dan penyerahan buku nikah."},
	{Key: "wedding.rundown_07", ScopeKey: "akad", Start: "08:01", End: "08:10", Name: "Sesi foto", PIC: "Fotografer", Description: "Sesi foto bersama dan penyerahan maskawin."},
	{Key: "wedding.rundown_08", ScopeKey: "akad", Start: "08:11", End: "09:00", Name: "Selesai akad", PIC: "MC", Description: "Persiapan melanjutkan ke acara resepsi."},
	{Key: "wedding.rundown_09", ScopeKey: "resepsi", Start: "09:15", Name: "Resepsi", Description: "Pasangan pengantin sudah siap di lokasi acara."},
	{Key: "wedding.rundown_10", ScopeKey: "resepsi", Start: "09:15", End: "09:30", Name: "Persiapan resepsi", PIC: "CMP & CMW", Description: "Menunggu tamu undangan berdatangan."},
	{Key: "wedding.rundown_11", ScopeKey: "resepsi", Start: "09:31", End: "09:35", Name: "Pembukaan acara", PIC: "MC", Description: "Acara dibuka oleh pembawa acara."},
	{Key: "wedding.rundown_12", ScopeKey: "resepsi", Start: "09:36", End: "09:40", Name: "Iring-iring", PIC: "CMP & CMW", Description: "Pengantin dan iring-iringannya berjalan ke pelaminan."},
	{Key: "wedding.rundown_13", ScopeKey: "resepsi", Start: "09:41", End: "10:15", Name: "Sambutan", PIC: "Pihak keluarga", Description: "Pidato penyambutan dari keluarga."},
	{Key: "wedding.rundown_14", ScopeKey: "resepsi", Start: "09:50", End: "10:20", Name: "Acara adat", PIC: "Yang bertugas", Description: "Acara adat jika ada."},
	{Key: "wedding.rundown_15", ScopeKey: "resepsi", Start: "10:20", End: "10:21", Name: "Doa bersama", PIC: "Kyai/Ustadz", Description: "Doa bersama."},
	{Key: "wedding.rundown_16", ScopeKey: "resepsi", Start: "10:21", End: "10:25", Name: "Potong kue", PIC: "CMP & CMW", Description: "Potong kue pernikahan yang dilakukan kedua mempelai."},
	{Key: "wedding.rundown_17", ScopeKey: "resepsi", Start: "10:26", End: "12:30", Name: "Sesi makan", PIC: "MC", Description: "Tamu undangan dipersilahkan mengambil makanan dan minum."},
	{Key: "wedding.rundown_18", ScopeKey: "resepsi", Start: "10:26", End: "12:30", Name: "Entertain", PIC: "Pengisi acara", Description: "Acara hiburan berlangsung."},
	{Key: "wedding.rundown_19", ScopeKey: "resepsi", Start: "10:30", End: "12:30", Name: "Sesi salaman", PIC: "CMP & CMW", Description: "Tamu undangan bisa mengantre ke pelaminan untuk memberikan selamat kepada calon pengantin."},
	{Key: "wedding.rundown_20", ScopeKey: "resepsi", Start: "12:31", End: "12:33", Name: "Penutupan acara", PIC: "MC", Description: "Penutupan acara oleh pembawa acara."},
	{Key: "wedding.rundown_21", ScopeKey: "resepsi", Start: "12:35", End: "12:40", Name: "Lempar bunga", PIC: "CMP & CMW", Description: "Pengantin keluar dari lokasi undangan dan melempar buket bunga."},
	{Key: "wedding.rundown_22", ScopeKey: "resepsi", Start: "12:41", End: "13:00", Name: "Sesi foto", PIC: "Fotografer", Description: "Melakukan sesi foto terakhir dan pulang dari lokasi acara."},
}
