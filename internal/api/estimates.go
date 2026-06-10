package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Estimate struct {
	ID         string    `json:"id"`
	FileName   string    `json:"file_name"`
	Status     string    `json:"status"`
	Score      int       `json:"score"`
	FileSize   int64     `json:"file_size"`
	UploadedAt time.Time `json:"uploaded_at"`
	Findings   []Finding `json:"findings"`
	FilePath   string    `json:"-"`
	ReportPath string    `json:"-"`
}

type Finding struct {
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}

var estimateStoreMu sync.Mutex

func EstimateUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(envInt64("MAX_UPLOAD_MB", 25) * 1024 * 1024); err != nil {
		estimateWriteError(w, http.StatusBadRequest, "invalid upload form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		estimateWriteError(w, http.StatusBadRequest, "file field is required")
		return
	}
	defer file.Close()

	id := newEstimateID()
	fileName := sanitizeFileName(header.Filename)
	if fileName == "" {
		fileName = "estimate-file"
	}

	uploadDir := estimateUploadDir()
	reportDir := estimateReportDir()
	if err := os.MkdirAll(uploadDir, 0o750); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot create upload directory")
		return
	}
	if err := os.MkdirAll(reportDir, 0o750); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot create report directory")
		return
	}

	storedPath := filepath.Join(uploadDir, id+"_"+fileName)
	storedFile, err := os.OpenFile(storedPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot store uploaded file")
		return
	}
	written, copyErr := io.Copy(storedFile, file)
	closeErr := storedFile.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(storedPath)
		estimateWriteError(w, http.StatusInternalServerError, "cannot save uploaded file")
		return
	}

	findings := buildInitialFindings(fileName, written)
	estimate := Estimate{
		ID:         id,
		FileName:   fileName,
		Status:     "ready",
		Score:      scoreFromFindings(findings),
		FileSize:   written,
		UploadedAt: time.Now().UTC(),
		Findings:   findings,
		FilePath:   storedPath,
		ReportPath: filepath.Join(reportDir, id+"_report.txt"),
	}

	if err := writeEstimateReport(estimate); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot create report")
		return
	}
	if err := saveEstimate(estimate); err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot save estimate metadata")
		return
	}

	estimateWriteJSON(w, http.StatusCreated, estimate)
}

func EstimateList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		estimateWriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	estimates, err := loadEstimates()
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load estimates")
		return
	}
	sort.Slice(estimates, func(i, j int) bool { return estimates[i].UploadedAt.After(estimates[j].UploadedAt) })
	estimateWriteJSON(w, http.StatusOK, map[string]any{"estimates": estimates})
}

func EstimateDetailRouter(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/estimates/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		estimateWriteError(w, http.StatusNotFound, "estimate not found")
		return
	}

	id := parts[0]
	if len(parts) == 2 && parts[1] == "report" {
		handleEstimateReport(w, r, id)
		return
	}
	if len(parts) == 1 {
		handleEstimateDetail(w, r, id)
		return
	}

	estimateWriteError(w, http.StatusNotFound, "endpoint not found")
}

func handleEstimateDetail(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		estimateWriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	estimate, ok, err := findEstimate(id)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load estimate")
		return
	}
	if !ok {
		estimateWriteError(w, http.StatusNotFound, "estimate not found")
		return
	}
	estimateWriteJSON(w, http.StatusOK, estimate)
}

func handleEstimateReport(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		estimateWriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	estimate, ok, err := findEstimate(id)
	if err != nil {
		estimateWriteError(w, http.StatusInternalServerError, "cannot load estimate")
		return
	}
	if !ok {
		estimateWriteError(w, http.StatusNotFound, "estimate not found")
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", estimate.ID+"_report.txt"))
	http.ServeFile(w, r, estimate.ReportPath)
}

func saveEstimate(estimate Estimate) error {
	estimateStoreMu.Lock()
	defer estimateStoreMu.Unlock()

	estimates, err := loadEstimatesUnlocked()
	if err != nil {
		return err
	}
	estimates = append(estimates, estimate)
	return writeEstimatesUnlocked(estimates)
}

func findEstimate(id string) (Estimate, bool, error) {
	estimates, err := loadEstimates()
	if err != nil {
		return Estimate{}, false, err
	}
	for _, estimate := range estimates {
		if estimate.ID == id {
			return estimate, true, nil
		}
	}
	return Estimate{}, false, nil
}

func loadEstimates() ([]Estimate, error) {
	estimateStoreMu.Lock()
	defer estimateStoreMu.Unlock()
	return loadEstimatesUnlocked()
}

func loadEstimatesUnlocked() ([]Estimate, error) {
	path := estimateStorePath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []Estimate{}, nil
	}
	if err != nil {
		return nil, err
	}
	var estimates []Estimate
	if err := json.Unmarshal(data, &estimates); err != nil {
		return nil, err
	}
	return estimates, nil
}

func writeEstimatesUnlocked(estimates []Estimate) error {
	path := estimateStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	data, err := json.MarshalIndent(estimates, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o640)
}

func writeEstimateReport(estimate Estimate) error {
	var b strings.Builder
	b.WriteString("SmetaCheck KG Estimate Review\n")
	b.WriteString("================================\n\n")
	b.WriteString("File: " + estimate.FileName + "\n")
	b.WriteString("Score: " + fmt.Sprint(estimate.Score) + "\n")
	b.WriteString("Status: " + estimate.Status + "\n")
	b.WriteString("Uploaded at: " + estimate.UploadedAt.Format(time.RFC3339) + "\n\n")
	b.WriteString("Findings\n")
	for i, finding := range estimate.Findings {
		b.WriteString(fmt.Sprintf("%d. [%s] %s - %s\n", i+1, finding.Severity, finding.Title, finding.Detail))
	}
	return os.WriteFile(estimate.ReportPath, []byte(b.String()), 0o640)
}

func buildInitialFindings(fileName string, size int64) []Finding {
	findings := []Finding{
		{Title: "File stored successfully", Severity: "Info", Detail: "The estimate file was uploaded and stored for review."},
		{Title: "Manual review required", Severity: "Medium", Detail: "Automated row-level parsing is the next backend step for this file type."},
	}
	if size == 0 {
		findings = append(findings, Finding{Title: "Empty file", Severity: "High", Detail: "The uploaded file is empty and cannot be reviewed."})
	}
	lowerName := strings.ToLower(fileName)
	if !strings.HasSuffix(lowerName, ".xlsx") && !strings.HasSuffix(lowerName, ".xls") && !strings.HasSuffix(lowerName, ".pdf") {
		findings = append(findings, Finding{Title: "File type review", Severity: "Medium", Detail: "Use Excel or PDF files for the cleanest production workflow."})
	}
	return findings
}

func scoreFromFindings(findings []Finding) int {
	score := 92
	for _, finding := range findings {
		switch strings.ToLower(finding.Severity) {
		case "high":
			score -= 18
		case "medium":
			score -= 8
		}
	}
	if score < 0 {
		return 0
	}
	return score
}

func estimateUploadDir() string {
	if value := strings.TrimSpace(os.Getenv("UPLOAD_DIR")); value != "" {
		return value
	}
	return filepath.Join(os.TempDir(), "smetacheck", "uploads")
}

func estimateReportDir() string {
	if value := strings.TrimSpace(os.Getenv("REPORT_DIR")); value != "" {
		return value
	}
	return filepath.Join(os.TempDir(), "smetacheck", "reports")
}

func estimateStorePath() string {
	return filepath.Join(estimateReportDir(), "estimates.json")
}

func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			return r
		}
		return -1
	}, name)
	return name
}

func newEstimateID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("est_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}

func estimateWriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func estimateWriteError(w http.ResponseWriter, status int, message string) {
	estimateWriteJSON(w, status, map[string]string{"error": message})
}
