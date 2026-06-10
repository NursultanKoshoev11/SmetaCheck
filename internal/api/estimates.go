package api

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

type Estimate struct {
	ID          string         `json:"id"`
	FileName    string         `json:"file_name"`
	Status      string         `json:"status"`
	Score       int            `json:"score"`
	FileSize    int64          `json:"file_size"`
	UploadedAt  time.Time      `json:"uploaded_at"`
	ItemsCount  int            `json:"items_count"`
	TotalAmount float64        `json:"total_amount"`
	Findings    []Finding      `json:"findings"`
	Items       []EstimateItem `json:"items,omitempty"`
	FilePath    string         `json:"-"`
	ReportPath  string         `json:"-"`
}

type EstimateItem struct {
	Row       int     `json:"row"`
	Name      string  `json:"name"`
	Unit      string  `json:"unit"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Total     float64 `json:"total"`
}

type Finding struct {
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}

type columnMap struct {
	Name      int
	Unit      int
	Quantity  int
	UnitPrice int
	Total     int
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

	items, findings := analyzeEstimateFile(storedPath, fileName, written)
	totalAmount := sumItemsTotal(items)
	estimate := Estimate{
		ID:          id,
		FileName:    fileName,
		Status:      estimateStatus(findings),
		Score:       scoreFromFindings(findings),
		FileSize:    written,
		UploadedAt:  time.Now().UTC(),
		ItemsCount:  len(items),
		TotalAmount: totalAmount,
		Findings:    findings,
		Items:       items,
		FilePath:    storedPath,
		ReportPath:  filepath.Join(reportDir, id+"_report.txt"),
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

func analyzeEstimateFile(path string, fileName string, size int64) ([]EstimateItem, []Finding) {
	findings := []Finding{{Title: "Файл загружен", Severity: "Info", Detail: "Файл сметы сохранён и готов к проверке."}}
	if size == 0 {
		return nil, append(findings, Finding{Title: "Пустой файл", Severity: "High", Detail: "Файл пустой, проверка невозможна."})
	}

	lowerName := strings.ToLower(fileName)
	var rows [][]string
	var err error
	switch {
	case strings.HasSuffix(lowerName, ".xlsx"), strings.HasSuffix(lowerName, ".xlsm"):
		rows, err = readExcelRows(path)
	case strings.HasSuffix(lowerName, ".csv"):
		rows, err = readCSVRows(path)
	case strings.HasSuffix(lowerName, ".pdf"):
		return nil, append(findings, Finding{Title: "PDF принят", Severity: "Medium", Detail: "PDF сохранён. Для точной автоматической проверки загрузите Excel или CSV версию сметы."})
	default:
		return nil, append(findings, Finding{Title: "Неподдерживаемый формат", Severity: "High", Detail: "Для production-проверки используйте XLSX, XLSM или CSV файл."})
	}
	if err != nil {
		return nil, append(findings, Finding{Title: "Ошибка чтения файла", Severity: "High", Detail: err.Error()})
	}
	if len(rows) == 0 {
		return nil, append(findings, Finding{Title: "Нет строк", Severity: "High", Detail: "В файле не найдены строки сметы."})
	}

	headerRow, cols := detectColumns(rows)
	if cols.Name < 0 || cols.Quantity < 0 || cols.UnitPrice < 0 || cols.Total < 0 {
		findings = append(findings, Finding{Title: "Не все колонки найдены", Severity: "High", Detail: "Нужны колонки: наименование, количество, цена, сумма. Проверьте заголовки файла."})
	}

	items := extractItems(rows, headerRow+1, cols)
	findings = append(findings, validateItems(items)...)
	if len(items) == 0 {
		findings = append(findings, Finding{Title: "Позиции не найдены", Severity: "High", Detail: "Система не смогла выделить строки сметы. Проверьте структуру таблицы."})
	} else {
		findings = append(findings, Finding{Title: "Позиции прочитаны", Severity: "Info", Detail: fmt.Sprintf("Найдено строк сметы: %d. Общая сумма по найденным строкам: %.2f.", len(items), sumItemsTotal(items))})
	}
	return items, findings
}

func readExcelRows(path string) ([][]string, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("в Excel файле нет листов")
	}
	var best [][]string
	for _, sheet := range sheets {
		rows, err := file.GetRows(sheet)
		if err != nil {
			continue
		}
		if len(rows) > len(best) {
			best = rows
		}
	}
	if len(best) == 0 {
		return nil, fmt.Errorf("не удалось прочитать строки Excel")
	}
	return best, nil
}

func readCSVRows(path string) ([][]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true
	return reader.ReadAll()
}

func detectColumns(rows [][]string) (int, columnMap) {
	bestRow := 0
	bestScore := -1
	best := columnMap{Name: -1, Unit: -1, Quantity: -1, UnitPrice: -1, Total: -1}
	limit := len(rows)
	if limit > 25 {
		limit = 25
	}
	for i := 0; i < limit; i++ {
		candidate := columnMap{Name: -1, Unit: -1, Quantity: -1, UnitPrice: -1, Total: -1}
		for j, cell := range rows[i] {
			n := normalizeHeader(cell)
			switch {
			case containsAny(n, "наименование", "работ", "материал", "описание", "позиция") && candidate.Name < 0:
				candidate.Name = j
			case containsAny(n, "ед", "изм", "единица") && candidate.Unit < 0:
				candidate.Unit = j
			case containsAny(n, "кол", "объем", "объём", "quantity", "qty") && candidate.Quantity < 0:
				candidate.Quantity = j
			case containsAny(n, "цена", "стоимостьед", "price") && candidate.UnitPrice < 0:
				candidate.UnitPrice = j
			case containsAny(n, "сумма", "итого", "total", "стоимость") && !containsAny(n, "ед") && candidate.Total < 0:
				candidate.Total = j
			}
		}
		score := 0
		if candidate.Name >= 0 { score++ }
		if candidate.Unit >= 0 { score++ }
		if candidate.Quantity >= 0 { score++ }
		if candidate.UnitPrice >= 0 { score++ }
		if candidate.Total >= 0 { score++ }
		if score > bestScore {
			bestScore = score
			bestRow = i
			best = candidate
		}
	}
	if bestScore < 3 {
		best = columnMap{Name: 1, Unit: 2, Quantity: 3, UnitPrice: 4, Total: 5}
		bestRow = 0
	}
	return bestRow, best
}

func extractItems(rows [][]string, start int, cols columnMap) []EstimateItem {
	items := make([]EstimateItem, 0)
	for i := start; i < len(rows); i++ {
		row := rows[i]
		name := cell(row, cols.Name)
		unit := cell(row, cols.Unit)
		qty := parseNumber(cell(row, cols.Quantity))
		price := parseNumber(cell(row, cols.UnitPrice))
		total := parseNumber(cell(row, cols.Total))
		if strings.TrimSpace(name) == "" && qty == 0 && price == 0 && total == 0 {
			continue
		}
		if isLikelySummary(name) {
			continue
		}
		items = append(items, EstimateItem{Row: i + 1, Name: name, Unit: unit, Quantity: qty, UnitPrice: price, Total: total})
	}
	return items
}

func validateItems(items []EstimateItem) []Finding {
	findings := make([]Finding, 0)
	seen := map[string]int{}
	for _, item := range items {
		if strings.TrimSpace(item.Name) == "" {
			findings = append(findings, Finding{Title: "Пустое наименование", Severity: "High", Detail: fmt.Sprintf("Строка %d: нет названия работы или материала.", item.Row)})
		}
		if strings.TrimSpace(item.Unit) == "" {
			findings = append(findings, Finding{Title: "Не указана единица измерения", Severity: "Medium", Detail: fmt.Sprintf("Строка %d: нет единицы измерения.", item.Row)})
		}
		if item.Quantity <= 0 {
			findings = append(findings, Finding{Title: "Нет количества", Severity: "High", Detail: fmt.Sprintf("Строка %d: количество равно нулю или не указано.", item.Row)})
		}
		if item.UnitPrice <= 0 {
			findings = append(findings, Finding{Title: "Нет цены", Severity: "High", Detail: fmt.Sprintf("Строка %d: цена равна нулю или не указана.", item.Row)})
		}
		if item.Total <= 0 {
			findings = append(findings, Finding{Title: "Нет суммы", Severity: "Medium", Detail: fmt.Sprintf("Строка %d: итоговая сумма не указана.", item.Row)})
		}
		expected := item.Quantity * item.UnitPrice
		if item.Quantity > 0 && item.UnitPrice > 0 && item.Total > 0 {
			diff := math.Abs(expected - item.Total)
			limit := math.Max(1, math.Abs(item.Total)*0.02)
			if diff > limit {
				findings = append(findings, Finding{Title: "Сумма строки не сходится", Severity: "High", Detail: fmt.Sprintf("Строка %d: количество × цена = %.2f, в смете указано %.2f.", item.Row, expected, item.Total)})
			}
		}
		if item.UnitPrice > 1000000 || item.Total > 5000000 {
			findings = append(findings, Finding{Title: "Крупная сумма требует проверки", Severity: "Medium", Detail: fmt.Sprintf("Строка %d: высокая цена или сумма. Проверьте обоснование.", item.Row)})
		}
		key := normalizeDuplicateKey(item.Name, item.Unit)
		if key != "" {
			if prev, ok := seen[key]; ok {
				findings = append(findings, Finding{Title: "Возможный дубль", Severity: "Medium", Detail: fmt.Sprintf("Строки %d и %d похожи по названию и единице измерения.", prev, item.Row)})
			} else {
				seen[key] = item.Row
			}
		}
	}
	return findings
}

func writeEstimateReport(estimate Estimate) error {
	var b strings.Builder
	b.WriteString("SmetaCheck KG — отчёт проверки сметы\n")
	b.WriteString("====================================\n\n")
	b.WriteString("Файл: " + estimate.FileName + "\n")
	b.WriteString("Оценка: " + fmt.Sprint(estimate.Score) + "/100\n")
	b.WriteString("Статус: " + estimate.Status + "\n")
	b.WriteString("Строк сметы: " + fmt.Sprint(estimate.ItemsCount) + "\n")
	b.WriteString(fmt.Sprintf("Сумма по найденным строкам: %.2f\n", estimate.TotalAmount))
	b.WriteString("Дата проверки: " + estimate.UploadedAt.Format(time.RFC3339) + "\n\n")
	b.WriteString("Ключевые замечания\n")
	for i, finding := range estimate.Findings {
		b.WriteString(fmt.Sprintf("%d. [%s] %s — %s\n", i+1, finding.Severity, finding.Title, finding.Detail))
	}
	b.WriteString("\nРекомендация\n")
	b.WriteString("Проверьте строки с высоким и средним риском до оплаты или утверждения бюджета. Используйте отчёт как список вопросов к прорабу, сметчику или подрядчику.\n")
	return os.WriteFile(estimate.ReportPath, []byte(b.String()), 0o640)
}

func saveEstimate(estimate Estimate) error {
	estimateStoreMu.Lock()
	defer estimateStoreMu.Unlock()
	estimates, err := loadEstimatesUnlocked()
	if err != nil { return err }
	estimates = append(estimates, estimate)
	return writeEstimatesUnlocked(estimates)
}

func findEstimate(id string) (Estimate, bool, error) {
	estimates, err := loadEstimates()
	if err != nil { return Estimate{}, false, err }
	for _, estimate := range estimates {
		if estimate.ID == id { return estimate, true, nil }
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
	if os.IsNotExist(err) { return []Estimate{}, nil }
	if err != nil { return nil, err }
	var estimates []Estimate
	if err := json.Unmarshal(data, &estimates); err != nil { return nil, err }
	return estimates, nil
}

func writeEstimatesUnlocked(estimates []Estimate) error {
	path := estimateStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil { return err }
	data, err := json.MarshalIndent(estimates, "", "  ")
	if err != nil { return err }
	return os.WriteFile(path, data, 0o640)
}

func scoreFromFindings(findings []Finding) int {
	score := 100
	for _, finding := range findings {
		switch strings.ToLower(finding.Severity) {
		case "high":
			score -= 12
		case "medium":
			score -= 6
		}
	}
	if score < 0 { return 0 }
	return score
}

func estimateStatus(findings []Finding) string {
	for _, finding := range findings {
		if strings.EqualFold(finding.Severity, "High") { return "review_required" }
	}
	return "ready"
}

func sumItemsTotal(items []EstimateItem) float64 {
	var total float64
	for _, item := range items { total += item.Total }
	return total
}

func normalizeHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "-", "")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, ".", "")
	return value
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) { return true }
	}
	return false
}

func cell(row []string, index int) string {
	if index < 0 || index >= len(row) { return "" }
	return strings.TrimSpace(row[index])
}

func parseNumber(value string) float64 {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "\u00a0", "")
	value = strings.ReplaceAll(value, "сом", "")
	value = strings.ReplaceAll(value, "KGS", "")
	value = strings.ReplaceAll(value, "kgs", "")
	if strings.Count(value, ",") == 1 && strings.Count(value, ".") == 0 { value = strings.ReplaceAll(value, ",", ".") }
	value = strings.ReplaceAll(value, ",", "")
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil { return 0 }
	return parsed
}

func isLikelySummary(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n == "" || strings.Contains(n, "итого") || strings.Contains(n, "всего") || strings.Contains(n, "налог") || strings.Contains(n, "ндс")
}

func normalizeDuplicateKey(name, unit string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	unit = strings.ToLower(strings.TrimSpace(unit))
	if len(name) < 4 { return "" }
	for _, old := range []string{",", ".", ";", ":", "-", "_", "  "} {
		name = strings.ReplaceAll(name, old, " ")
	}
	return strings.Join(strings.Fields(name), " ") + "|" + unit
}

func estimateUploadDir() string {
	if value := strings.TrimSpace(os.Getenv("UPLOAD_DIR")); value != "" { return value }
	return filepath.Join(os.TempDir(), "smetacheck", "uploads")
}

func estimateReportDir() string {
	if value := strings.TrimSpace(os.Getenv("REPORT_DIR")); value != "" { return value }
	return filepath.Join(os.TempDir(), "smetacheck", "reports")
}

func estimateStorePath() string { return filepath.Join(estimateReportDir(), "estimates.json") }

func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' { return r }
		return -1
	}, name)
	return name
}

func newEstimateID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil { return fmt.Sprintf("est_%d", time.Now().UnixNano()) }
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
