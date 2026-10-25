package xlsx

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

func weddingInput(t *testing.T) domain.TemplateInput {
	s := specsOf(t)
	return domain.TemplateInput{GuestWord: "Tamu", Sides: s.Sides, Cats: s.Categories, Sessions: []string{"Umum / Semua Sesi", "Lamaran", "Akad Nikah", "Resepsi"}}
}

func specsOf(t *testing.T) domain.GuestSpec {
	return domain.GuestSpec{
		Sides:      []domain.Option{{Key: "pria", Label: "Pihak Pria (CPP)"}, {Key: "wanita", Label: "Pihak Wanita (CPW)"}},
		Categories: []domain.Option{{Key: "teman", Label: "Teman"}, {Key: "kolega", Label: "Kolega"}, {Key: "keluarga", Label: "Keluarga"}, {Key: "tetangga", Label: "Tetangga"}, {Key: "teman_ortu", Label: "Teman Orang Tua"}, {Key: "vip", Label: "VIP"}},
	}
}

// Templat yang dibuat endpoint harus dapat dibaca pembaca impor: judul dikenali, tanpa baris data.
func TestTheTemplateIsReadBackByTheImporter(t *testing.T) {
	data, err := Template(weddingInput(t))
	if err != nil {
		t.Fatal(err)
	}
	recs, err := Read(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("records = %d, want only the header row", len(recs))
	}
	cols, err := domain.MapColumns(recs[0].Cells)
	if err != nil || len(cols) != 8 {
		t.Fatalf("columns = %v, err = %v", cols, err)
	}
	p, err := domain.BuildImportPlan(domain.ImportConfig{Spec: specsOf(t), Sessions: []domain.SessionRef{{ScopeKey: "all", Name: "Umum / Semua Sesi"}}}, recs)
	if err != nil || p.Result.TotalRows != 0 {
		t.Fatalf("an empty template must import 0 rows: %v %+v", err, p.Result)
	}
}

func TestTemplateDropdownsPhoneColumnAndSeminarWording(t *testing.T) {
	data, _ := Template(weddingInput(t))
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	dvs, err := f.GetDataValidations("Tamu")
	if err != nil || len(dvs) != 4 {
		t.Fatalf("validations = %d, err = %v", len(dvs), err)
	}
	found := map[string]string{}
	for _, d := range dvs {
		found[d.Sqref] = d.Formula1
	}
	want := map[string]string{"B2:B2001": "Petunjuk!$A$2:$A$3", "C2:C2001": "Petunjuk!$B$2:$B$7", "D2:D2001": "Petunjuk!$C$2:$C$5", "F2:F2001": "1"}
	for k, v := range want {
		if found[k] != v {
			t.Errorf("validation %s = %q, want %q (all: %v)", k, found[k], v, found)
		}
	}
	// kolom telepon: sel baru yang diisi pengguna bertipe teks
	idx, _ := f.GetColStyle("Tamu", "G")
	st, _ := f.GetStyle(idx)
	if st == nil || st.NumFmt != 49 {
		t.Errorf("phone column style = %+v", st)
	}
	// jenis tanpa sisi: tanpa kolom Pihak; seminar: "Nama Peserta"
	in := weddingInput(t)
	in.Sides, in.GuestWord = nil, "Peserta"
	data, _ = Template(in)
	recs, _ := Read(data)
	if len(recs[0].Cells) != 7 || recs[0].Cells[0] != "Nama Peserta" {
		t.Errorf("header = %v", recs[0].Cells)
	}
	if _, err := domain.MapColumns(recs[0].Cells); err != nil {
		t.Errorf("'Nama Peserta' must be recognised: %v", err)
	}
}
