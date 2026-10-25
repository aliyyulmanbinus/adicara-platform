package domain

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
)

var sessions = []SessionRef{{"all", "Umum / Semua Sesi"}, {"lamaran", "Lamaran"}, {"akad", "Akad Nikah"}, {"resepsi", "Resepsi"}}

func weddingCfg(t *testing.T) ImportConfig {
	return ImportConfig{Spec: specsFromTemplateEvent(t)["wedding"], Sessions: sessions}
}

const header = "Nama Tamu;Pihak;Kategori;Sesi;No Meja;Pax;No WhatsApp;Catatan"

func plan(t *testing.T, cfg ImportConfig, csv string) ImportPlan {
	t.Helper()
	recs, err := DecodeCSV([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	p, err := BuildImportPlan(cfg, recs)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExcelCSVWithBOMAndSemicolonIsReadAsFourColumnsNotOne(t *testing.T) {
	// Excel berbahasa Indonesia menyimpan "CSV UTF-8" dengan BOM dan titik koma.
	p := plan(t, weddingCfg(t), "\xef\xbb\xbf"+header+"\r\nBpk. Prof. Bambang;Pria;VIP;Resepsi;VIP-01;2;081200000101;\r\n")
	if p.Result.ValidRows != 1 || p.Result.InvalidRows != 0 {
		t.Fatalf("%+v", p.Result)
	}
	g := p.Insert[0]
	if g.Name != "Bpk. Prof. Bambang" || *g.Side != "pria" || g.Category != "vip" || g.Scope != "resepsi" || *g.Table != "VIP-01" || g.Pax != 2 || *g.WA != "6281200000101" || *g.Phone != "081200000101" {
		t.Errorf("%+v", g)
	}
}

func TestDelimitersAndQuoting(t *testing.T) {
	want := plan(t, weddingCfg(t), header+"\nBudi;Pria;Teman;Resepsi;T-1;2;081200000102;\n").Insert[0]
	for name, sep := range map[string]string{"comma": ",", "tab": "\t"} {
		csv := strings.ReplaceAll(header+"\nBudi;Pria;Teman;Resepsi;T-1;2;081200000102;\n", ";", sep)
		got := plan(t, weddingCfg(t), csv).Insert[0]
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: %+v != %+v", name, got, want)
		}
	}
	// koma di dalam sel bertanda kutip tidak membuat koma menjadi pemisah
	p := plan(t, weddingCfg(t), "Nama Tamu,Pihak,Kategori,Pax\n\"Bpk. Hendro, S.H. & Keluarga\",Pria,Keluarga,4\n")
	if p.Result.ValidRows != 1 || p.Insert[0].Name != "Bpk. Hendro, S.H. & Keluarga" || p.Insert[0].Pax != 4 {
		t.Errorf("%+v %+v", p.Result, p.Insert)
	}
}

func TestWindows1252IsDecoded(t *testing.T) {
	// "José Núñez" disimpan Excel (Windows) sebagai CSV biasa: bukan UTF-8 yang sah.
	raw := []byte("Nama Tamu;Pax\r\nJos\xe9 N\xfa\xf1ez;2\r\n")
	recs, err := DecodeCSV(raw)
	if err != nil {
		t.Fatal(err)
	}
	if recs[1].Cells[0] != "José Núñez" {
		t.Errorf("%q", recs[1].Cells[0])
	}
}

func TestHeaderMapping(t *testing.T) {
	m, err := MapColumns([]string{" nama   PESERTA* ", "SISI", "Kategori", "no. meja", "Jumlah", "Telepon", "Keterangan", "kolom asing"})
	if err == nil && m[ColTable] != 0 {
		t.Log("'no. meja' with a dot is not an alias; the column is ignored by design")
	}
	if err != nil {
		t.Fatal(err)
	}
	if m[ColName] != 0 || m[ColSide] != 1 || m[ColPax] != 4 || m[ColPhone] != 5 || m[ColNotes] != 6 {
		t.Errorf("%v", m)
	}
	if _, ok := m[ColTable]; ok {
		t.Errorf("'no. meja' must not match (alias is 'no meja'): %v", m)
	}
	if m, _ := MapColumns([]string{"Nama Tamu", "Nama", "Pihak", "Sisi"}); m[ColName] != 0 || m[ColSide] != 2 {
		t.Errorf("a repeated heading must keep the first column: %v", m)
	}
	if _, err := MapColumns([]string{"Pihak", "Kategori"}); err == nil || err.Error() != "missing column: Nama Tamu" {
		t.Errorf("err = %v", err)
	}
	if _, err := BuildImportPlan(weddingCfg(t), nil); err == nil {
		t.Error("empty file must fail")
	}
	if _, err := DecodeCSV([]byte("Nama\x00Tamu")); err == nil {
		t.Error("NUL bytes must fail")
	}
}

func TestRowValidation(t *testing.T) {
	long := strings.Repeat("x", 256)
	csv := header + "\n" +
		";Pria;Teman;Resepsi;T-1;2;;\n" + // 2: nama kosong
		"Siti;laki-laki;Teman;Resepsi;;2;;\n" + // 3: pihak tak dikenal
		"Tono;Pria;Teman;Pesta Kebun;;2;;\n" + // 4: sesi tak dikenal
		"Ani;Pria;Teman;Resepsi;;abc;;\n" + // 5: pax bukan angka
		"Ina;Pria;Teman;Resepsi;;0;;\n" + // 6: pax 0
		"Ida;Pria;Teman;Resepsi;;21;;\n" + // 7: pax 21
		"Eka;Pria;Teman;Resepsi;;2;81200000101;\n" + // 8: telepon numerik tanpa 0 di depan
		"Ayu;Pria;Teman;Resepsi;;2;0812abc;\n" + // 9: telepon salah
		long + ";Pria;Teman;Resepsi;;2;;\n" // 10: nama 256
	p := plan(t, weddingCfg(t), csv)
	got := map[string]string{}
	for _, e := range p.Result.Errors {
		got[fmt.Sprintf("%d/%s", e.Row, e.Field)] = e.Reason
	}
	want := map[string]string{
		"2/name":            "required",
		"3/guest_side":      "must be a registry side key",
		"4/sub_event_scope": "must be a session of this project",
		"5/pax_count":       "must be an integer",
		"6/pax_count":       "must be between 1 and 20",
		"7/pax_count":       "must be between 1 and 20",
		"8/phone_number":    "must be an Indonesian phone number (08…, +62…)",
		"9/phone_number":    "must be an Indonesian phone number (08…, +62…)",
		"10/name":           "must be at most 255 characters",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q; want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) || p.Result.InvalidRows != 9 || p.Result.ValidRows != 0 || !p.Blocked || p.Insert != nil {
		t.Errorf("got %d errors, invalid=%d valid=%d blocked=%v", len(got), p.Result.InvalidRows, p.Result.ValidRows, p.Blocked)
	}
}

func TestBlankRowsAreSkippedAndRowNumbersAreFileLines(t *testing.T) {
	p := plan(t, weddingCfg(t), header+"\nAndi;Pria;Teman;;;;;\n\n;;;;;;;\nBudi;;;;;;;\n;Pria;;;;;;\n")
	if p.Result.TotalRows != 3 || p.Result.ValidRows != 2 || p.Result.InvalidRows != 1 {
		t.Fatalf("%+v", p.Result)
	}
	if p.Result.Errors[0].Row != 6 || p.Result.Errors[0].Field != "name" {
		t.Errorf("error row = %+v (file line 6)", p.Result.Errors[0])
	}
	if p.Insert == nil && !p.Blocked {
		t.Error("expected blocked")
	}
}

func TestDefaultsOfABareRow(t *testing.T) {
	p := plan(t, weddingCfg(t), "Nama Tamu\nCukup Nama\n")
	g := p.Insert[0]
	if g.Pax != 1 || g.Category != "lainnya" || g.Scope != "all" || g.Side != nil || g.Table != nil || g.WA != nil {
		t.Errorf("%+v", g)
	}
}

func TestSessionsResolveByNameOrKey(t *testing.T) {
	p := plan(t, weddingCfg(t), "Nama Tamu;Sesi\nA;resepsi\nB;AKAD NIKAH\nC;Umum\nD;Umum / Semua Sesi\nE;Semua\nF;\n")
	var got []string
	for _, g := range p.Insert {
		got = append(got, g.Scope)
	}
	if strings.Join(got, ",") != "resepsi,akad,all,all,all,all" {
		t.Errorf("%v", got)
	}
}

func TestAnEventTypeWithoutSidesRejectsAFilledPihakButNotAnEmptyOne(t *testing.T) {
	cfg := ImportConfig{Spec: specsFromTemplateEvent(t)["birthday"], Sessions: sessions}
	p := plan(t, cfg, "Nama Tamu;Pihak\nA;Pria\nB;\n")
	if p.Result.InvalidRows != 1 || p.Result.Errors[0].Reason != "must be empty for an event type without sides" || p.Result.ValidRows != 1 {
		t.Errorf("%+v", p.Result)
	}
}

func TestCustomCategoryAndRegistryLabel(t *testing.T) {
	p := plan(t, weddingCfg(t), "Nama Tamu;Kategori\nA;Teman Orang Tua\nB;Kolega  Kantor\nC;vip\nD;"+strings.Repeat("k", 61)+"\n")
	var got []string
	for _, g := range p.Insert {
		got = append(got, g.Category)
	}
	if p.Result.InvalidRows != 1 || p.Result.Errors[0].Field != "category" || p.Result.Errors[0].Row != 5 {
		t.Errorf("%+v", p.Result)
	}
	_ = got
	p = plan(t, weddingCfg(t), "Nama Tamu;Kategori\nA;Teman Orang Tua\nB;Kolega  Kantor\nC;vip\n")
	got = nil
	for _, g := range p.Insert {
		got = append(got, g.Category)
	}
	if strings.Join(got, "|") != "teman_ortu|Kolega Kantor|vip" {
		t.Errorf("%v", got)
	}
}

func TestDuplicates(t *testing.T) {
	id := uuid.New()
	cfg := weddingCfg(t)
	cfg.Existing = []ExistingGuest{{ID: id, Name: "Raditya Pratama & Pasangan", WA: "6281200000102"}, {ID: uuid.New(), Name: "Fajar Nugraha"}}
	csv := "Nama Tamu;No WhatsApp\n" +
		"Raditya  Pratama & PASANGAN;081200000102\n" + // 2: sama dengan tamu yang ada (huruf/spasi tak berpengaruh)
		"Raditya Pratama & Pasangan;081299999999\n" + // 3: nama sama, nomor lain: bukan duplikat
		"Budi;081200000150\n" + // 4: pertama
		"budi;+62 812-0000-0150\n" + // 5: duplikat baris 4 (nomor sama setelah normalisasi)
		"Fajar Nugraha;\n" + // 6: sama dengan tamu yang ada, keduanya tanpa nomor
		"Citra;\n" + // 7
		"Citra;\n" //  8: duplikat baris 7
	p := plan(t, cfg, csv)
	if p.Result.ValidRows != 7 || len(p.Result.Duplicates) != 4 {
		t.Fatalf("%+v", p.Result)
	}
	d := p.Result.Duplicates
	if d[0].Row != 2 || *d[0].OfGuest != id || d[1].Row != 5 || *d[1].OfRow != 4 || d[2].Row != 6 || d[2].OfGuest == nil || d[3].Row != 8 || *d[3].OfRow != 7 {
		t.Errorf("%+v", d)
	}
	if len(p.Insert) != 7 { // duplikat hanya dilaporkan, tetap ditulis
		t.Errorf("insert = %d", len(p.Insert))
	}
	cfg.Options.SkipDuplicates = true
	p = plan(t, cfg, csv)
	if len(p.Insert) != 3 || p.Result.Skipped != 4 {
		t.Errorf("insert=%d skipped=%d", len(p.Insert), p.Result.Skipped)
	}
}

func TestSkipInvalidWritesTheValidRows(t *testing.T) {
	csv := "Nama Tamu;Pax\nA;2\nB;99\nC;3\n"
	cfg := weddingCfg(t)
	p := plan(t, cfg, csv)
	if !p.Blocked || p.Insert != nil {
		t.Errorf("without skip_invalid nothing may be written: %+v", p)
	}
	cfg.Options.SkipInvalid = true
	p = plan(t, cfg, csv)
	if p.Blocked || len(p.Insert) != 2 || p.Result.Skipped != 1 || p.Result.InvalidRows != 1 {
		t.Errorf("%+v blocked=%v insert=%d", p.Result, p.Blocked, len(p.Insert))
	}
}

func TestRowLimitAndPreview(t *testing.T) {
	build := func(n int) string {
		var b strings.Builder
		b.WriteString("Nama Tamu\n")
		for i := 0; i < n; i++ {
			fmt.Fprintf(&b, "Tamu %d\n", i)
		}
		return b.String()
	}
	recs, _ := DecodeCSV([]byte(build(MaxImportRows)))
	p, err := BuildImportPlan(weddingCfg(t), recs)
	if err != nil || p.Result.TotalRows != 2000 || len(p.Result.Preview) != 5 {
		t.Fatalf("2000 rows: err=%v total=%d preview=%d", err, p.Result.TotalRows, len(p.Result.Preview))
	}
	recs, _ = DecodeCSV([]byte(build(MaxImportRows + 1)))
	if _, err := BuildImportPlan(weddingCfg(t), recs); err == nil || err.Error() != "too many rows (max 2000)" {
		t.Errorf("2001 rows: err = %v", err)
	}
}

func TestResultJSONHasNoNullLists(t *testing.T) {
	p := plan(t, weddingCfg(t), "Nama Tamu\nA\n")
	if p.Result.Errors == nil || p.Result.Duplicates == nil || p.Result.Preview == nil {
		t.Errorf("%+v", p.Result)
	}
}
