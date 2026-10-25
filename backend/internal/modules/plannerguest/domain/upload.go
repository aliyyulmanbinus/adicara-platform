package domain

import (
	"bytes"
	"encoding/base64"
	"errors"
	"strings"
)

// ErrTooLarge becomes 413 file_too_large (the decoded file is larger than MaxImportBytes).
var ErrTooLarge = errors.New("file too large")

type FileKind int

const (
	KindCSV FileKind = iota
	KindXLSX
)

// DecodeUpload opens an ImportRequest: the file name must end in .csv or .xlsx and the content must match that
// extension (.xlsx is a zip archive, "PK\x03\x04"). The HTTP body is bounded first by http.MaxBytesReader
// (7.5 MiB); this function bounds the decoded file at 5 MiB.
func DecodeUpload(r ImportRequest) ([]byte, FileKind, error) {
	name := strings.ToLower(strings.TrimSpace(r.Filename))
	var kind FileKind
	switch {
	case strings.HasSuffix(name, ".csv"):
		kind = KindCSV
	case strings.HasSuffix(name, ".xlsx"):
		kind = KindXLSX
	default:
		return nil, 0, &FileError{"filename must end with .csv or .xlsx"}
	}
	if base64.StdEncoding.DecodedLen(len(r.ContentBase64)) > MaxImportBytes+3 {
		return nil, 0, ErrTooLarge
	}
	data, err := base64.StdEncoding.DecodeString(r.ContentBase64)
	if err != nil {
		return nil, 0, &FileError{"content_base64 is not valid base64"}
	}
	if len(data) > MaxImportBytes {
		return nil, 0, ErrTooLarge
	}
	if len(data) == 0 {
		return nil, 0, &FileError{"empty file"}
	}
	isZip := bytes.HasPrefix(data, []byte("PK\x03\x04"))
	if kind == KindXLSX && !isZip {
		return nil, 0, &FileError{"the content is not an .xlsx file"}
	}
	if kind == KindCSV && isZip {
		return nil, 0, &FileError{"the content is not a CSV file"}
	}
	return data, kind, nil
}
