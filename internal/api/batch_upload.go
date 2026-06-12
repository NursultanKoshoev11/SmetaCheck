package api

import (
	"context"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

func saveBatchUpload(batchID, batchDir string, header *multipart.FileHeader) (AnalysisBatchFile, error) {
	maxBytes := envInt64("MAX_UPLOAD_MB", 25) * 1024 * 1024
	if header.Size <= 0 { return AnalysisBatchFile{}, fmt.Errorf("file %q is empty", header.Filename) }
	if header.Size > maxBytes { return AnalysisBatchFile{}, fmt.Errorf("file %q exceeds maximum size", header.Filename) }
	fileName := sanitizeFileName(header.Filename)
	if fileName == "" { return AnalysisBatchFile{}, fmt.Errorf("invalid file name") }
	ext := strings.ToLower(filepath.Ext(fileName))
	if ext != ".xlsx" && ext != ".xlsm" && ext != ".csv" && ext != ".pdf" { return AnalysisBatchFile{}, fmt.Errorf("unsupported file format for %q", fileName) }
	source, err := header.Open()
	if err != nil { return AnalysisBatchFile{}, fmt.Errorf("open %q: %w", fileName, err) }
	defer source.Close()
	prefix := make([]byte, 512)
	readCount, readErr := io.ReadFull(source, prefix)
	if readErr != nil && readErr != io.ErrUnexpectedEOF { return AnalysisBatchFile{}, fmt.Errorf("read %q: %w", fileName, readErr) }
	prefix = prefix[:readCount]
	fileID := newDatabaseID("baf")
	storedPath := filepath.Join(batchDir, fileID+"_"+fileName)
	destination, err := os.OpenFile(storedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil { return AnalysisBatchFile{}, fmt.Errorf("store %q: %w", fileName, err) }
	limited := io.LimitReader(source, maxBytes-int64(len(prefix))+1)
	written, copyErr := io.Copy(destination, io.MultiReader(bytes.NewReader(prefix), limited))
	closeErr := destination.Close()
	if copyErr != nil || closeErr != nil { _ = os.Remove(storedPath); return AnalysisBatchFile{}, fmt.Errorf("save %q", fileName) }
	if written > maxBytes { _ = os.Remove(storedPath); return AnalysisBatchFile{}, fmt.Errorf("file %q exceeds maximum size", fileName) }
	inspection, err := inspectUploadedFile(context.Background(), storedPath, fileName)
	if err != nil { _ = os.Remove(storedPath); return AnalysisBatchFile{}, fmt.Errorf("%s: %w", fileName, err) }
	return AnalysisBatchFile{ID: fileID, BatchID: batchID, FileName: fileName, FilePath: storedPath, MIMEType: inspection.MIMEType, FileSize: written, Status: "pending"}, nil
}

func publicBatchFiles(files []AnalysisBatchFile) []AnalysisBatchFile {
	result := make([]AnalysisBatchFile, len(files))
	copy(result, files)
	for index := range result { result[index].FilePath = "" }
	return result
}
