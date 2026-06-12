package api

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/NursultanKoshoev11/SmetaCheck/internal/filesecurity"
)

func inspectUploadedFile(ctx context.Context, path, originalName string) (filesecurity.Result, error) {
	limits := filesecurity.DefaultLimits()
	limits.MaxArchiveFiles = int(envInt64("ZIP_MAX_FILES", 2000))
	limits.MaxUncompressedBytes = envInt64("ZIP_MAX_UNCOMPRESSED_MB", 200) * 1024 * 1024
	limits.MaxEntryBytes = envInt64("ZIP_MAX_ENTRY_MB", 50) * 1024 * 1024
	limits.MaxCompressionRatio = float64(envInt64("ZIP_MAX_COMPRESSION_RATIO", 100))
	limits.MaxNestedArchives = int(envInt64("ZIP_MAX_NESTED_ARCHIVES", 0))
	limits.InspectionTimeout = envDuration("FILE_INSPECTION_TIMEOUT", 5*time.Second)
	limits.AllowXLSM = !strings.EqualFold(strings.TrimSpace(os.Getenv("XLSM_POLICY")), "reject")

	result, err := filesecurity.Validate(path, originalName, limits)
	if err != nil {
		return filesecurity.Result{}, fmt.Errorf("file validation failed: %w", err)
	}
	if !antivirusEnabled() {
		return result, nil
	}

	scanCtx, cancel := context.WithTimeout(ctx, envDuration("ANTIVIRUS_TIMEOUT", 30*time.Second))
	defer cancel()
	err = filesecurity.ScanClamAV(scanCtx, path, strings.TrimSpace(os.Getenv("CLAMAV_ADDRESS")), envInt64("ANTIVIRUS_MAX_FILE_MB", 100)*1024*1024)
	if err == nil {
		return result, nil
	}
	if errors.Is(err, filesecurity.ErrMalwareDetected) {
		return filesecurity.Result{}, fmt.Errorf("file was rejected by antivirus")
	}
	if antivirusFailOpen() {
		return result, nil
	}
	return filesecurity.Result{}, fmt.Errorf("antivirus scan failed: %w", err)
}

func antivirusEnabled() bool {
	value := strings.TrimSpace(os.Getenv("ANTIVIRUS_ENABLED"))
	if value == "" {
		return strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
	}
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func antivirusFailOpen() bool {
	value := strings.TrimSpace(os.Getenv("ANTIVIRUS_FAIL_OPEN"))
	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed && !strings.EqualFold(strings.TrimSpace(os.Getenv("APP_ENV")), "production")
}
