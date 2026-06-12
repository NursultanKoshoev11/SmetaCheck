package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CompareResponse struct {
	ID         string         `json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	BaseFile   string         `json:"base_file"`
	NewFile    string         `json:"new_file"`
	BaseTotal  float64        `json:"base_total"`
	NewTotal   float64        `json:"new_total"`
	DeltaTotal float64        `json:"delta_total"`
	Added      []EstimateItem `json:"added"`
	Removed    []EstimateItem `json:"removed"`
	Changed    []ChangedItem  `json:"changed"`
	Findings   []Finding      `json:"findings"`
}

type ChangedItem struct {
	Name       string  `json:"name"`
	BaseRow    int     `json:"base_row"`
	NewRow     int     `json:"new_row"`
	BaseTotal  float64 `json:"base_total"`
	NewTotal   float64 `json:"new_total"`
	DeltaTotal float64 `json:"delta_total"`
}

func EstimateCompare(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(envInt64("MAX_UPLOAD_MB", 25) * 1024 * 1024 * 2); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid compare form")
		return
	}
	defer cleanupMultipartForm(r)

	basePath, baseName, baseSize, err := saveCompareFile(r, "base")
	if err != nil {
		estimateWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer removeTemporaryFile(basePath)

	newPath, newName, newSize, err := saveCompareFile(r, "new")
	if err != nil {
		estimateWriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	defer removeTemporaryFile(newPath)

	baseItems, baseFindings := analyzeEstimateFile(basePath, baseName, baseSize)
	newItems, newFindings := analyzeEstimateFile(newPath, newName, newSize)
	response := compareEstimateItems(baseName, newName, baseItems, newItems)
	response.Findings = append(response.Findings, baseFindings...)
	response.Findings = append(response.Findings, newFindings...)
	estimateWriteJSON(w, http.StatusOK, response)
}

func saveCompareFile(r *http.Request, field string) (string, string, int64, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		return "", "", 0, fmt.Errorf("file field %q is required", field)
	}
	defer file.Close()
	maxBytes := envInt64("MAX_UPLOAD_MB", 25) * 1024 * 1024
	if header.Size <= 0 {
		return "", "", 0, fmt.Errorf("%s file is empty", field)
	}
	if header.Size > maxBytes {
		return "", "", 0, fmt.Errorf("%s file exceeds maximum size", field)
	}

	fileName := sanitizeFileName(header.Filename)
	if fileName == "" {
		fileName = field + "-estimate"
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	if extension != ".xlsx" && extension != ".xlsm" && extension != ".csv" && extension != ".pdf" {
		return "", "", 0, fmt.Errorf("unsupported %s file format", field)
	}
	temporary, err := os.CreateTemp("", "smetacheck-compare-*"+extension)
	if err != nil {
		return "", "", 0, fmt.Errorf("cannot create temporary %s file", field)
	}
	path := temporary.Name()
	written, copyErr := io.Copy(temporary, io.LimitReader(file, maxBytes+1))
	closeErr := temporary.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return "", "", 0, fmt.Errorf("cannot write %s file", field)
	}
	if written > maxBytes {
		_ = os.Remove(path)
		return "", "", 0, fmt.Errorf("%s file exceeds maximum size", field)
	}
	if _, err := inspectUploadedFile(r.Context(), path, fileName); err != nil {
		_ = os.Remove(path)
		return "", "", 0, fmt.Errorf("%s file: %w", field, err)
	}
	return path, fileName, written, nil
}

func cleanupMultipartForm(r *http.Request) {
	if r != nil && r.MultipartForm != nil {
		_ = r.MultipartForm.RemoveAll()
	}
}

func removeTemporaryFile(path string) {
	if strings.TrimSpace(path) != "" {
		_ = os.Remove(path)
	}
}

func compareEstimateItems(baseName, newName string, baseItems, newItems []EstimateItem) CompareResponse {
	baseMap := map[string]EstimateItem{}
	newMap := map[string]EstimateItem{}
	for _, item := range baseItems {
		key := compareKey(item)
		if key != "" { baseMap[key] = item }
	}
	for _, item := range newItems {
		key := compareKey(item)
		if key != "" { newMap[key] = item }
	}

	response := CompareResponse{ID: newEstimateID(), CreatedAt: time.Now().UTC(), BaseFile: baseName, NewFile: newName, BaseTotal: sumItemsTotal(baseItems), NewTotal: sumItemsTotal(newItems)}
	response.DeltaTotal = response.NewTotal - response.BaseTotal
	for key, newItem := range newMap {
		baseItem, ok := baseMap[key]
		if !ok { response.Added = append(response.Added, newItem); continue }
		delta := newItem.Total - baseItem.Total
		if delta > 1 || delta < -1 { response.Changed = append(response.Changed, ChangedItem{Name: newItem.Name, BaseRow: baseItem.Row, NewRow: newItem.Row, BaseTotal: baseItem.Total, NewTotal: newItem.Total, DeltaTotal: delta}) }
	}
	for key, baseItem := range baseMap {
		if _, ok := newMap[key]; !ok { response.Removed = append(response.Removed, baseItem) }
	}
	if len(response.Added) > 0 { response.Findings = append(response.Findings, Finding{Title: "Добавлены позиции", Severity: "Medium", Detail: fmt.Sprintf("В новой смете добавлено позиций: %d.", len(response.Added))}) }
	if len(response.Removed) > 0 { response.Findings = append(response.Findings, Finding{Title: "Удалены позиции", Severity: "Medium", Detail: fmt.Sprintf("Из новой сметы удалено позиций: %d.", len(response.Removed))}) }
	if len(response.Changed) > 0 { response.Findings = append(response.Findings, Finding{Title: "Изменены суммы", Severity: "High", Detail: fmt.Sprintf("Позиции с изменённой суммой: %d.", len(response.Changed))}) }
	if response.DeltaTotal != 0 { response.Findings = append(response.Findings, Finding{Title: "Изменился общий бюджет", Severity: "High", Detail: fmt.Sprintf("Разница между версиями: %.2f.", response.DeltaTotal)}) }
	return response
}

func compareKey(item EstimateItem) string {
	name := strings.ToLower(strings.TrimSpace(item.Name))
	unit := strings.ToLower(strings.TrimSpace(item.Unit))
	if len(name) < 4 { return "" }
	return normalizeDuplicateKey(name, unit)
}
