package filesecurity

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCSVRejectsBinary(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.csv")
	if err := os.WriteFile(path, []byte{'M', 'Z', 0, 1}, 0o600); err != nil { t.Fatal(err) }
	if _, err := Validate(path, "bad.csv", DefaultLimits()); err == nil { t.Fatal("expected binary CSV rejection") }
}

func TestValidateXLSX(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	writeOfficeArchive(t, path, map[string]string{
		"[Content_Types].xml": `<Types><Override ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/></Types>`,
		"xl/workbook.xml": `<workbook/>`,
	})
	result, err := Validate(path, "book.xlsx", DefaultLimits())
	if err != nil { t.Fatal(err) }
	if !strings.Contains(result.MIMEType, "spreadsheetml") { t.Fatalf("unexpected MIME: %s", result.MIMEType) }
}

func TestValidateRejectsNestedArchive(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	writeOfficeArchive(t, path, map[string]string{
		"[Content_Types].xml": `<Types><Override ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/></Types>`,
		"xl/workbook.xml": `<workbook/>`,
		"xl/embeddings/archive.zip": "nested",
	})
	if _, err := Validate(path, "book.xlsx", DefaultLimits()); err == nil { t.Fatal("expected nested archive rejection") }
}

func TestValidateRejectsMacrosHiddenAsXLSX(t *testing.T) {
	path := filepath.Join(t.TempDir(), "book.xlsx")
	writeOfficeArchive(t, path, map[string]string{
		"[Content_Types].xml": `<Types><Override ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override ContentType="application/vnd.ms-office.vbaProject"/></Types>`,
		"xl/workbook.xml": `<workbook/>`,
		"xl/vbaProject.bin": "macro",
	})
	if _, err := Validate(path, "book.xlsx", DefaultLimits()); err == nil { t.Fatal("expected macro rejection") }
}

func writeOfficeArchive(t *testing.T, path string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil { t.Fatal(err) }
	w := zip.NewWriter(f)
	for name, content := range entries {
		e, err := w.Create(name)
		if err != nil { t.Fatal(err) }
		if _, err := e.Write([]byte(content)); err != nil { t.Fatal(err) }
	}
	if err := w.Close(); err != nil { t.Fatal(err) }
	if err := f.Close(); err != nil { t.Fatal(err) }
}
