package xlsx

import (
	"bytes"

	"github.com/xuri/excelize/v2"

	"github.com/aliyyulmanbinus/adicara-platform/backend/internal/modules/plannerguest/domain"
)

// Extraction limits: an .xlsx file is a zip archive, so 5 MB compressed can claim gigabytes of XML. excelize allows
// 16 GB by default (tested: 1 MiB compressed that claims 300 MiB is extracted to a temporary directory). A template of
// 2,000 rows x 8 columns is only about 1 MB uncompressed.
const (
	unzipSizeLimit    = 64 << 20
	unzipXMLSizeLimit = 16 << 20
)

// Read takes the first sheet as text records. Blank rows are skipped (the row numbers stay right) and reading stops
// after MaxImportRows+1 non-blank rows (header + data), so a very long sheet is not read whole; BuildImportPlan then
// refuses a file with too many rows.
func Read(data []byte) ([]domain.Record, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{UnzipSizeLimit: unzipSizeLimit, UnzipXMLSizeLimit: unzipXMLSizeLimit})
	if err != nil {
		return nil, &domain.FileError{Reason: "not a readable .xlsx file"}
	}
	defer f.Close() // removes the temporary files of the extraction

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, &domain.FileError{Reason: "the workbook has no sheet"}
	}
	rows, err := f.Rows(sheets[0])
	if err != nil {
		return nil, &domain.FileError{Reason: "not a readable .xlsx file"}
	}
	defer rows.Close()

	var out []domain.Record
	n := 0
	for rows.Next() {
		n++
		cells, err := rows.Columns()
		if err != nil {
			return nil, &domain.FileError{Reason: "not a readable .xlsx file"}
		}
		blank := true
		for _, c := range cells {
			if len(bytes.TrimSpace([]byte(c))) > 0 {
				blank = false
				break
			}
		}
		if blank {
			continue
		}
		out = append(out, domain.Record{Row: n, Cells: cells})
		if len(out) > domain.MaxImportRows+1 {
			break
		}
	}
	return out, nil
}
