package api

import (
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	xls "github.com/extrame/xls"
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

func analyzeEstimateFile(path string, fileName string, size int64) ([]EstimateItem, []Finding) {
	findings := []Finding{{Title: "Файл загружен", Severity: "Info", Detail: "Файл сметы сохранён и готов к проверке."}}
	if size == 0 {
		return nil, append(findings, Finding{Title: "Пустой файл", Severity: "High", Detail: "Файл пустой, проверка невозможна."})
	}

	lowerName := strings.ToLower(fileName)
	if strings.HasSuffix(lowerName, ".xlsx") || strings.HasSuffix(lowerName, ".xlsm") {
		items, excelFindings, err := analyzeExcelWorkbook(path)
		findings = append(findings, excelFindings...)
		if err != nil {
			return nil, append(findings, Finding{Title: "Ошибка чтения Excel", Severity: "High", Detail: err.Error()})
		}
		findings = append(findings, validateItems(items)...)
		return finalizeAnalyzedItems(items, findings)
	}

	var rows [][]string
	var err error
	switch {
	case strings.HasSuffix(lowerName, ".xls"):
		rows, err = readXLSRows(path)
	case strings.HasSuffix(lowerName, ".csv"):
		rows, err = readCSVRows(path)
	case strings.HasSuffix(lowerName, ".pdf"):
		return nil, append(findings, Finding{Title: "PDF передан AI", Severity: "Info", Detail: "Backend не извлекает таблицу из PDF; исходный документ анализируется выбранной AI-моделью."})
	default:
		return nil, append(findings, Finding{Title: "Неподдерживаемый формат", Severity: "High", Detail: "Используйте XLS, XLSX, XLSM, CSV или PDF."})
	}
	if err != nil {
		return nil, append(findings, Finding{Title: "Ошибка чтения файла", Severity: "High", Detail: err.Error()})
	}
	if len(rows) == 0 {
		return nil, append(findings, Finding{Title: "Нет строк", Severity: "High", Detail: "В файле не найдены строки сметы."})
	}

	headerRow, cols := detectColumns(rows)
	if !requiredColumnsFound(cols) {
		findings = append(findings, Finding{Title: "Колонки не распознаны", Severity: "High", Detail: "Не удалось надёжно определить наименование, количество, цену и сумму. Автоматическое сопоставление первых колонок отключено."})
		return nil, findings
	}
	items := extractItems(rows, headerRow+1, cols)
	findings = append(findings, validateItems(items)...)
	return finalizeAnalyzedItems(items, findings)
}

func analyzeExcelWorkbook(path string) ([]EstimateItem, []Finding, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, nil, fmt.Errorf("в Excel файле нет листов")
	}
	maxSheets := int(envInt64("MAX_EXCEL_SHEETS", 50))
	if maxSheets < 1 {
		maxSheets = 1
	}
	if maxSheets > 200 {
		maxSheets = 200
	}
	if len(sheets) > maxSheets {
		return nil, nil, fmt.Errorf("в книге %d листов; разрешено не более %d", len(sheets), maxSheets)
	}
	maxRows := int(envInt64("MAX_EXCEL_ROWS_PER_SHEET", 100000))
	if maxRows < 100 {
		maxRows = 100
	}
	if maxRows > 500000 {
		maxRows = 500000
	}

	allItems := make([]EstimateItem, 0)
	findings := make([]Finding, 0)
	rowOffset := 0
	parsedSheets := 0
	for _, sheet := range sheets {
		rows, readErr := file.GetRows(sheet)
		if readErr != nil {
			findings = append(findings, Finding{Title: "Лист не прочитан", Severity: "High", Detail: fmt.Sprintf("Лист %q: %v", sheet, readErr)})
			continue
		}
		if len(rows) == 0 {
			continue
		}
		if len(rows) > maxRows {
			findings = append(findings, Finding{Title: "Лист слишком большой", Severity: "High", Detail: fmt.Sprintf("Лист %q содержит %d строк; лимит %d.", sheet, len(rows), maxRows)})
			rows = rows[:maxRows]
		}
		headerRow, cols := detectColumns(rows)
		if !requiredColumnsFound(cols) {
			findings = append(findings, Finding{Title: "Лист пропущен", Severity: "Medium", Detail: fmt.Sprintf("Лист %q не содержит надёжно распознанной таблицы сметы.", sheet)})
			rowOffset += len(rows) + 1
			continue
		}
		items := extractItems(rows, headerRow+1, cols)
		for index := range items {
			originalRow := items[index].Row
			items[index].Row = rowOffset + originalRow
			items[index].Name = fmt.Sprintf("Лист %s · %s", sheet, items[index].Name)
		}
		if len(items) > 0 {
			parsedSheets++
			allItems = append(allItems, items...)
			findings = append(findings, Finding{Title: "Лист прочитан", Severity: "Info", Detail: fmt.Sprintf("Лист %q: найдено %d позиций.", sheet, len(items))})
		}
		rowOffset += len(rows) + 1
	}
	if parsedSheets == 0 {
		return nil, findings, fmt.Errorf("ни на одном листе не найдена таблица сметы")
	}
	findings = append(findings, Finding{Title: "Excel обработан полностью", Severity: "Info", Detail: fmt.Sprintf("Проанализировано листов с таблицами: %d из %d.", parsedSheets, len(sheets))})
	return allItems, findings, nil
}

func finalizeAnalyzedItems(items []EstimateItem, findings []Finding) ([]EstimateItem, []Finding) {
	if len(items) == 0 {
		findings = append(findings, Finding{Title: "Позиции не найдены", Severity: "High", Detail: "Система не смогла выделить строки сметы. Проверьте структуру таблицы."})
	} else {
		findings = append(findings, Finding{Title: "Позиции прочитаны", Severity: "Info", Detail: fmt.Sprintf("Найдено строк сметы: %d. Общая сумма по найденным строкам: %.2f.", len(items), sumItemsTotal(items))})
	}
	return items, findings
}

func readXLSRows(path string) ([][]string, error) {
	workbook, err := xls.Open(path, "utf-8")
	if err != nil {
		return nil, fmt.Errorf("read XLS workbook: %w", err)
	}
	maxSheets := int(envInt64("MAX_EXCEL_SHEETS", 50))
	if maxSheets < 1 {
		maxSheets = 1
	}
	if maxSheets > 200 {
		maxSheets = 200
	}
	maxRows := int(envInt64("MAX_EXCEL_ROWS_PER_SHEET", 100000))
	if maxRows < 100 {
		maxRows = 100
	}
	if maxRows > 500000 {
		maxRows = 500000
	}
	rows := make([][]string, 0)
	for sheetIndex := 0; sheetIndex < workbook.NumSheets(); sheetIndex++ {
		if sheetIndex >= maxSheets {
			break
		}
		sheet := workbook.GetSheet(sheetIndex)
		if sheet == nil {
			continue
		}
		for rowIndex := 0; rowIndex <= int(sheet.MaxRow); rowIndex++ {
			if rowIndex >= maxRows {
				break
			}
			row := sheet.Row(rowIndex)
			if row == nil {
				continue
			}
			lastCol := row.LastCol()
			values := make([]string, 0, lastCol)
			for colIndex := 0; colIndex < lastCol; colIndex++ {
				values = append(values, strings.TrimSpace(row.Col(colIndex)))
			}
			rows = append(rows, values)
		}
		if sheetIndex+1 < workbook.NumSheets() {
			rows = append(rows, []string{})
		}
	}
	return rows, nil
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
	rows, err := reader.ReadAll()
	if err == nil && len(rows) > 0 && len(rows[0]) == 1 && strings.Contains(rows[0][0], ";") {
		if _, seekErr := file.Seek(0, 0); seekErr != nil {
			return nil, seekErr
		}
		reader = csv.NewReader(file)
		reader.Comma = ';'
		reader.FieldsPerRecord = -1
		reader.TrimLeadingSpace = true
		return reader.ReadAll()
	}
	return rows, err
}

func detectColumns(rows [][]string) (int, columnMap) {
	bestRow := 0
	bestScore := -1
	best := columnMap{Name: -1, Unit: -1, Quantity: -1, UnitPrice: -1, Total: -1}
	limit := len(rows)
	if limit > 50 {
		limit = 50
	}
	for i := 0; i < limit; i++ {
		candidate := columnMap{Name: -1, Unit: -1, Quantity: -1, UnitPrice: -1, Total: -1}
		for j, cellValue := range rows[i] {
			n := normalizeHeader(cellValue)
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
		if candidate.Name >= 0 {
			score++
		}
		if candidate.Unit >= 0 {
			score++
		}
		if candidate.Quantity >= 0 {
			score++
		}
		if candidate.UnitPrice >= 0 {
			score++
		}
		if candidate.Total >= 0 {
			score++
		}
		if score > bestScore {
			bestScore = score
			bestRow = i
			best = candidate
		}
	}
	return bestRow, best
}

func requiredColumnsFound(cols columnMap) bool {
	return cols.Name >= 0 && cols.Quantity >= 0 && cols.UnitPrice >= 0 && cols.Total >= 0
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
			limit := math.Max(0.01, math.Abs(item.Total)*0.0001)
			if diff > limit {
				findings = append(findings, Finding{Title: "Сумма строки не сходится", Severity: "High", Detail: fmt.Sprintf("Строка %d: количество × цена = %.2f, в смете указано %.2f.", item.Row, expected, item.Total)})
			}
		}
		if item.UnitPrice > 1000000 || item.Total > 5000000 {
			findings = append(findings, Finding{Title: "Крупная сумма требует проверки", Severity: "Medium", Detail: fmt.Sprintf("Строка %d: высокая цена или сумма. Проверьте обоснование.", item.Row)})
		}
		key := normalizeDuplicateKey(item.Name, item.Unit)
		if key != "" {
			if prev, exists := seen[key]; exists {
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
	b.WriteString("Проверьте строки с высоким и средним риском до оплаты или утверждения бюджета.\n")
	return os.WriteFile(estimate.ReportPath, []byte(b.String()), 0o640)
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
	if score < 0 {
		return 0
	}
	return score
}

func estimateStatus(findings []Finding) string {
	for _, finding := range findings {
		if strings.EqualFold(finding.Severity, "High") {
			return "review_required"
		}
	}
	return "ready"
}

func sumItemsTotal(items []EstimateItem) float64 {
	var total float64
	for _, item := range items {
		total += item.Total
	}
	return total
}

func normalizeHeader(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, old := range []string{" ", "-", "_", "."} {
		value = strings.ReplaceAll(value, old, "")
	}
	return value
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func cell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func parseNumber(value string) float64 {
	value = strings.TrimSpace(value)
	for _, token := range []string{" ", "\u00a0", "сом", "KGS", "kgs"} {
		value = strings.ReplaceAll(value, token, "")
	}
	if value == "" {
		return 0
	}
	comma := strings.LastIndex(value, ",")
	dot := strings.LastIndex(value, ".")
	switch {
	case comma >= 0 && dot >= 0:
		if comma > dot {
			value = strings.ReplaceAll(value, ".", "")
			value = strings.ReplaceAll(value, ",", ".")
		} else {
			value = strings.ReplaceAll(value, ",", "")
		}
	case comma >= 0:
		digitsAfter := len(value) - comma - 1
		if digitsAfter == 1 || digitsAfter == 2 {
			value = strings.ReplaceAll(value, ",", ".")
		} else {
			value = strings.ReplaceAll(value, ",", "")
		}
	case dot >= 0:
		digitsAfter := len(value) - dot - 1
		if digitsAfter == 3 && strings.Count(value, ".") == 1 {
			value = strings.ReplaceAll(value, ".", "")
		}
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func isLikelySummary(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	return n == "" || strings.Contains(n, "итого") || strings.Contains(n, "всего") || strings.Contains(n, "налог") || strings.Contains(n, "ндс")
}

func normalizeDuplicateKey(name, unit string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	if separator := strings.Index(name, " · "); separator >= 0 {
		name = name[separator+len(" · "):]
	}
	unit = strings.ToLower(strings.TrimSpace(unit))
	if len(name) < 4 {
		return ""
	}
	for _, old := range []string{",", ".", ";", ":", "-", "_", "  "} {
		name = strings.ReplaceAll(name, old, " ")
	}
	return strings.Join(strings.Fields(name), " ") + "|" + unit
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

func sanitizeFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '.' || r == '_' || r == '-' {
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
	return "est_" + hex.EncodeToString(buf)
}

func estimateWriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func estimateWriteError(w http.ResponseWriter, status int, message string) {
	estimateWriteJSON(w, status, map[string]string{"error": message})
}
