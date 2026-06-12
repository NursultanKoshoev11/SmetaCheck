package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveStoredFileWithinRoot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "estimate.xlsx")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatalf("create nested directory: %v", err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatalf("create test file: %v", err)
	}
	if err := removeStoredFileWithinRoot(path, root); err != nil {
		t.Fatalf("removeStoredFileWithinRoot returned error: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected stored file to be deleted, stat error: %v", err)
	}
	if err := removeStoredFileWithinRoot(path, root); err != nil {
		t.Fatalf("removing an already missing file must be idempotent: %v", err)
	}
}

func TestRemoveStoredFileWithinRootRejectsOutsidePath(t *testing.T) {
	root := t.TempDir()
	outsideRoot := t.TempDir()
	outsidePath := filepath.Join(outsideRoot, "outside.txt")
	if err := os.WriteFile(outsidePath, []byte("keep"), 0o600); err != nil {
		t.Fatalf("create outside test file: %v", err)
	}
	if err := removeStoredFileWithinRoot(outsidePath, root); err == nil {
		t.Fatal("expected outside path to be rejected")
	}
	if _, err := os.Stat(outsidePath); err != nil {
		t.Fatalf("outside file must remain untouched: %v", err)
	}
}

func TestRemoveStoredFileWithinRootRejectsRootAndDirectory(t *testing.T) {
	root := t.TempDir()
	if err := removeStoredFileWithinRoot(root, root); err == nil {
		t.Fatal("expected root directory deletion to be rejected")
	}
	directory := filepath.Join(root, "directory")
	if err := os.Mkdir(directory, 0o750); err != nil {
		t.Fatalf("create directory: %v", err)
	}
	if err := removeStoredFileWithinRoot(directory, root); err == nil {
		t.Fatal("expected directory deletion to be rejected")
	}
}
