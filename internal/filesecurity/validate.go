package filesecurity

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var ErrMalwareDetected = errors.New("malware detected")

type Limits struct {
	MaxArchiveFiles      int
	MaxUncompressedBytes int64
	MaxEntryBytes        int64
	MaxCompressionRatio  float64
	MaxNestedArchives    int
	InspectionTimeout    time.Duration
	AllowXLSM            bool
}

type Result struct {
	MIMEType  string
	SHA256    string
	HasMacros bool
}

func DefaultLimits() Limits {
	return Limits{
		MaxArchiveFiles:      2000,
		MaxUncompressedBytes: 200 * 1024 * 1024,
		MaxEntryBytes:        50 * 1024 * 1024,
		MaxCompressionRatio:  100,
		MaxNestedArchives:    0,
		InspectionTimeout:    5 * time.Second,
		AllowXLSM:            true,
	}
}

func Validate(path, originalName string, limits Limits) (Result, error) {
	limits = normalizeLimits(limits)
	info, err := os.Stat(path)
	if err != nil {
		return Result{}, fmt.Errorf("stat uploaded file: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 {
		return Result{}, fmt.Errorf("uploaded file is empty or not a regular file")
	}

	digest, prefix, err := digestAndPrefix(path)
	if err != nil {
		return Result{}, err
	}
	result := Result{SHA256: digest}
	ext := strings.ToLower(filepath.Ext(originalName))
	switch ext {
	case ".csv":
		if err := validateCSV(path, prefix); err != nil {
			return Result{}, err
		}
		result.MIMEType = "text/csv"
		return result, nil
	case ".xls":
		if !bytes.HasPrefix(prefix, []byte{208, 207, 17, 224, 161, 177, 26, 225}) {
			return Result{}, fmt.Errorf("invalid XLS file")
		}
		result.MIMEType = "application/vnd.ms-excel"
		return result, nil
	case ".xlsx", ".xlsm":
		if ext == ".xlsm" && !limits.AllowXLSM {
			return Result{}, fmt.Errorf("XLSM files are disabled")
		}
		if !bytes.HasPrefix(prefix, []byte("PK\x03\x04")) && !bytes.HasPrefix(prefix, []byte("PK\x05\x06")) {
			return Result{}, fmt.Errorf("file content is not an Office Open XML archive")
		}
		hasMacros, mimeType, err := validateOfficeArchive(path, ext, limits)
		if err != nil {
			return Result{}, err
		}
		result.HasMacros = hasMacros
		result.MIMEType = mimeType
		return result, nil
	case ".pdf":
		if !bytes.HasPrefix(prefix, []byte("%PDF-")) {
			return Result{}, fmt.Errorf("file content is not a PDF")
		}
		result.MIMEType = "application/pdf"
		return result, nil
	default:
		return Result{}, fmt.Errorf("unsupported file format: use XLS, XLSX, XLSM, CSV or PDF")
	}
}

func normalizeLimits(l Limits) Limits {
	d := DefaultLimits()
	if l.MaxArchiveFiles <= 0 {
		l.MaxArchiveFiles = d.MaxArchiveFiles
	}
	if l.MaxUncompressedBytes <= 0 {
		l.MaxUncompressedBytes = d.MaxUncompressedBytes
	}
	if l.MaxEntryBytes <= 0 {
		l.MaxEntryBytes = d.MaxEntryBytes
	}
	if l.MaxCompressionRatio <= 0 {
		l.MaxCompressionRatio = d.MaxCompressionRatio
	}
	if l.MaxNestedArchives < 0 {
		l.MaxNestedArchives = d.MaxNestedArchives
	}
	if l.InspectionTimeout <= 0 {
		l.InspectionTimeout = d.InspectionTimeout
	}
	return l
}

func digestAndPrefix(path string) (string, []byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("open uploaded file: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	prefix := make([]byte, 512)
	n, readErr := io.ReadFull(f, prefix)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		return "", nil, fmt.Errorf("read uploaded file: %w", readErr)
	}
	prefix = prefix[:n]
	_, _ = h.Write(prefix)
	if _, err := io.Copy(h, f); err != nil {
		return "", nil, fmt.Errorf("hash uploaded file: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)), prefix, nil
}

func validateCSV(path string, prefix []byte) error {
	if bytes.IndexByte(prefix, 0) >= 0 || bytes.HasPrefix(prefix, []byte("MZ")) || bytes.HasPrefix(prefix, []byte("PK\x03\x04")) || bytes.HasPrefix(prefix, []byte("%PDF-")) {
		return fmt.Errorf("CSV extension does not match text content")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	sample, err := io.ReadAll(io.LimitReader(f, 1024*1024))
	if err != nil {
		return fmt.Errorf("inspect CSV: %w", err)
	}
	control := 0
	for _, b := range sample {
		if b < 0x20 && b != '\r' && b != '\n' && b != '\t' {
			control++
		}
	}
	if len(sample) > 0 && control*100/len(sample) > 2 {
		return fmt.Errorf("CSV contains too many control bytes")
	}
	return nil
}

func validateOfficeArchive(path, ext string, limits Limits) (bool, string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return false, "", fmt.Errorf("invalid Office archive: %w", err)
	}
	defer reader.Close()
	if len(reader.File) == 0 || len(reader.File) > limits.MaxArchiveFiles {
		return false, "", fmt.Errorf("Office archive contains an invalid number of entries")
	}
	deadline := time.Now().Add(limits.InspectionTimeout)
	var totalUncompressed uint64
	var contentTypes []byte
	hasWorkbook := false
	hasMacros := false
	nestedArchives := 0

	for _, entry := range reader.File {
		if time.Now().After(deadline) {
			return false, "", fmt.Errorf("Office archive inspection timed out")
		}
		clean := filepath.ToSlash(filepath.Clean(entry.Name))
		if clean == "." || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") {
			return false, "", fmt.Errorf("Office archive contains an unsafe path")
		}
		if entry.UncompressedSize64 > uint64(limits.MaxEntryBytes) {
			return false, "", fmt.Errorf("Office archive entry exceeds the maximum expanded size")
		}
		totalUncompressed += entry.UncompressedSize64
		if totalUncompressed > uint64(limits.MaxUncompressedBytes) {
			return false, "", fmt.Errorf("Office archive exceeds the maximum expanded size")
		}
		if entry.UncompressedSize64 > 0 {
			if entry.CompressedSize64 == 0 {
				return false, "", fmt.Errorf("Office archive contains an invalid compressed entry")
			}
			ratio := float64(entry.UncompressedSize64) / float64(entry.CompressedSize64)
			if ratio > limits.MaxCompressionRatio {
				return false, "", fmt.Errorf("Office archive compression ratio is unsafe")
			}
		}
		lower := strings.ToLower(clean)
		if lower == "xl/workbook.xml" {
			hasWorkbook = true
		}
		if lower == "xl/vbaproject.bin" {
			hasMacros = true
		}
		if isNestedArchiveName(lower) {
			nestedArchives++
			if nestedArchives > limits.MaxNestedArchives {
				return false, "", fmt.Errorf("nested archives are not allowed inside Office files")
			}
		}
		if lower == "[content_types].xml" {
			contentTypes, err = readZipEntryLimited(entry, 2*1024*1024)
			if err != nil {
				return false, "", fmt.Errorf("read Office content types: %w", err)
			}
		}
	}
	if !hasWorkbook || len(contentTypes) == 0 {
		return false, "", fmt.Errorf("archive is not a valid Excel workbook")
	}
	content := strings.ToLower(string(contentTypes))
	isMacroType := strings.Contains(content, "application/vnd.ms-excel.sheet.macroenabled.main+xml")
	isXLSXType := strings.Contains(content, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml")
	if strings.Contains(content, "application/vnd.ms-office.vbaproject") {
		hasMacros = true
	}
	if ext == ".xlsx" {
		if !isXLSXType || isMacroType || hasMacros {
			return false, "", fmt.Errorf("XLSX extension does not match workbook content or macros are present")
		}
		return false, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", nil
	}
	if !isMacroType {
		return false, "", fmt.Errorf("XLSM extension does not match workbook content")
	}
	return hasMacros, "application/vnd.ms-excel.sheet.macroEnabled.12", nil
}

func readZipEntryLimited(entry *zip.File, max int64) ([]byte, error) {
	r, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, max+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > max {
		return nil, fmt.Errorf("entry is too large")
	}
	return data, nil
}

func isNestedArchiveName(name string) bool {
	for _, suffix := range []string{".zip", ".7z", ".rar", ".tar", ".tgz", ".gz", ".bz2", ".xz"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

func ScanClamAV(ctx context.Context, path, address string, maxBytes int64) error {
	address = strings.TrimSpace(address)
	if address == "" {
		return fmt.Errorf("ClamAV address is not configured")
	}
	if maxBytes <= 0 {
		maxBytes = 100 * 1024 * 1024
	}
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("connect to ClamAV: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if _, err := conn.Write([]byte("zINSTREAM\x00")); err != nil {
		return fmt.Errorf("start ClamAV scan: %w", err)
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file for antivirus scan: %w", err)
	}
	defer f.Close()
	buf := make([]byte, 32*1024)
	var sent int64
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			sent += int64(n)
			if sent > maxBytes {
				return fmt.Errorf("file exceeds antivirus scan limit")
			}
			var size [4]byte
			binary.BigEndian.PutUint32(size[:], uint32(n))
			if _, err := conn.Write(size[:]); err != nil {
				return fmt.Errorf("stream size to ClamAV: %w", err)
			}
			if _, err := conn.Write(buf[:n]); err != nil {
				return fmt.Errorf("stream file to ClamAV: %w", err)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read file for ClamAV: %w", readErr)
		}
	}
	if _, err := conn.Write([]byte{0, 0, 0, 0}); err != nil {
		return fmt.Errorf("finish ClamAV stream: %w", err)
	}
	response, err := io.ReadAll(io.LimitReader(conn, 4096))
	if err != nil {
		return fmt.Errorf("read ClamAV response: %w", err)
	}
	text := strings.TrimSpace(strings.TrimRight(string(response), "\x00"))
	if strings.Contains(text, "FOUND") {
		return fmt.Errorf("%w: %s", ErrMalwareDetected, text)
	}
	if !strings.HasSuffix(text, "OK") {
		return fmt.Errorf("unexpected ClamAV response: %s", text)
	}
	return nil
}
