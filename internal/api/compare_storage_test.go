package api

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveCompareFileUsesProtectedTemporaryStorage(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("base", "estimate.csv")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("name,total\nitem,100")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest("POST", "/v1/estimates/compare", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatalf("parse multipart form: %v", err)
	}
	defer cleanupMultipartForm(request)

	path, name, size, err := saveCompareFile(request, "base")
	if err != nil {
		t.Fatalf("saveCompareFile returned error: %v", err)
	}
	defer removeTemporaryFile(path)

	if name != "estimate.csv" {
		t.Fatalf("unexpected sanitized name: %q", name)
	}
	if size != int64(len("name,total\nitem,100")) {
		t.Fatalf("unexpected stored size: %d", size)
	}
	absoluteTemp, err := filepath.Abs(os.TempDir())
	if err != nil {
		t.Fatalf("resolve temp directory: %v", err)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("resolve temporary file: %v", err)
	}
	relative, err := filepath.Rel(absoluteTemp, absolutePath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatalf("compare input was not stored in the temporary directory: %q", path)
	}
	if filepath.Ext(path) != ".csv" {
		t.Fatalf("temporary file must retain the safe extension, got %q", path)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat temporary file: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("temporary file is accessible to group or other users: %o", info.Mode().Perm())
	}
}

func TestSaveCompareFileCleansCreatedInputWhenSecondFileIsMissing(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("base", "estimate.csv")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	_, _ = part.Write([]byte("name,total\nitem,100"))
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest("POST", "/v1/estimates/compare", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	if err := request.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatalf("parse multipart form: %v", err)
	}
	defer cleanupMultipartForm(request)

	basePath, _, _, err := saveCompareFile(request, "base")
	if err != nil {
		t.Fatalf("save base file: %v", err)
	}
	if _, _, _, err := saveCompareFile(request, "new"); err == nil {
		t.Fatal("expected missing second file to fail")
	}
	removeTemporaryFile(basePath)
	if _, err := os.Stat(basePath); !os.IsNotExist(err) {
		t.Fatalf("temporary base file was not removed: %v", err)
	}
}
