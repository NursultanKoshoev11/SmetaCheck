package api

import "testing"

func TestParseLocalizedNumbers(t *testing.T) {
	tests := map[string]float64{
		"1 234,56": 1234.56,
		"1.234,56": 1234.56,
		"1,234.56": 1234.56,
		"1.234": 1234,
		"1234,5": 1234.5,
	}
	for input, expected := range tests {
		if actual := parseNumber(input); actual != expected {
			t.Fatalf("parseNumber(%q): expected %.2f, got %.2f", input, expected, actual)
		}
	}
}

func TestColumnsAreNotGuessedWhenHeadersAreUnknown(t *testing.T) {
	rows := [][]string{{"A", "B", "C", "D", "E"}, {"Brick", "pcs", "10", "12", "120"}}
	_, columns := detectColumns(rows)
	if requiredColumnsFound(columns) {
		t.Fatalf("unknown columns must not be guessed: %+v", columns)
	}
}
