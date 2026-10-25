package domain

import (
	"bytes"
	"encoding/base64"
	"errors"
	"testing"
)

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }

func TestDecodeUpload(t *testing.T) {
	zipLike := append([]byte("PK\x03\x04"), make([]byte, 20)...)
	cases := []struct {
		name string
		req  ImportRequest
		kind FileKind
		err  string
	}{
		{"csv", ImportRequest{"Tamu.CSV", b64([]byte("Nama Tamu\nA\n"))}, KindCSV, ""},
		{"xlsx", ImportRequest{"tamu.xlsx", b64(zipLike)}, KindXLSX, ""},
		{"wrong extension", ImportRequest{"tamu.xls", b64([]byte("x"))}, 0, "filename must end with .csv or .xlsx"},
		{"no extension", ImportRequest{"tamu", b64([]byte("x"))}, 0, "filename must end with .csv or .xlsx"},
		{"not base64", ImportRequest{"a.csv", "***"}, 0, "content_base64 is not valid base64"},
		{"empty", ImportRequest{"a.csv", ""}, 0, "empty file"},
		{"xlsx extension, text content", ImportRequest{"a.xlsx", b64([]byte("Nama Tamu"))}, 0, "the content is not an .xlsx file"},
		{"csv extension, zip content", ImportRequest{"a.csv", b64(zipLike)}, 0, "the content is not a CSV file"},
	}
	for _, c := range cases {
		_, kind, err := DecodeUpload(c.req)
		switch {
		case c.err == "" && err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.err != "" && (err == nil || err.Error() != c.err):
			t.Errorf("%s: err = %v, want %q", c.name, err, c.err)
		case c.err == "" && kind != c.kind:
			t.Errorf("%s: kind = %v", c.name, kind)
		}
	}
}

func TestDecodeUploadLimit(t *testing.T) {
	ok := bytes.Repeat([]byte("a"), MaxImportBytes)
	if _, _, err := DecodeUpload(ImportRequest{"a.csv", b64(ok)}); err != nil {
		t.Errorf("exactly 5 MiB must pass: %v", err)
	}
	big := bytes.Repeat([]byte("a"), MaxImportBytes+1)
	if _, _, err := DecodeUpload(ImportRequest{"a.csv", b64(big)}); !errors.Is(err, ErrTooLarge) {
		t.Errorf("5 MiB + 1 byte: err = %v", err)
	}
	// 5 MiB berkas -> base64 sekitar 6,7 MiB: di bawah client_max_body_size 8m dan batas badan 7,5 MiB
	if n := len(b64(ok)); n > 7_864_320 || n < 6_900_000 {
		t.Errorf("base64 size = %d", n)
	}
}
