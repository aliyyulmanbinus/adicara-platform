// Code generated from PLANNER_SPEC (03 BE sections 6.1 and 6.2) by a one-off script; edit freely afterwards,
// TestTemplatesMatchSpec keeps the two in step.

package domain

// weddingPresetKeys and weddingPresetTotals are the six packages of Wedding_Planner.pdf p. 4; the column i of every
// weddingPresetLine.Amounts belongs to package i.
var weddingPresetKeys = [6]string{"wedding_10jt", "wedding_20jt", "wedding_30jt", "wedding_50jt", "wedding_75jt", "wedding_100jt"}

var weddingPresetTotals = [6]int64{10_000_000, 20_000_000, 30_000_000, 50_000_000, 75_000_000, 100_000_000}

// weddingPresetLines are the ten items of every package (03 BE 6.1), in the order of the PDF.
var weddingPresetLines = []presetLine{
	{Scope: "akad", Key: "biaya_kua", Name: "Biaya KUA", Category: "Legalitas", Amounts: [6]int64{600_000, 600_000, 600_000, 600_000, 600_000, 600_000}},
	{Scope: "resepsi", Key: "mahar", Name: "Mahar & cincin pernikahan", Category: "Mahar", Amounts: [6]int64{1_400_000, 2_500_000, 4_500_000, 5_000_000, 5_000_000, 5_000_000}},
	{Scope: "resepsi", Key: "catering", Name: "Catering", Category: "Catering", Amounts: [6]int64{4_500_000, 5_000_000, 10_000_000, 20_000_000, 30_000_000, 50_000_000}},
	{Scope: "resepsi", Key: "rias_busana", Name: "Rias & baju pengantin", Category: "Busana & MUA", Amounts: [6]int64{250_000, 1_500_000, 2_400_000, 3_000_000, 5_000_000, 5_000_000}},
	{Scope: "resepsi", Key: "venue_dekor", Name: "Sewa tempat/gedung & dekorasi", Category: "Venue & Dekorasi", Amounts: [6]int64{2_000_000, 4_500_000, 4_500_000, 10_000_000, 20_000_000, 24_000_000}},
	{Scope: "resepsi", Key: "undangan_souvenir", Name: "Undangan & souvenir", Category: "Perlengkapan", Amounts: [6]int64{150_000, 800_000, 1_500_000, 2_500_000, 3_500_000, 3_500_000}},
	{Scope: "resepsi", Key: "dokumentasi", Name: "Dokumentasi", Category: "Dokumentasi", Amounts: [6]int64{0, 2_000_000, 2_000_000, 2_500_000, 3_000_000, 3_000_000}},
	{Scope: "resepsi", Key: "seserahan", Name: "Seserahan", Category: "Hantaran", Amounts: [6]int64{800_000, 1_600_000, 2_000_000, 3_000_000, 3_500_000, 3_500_000}},
	{Scope: "resepsi", Key: "hiburan", Name: "Hiburan & pengisi acara", Category: "Hiburan", Amounts: [6]int64{0, 1_000_000, 2_000_000, 2_400_000, 3_400_000, 4_400_000}},
	{Scope: "all", Key: "dana_darurat", Name: "Dana darurat", Category: "Operasional", Amounts: [6]int64{300_000, 500_000, 500_000, 1_000_000, 1_000_000, 1_000_000}},
}

// weddingBudgetLines are the 14 lines of Wedding_Planner.pdf p. 5-6 (03 BE 6.2). Their estimate starts at 0.
var weddingBudgetLines = []LineTemplate{
	{Scope: "lamaran", Key: "catering", Name: "Catering", Category: "Catering"},
	{Scope: "lamaran", Key: "gaun_kebaya", Name: "Gaun/Kebaya", Category: "Busana & MUA"},
	{Scope: "lamaran", Key: "fotografer", Name: "Fotografer", Category: "Dokumentasi"},
	{Scope: "lamaran", Key: "cincin", Name: "Cincin", Category: "Perhiasan"},
	{Scope: "lamaran", Key: "dekorasi", Name: "Dekorasi", Category: "Dekorasi"},
	{Scope: "akad", Key: "biaya_kua", Name: "Biaya KUA", Category: "Legalitas"},
	{Scope: "resepsi", Key: "mahar", Name: "Mahar", Category: "Mahar"},
	{Scope: "resepsi", Key: "catering", Name: "Catering", Category: "Catering"},
	{Scope: "resepsi", Key: "rias_busana", Name: "Rias & Baju Pengantin", Category: "Busana & MUA"},
	{Scope: "resepsi", Key: "venue_dekor", Name: "Sewa Gedung & Dekorasi", Category: "Venue & Dekorasi"},
	{Scope: "resepsi", Key: "undangan_souvenir", Name: "Undangan & souvenir", Category: "Perlengkapan"},
	{Scope: "resepsi", Key: "dokumentasi", Name: "Dokumentasi", Category: "Dokumentasi"},
	{Scope: "resepsi", Key: "seserahan", Name: "Seserahan", Category: "Hantaran"},
	{Scope: "resepsi", Key: "hiburan", Name: "Hiburan & Pengisi acara", Category: "Hiburan"},
}
