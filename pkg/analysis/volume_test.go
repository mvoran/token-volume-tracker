package analysis

import (
	"encoding/csv"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testDate is used for consistent testing.
var testDate = time.Date(2025, 3, 13, 0, 0, 0, 0, time.UTC)

// setupTestTime overrides the timeNow function to return our fixed test date.
func setupTestTime() func() {
	originalTimeNow := timeNow
	timeNow = func() time.Time {
		return testDate
	}
	return func() {
		timeNow = originalTimeNow
	}
}

// getTestDataPath returns the path to the test data directory.
func getTestDataPath() string {
	// Get the current directory
	currentDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// Go up two levels to reach the project root
	return filepath.Join(filepath.Dir(filepath.Dir(currentDir)), "testdata")
}

// --- Unit Tests ---

func TestDetectDataSource(t *testing.T) {
	tests := []struct {
		name     string
		header   []string
		expected DataSource
	}{
		{
			name:     "CoinGecko format",
			header:   []string{"snapped_at", "price", "market_cap", "total_volume"},
			expected: CoinGecko,
		},
		{
			name: "CoinMarketCap format",
			header: []string{
				"timeOpen", "timeClose", "timeHigh", "timeLow",
				"name", "open", "high", "low", "close", "volume", "marketCap", "timestamp",
			},
			expected: CoinMarketCap,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build CSV header string.
			csvData := strings.Join(tt.header, ",") + "\n"
			reader := csv.NewReader(strings.NewReader(csvData))

			ds, err := detectDataSource(reader)
			if err != nil {
				t.Fatalf("detectDataSource() error = %v", err)
			}
			if ds != tt.expected {
				t.Errorf("detectDataSource() = %v, want %v", ds, tt.expected)
			}
		})
	}
}

func TestParseRecord(t *testing.T) {
	tests := []struct {
		name     string
		source   DataSource
		record   []string
		wantTime time.Time
		wantVol  float64
		wantErr  bool
	}{
		{
			name:   "CoinGecko valid",
			source: CoinGecko,
			record: []string{
				"2024-03-12 00:00:00 UTC",
				"1.23",
				"1000000",
				"50000",
			},
			wantTime: time.Date(2024, 3, 12, 0, 0, 0, 0, time.UTC),
			wantVol:  50000,
			wantErr:  false,
		},
		{
			name:   "CoinMarketCap valid",
			source: CoinMarketCap,
			record: []string{
				`"2024-03-12T00:00:00Z"`,
				"dummy", "dummy", "dummy", "dummy", "dummy", "dummy", "dummy", "dummy",
				"75000",
				"dummy",
				"dummy",
			},
			wantTime: time.Date(2024, 3, 12, 0, 0, 0, 0, time.UTC),
			wantVol:  75000,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTime, gotVol, err := parseRecord(tt.record, tt.source)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !gotTime.Equal(tt.wantTime) {
				t.Errorf("parseRecord() time = %v, want %v", gotTime, tt.wantTime)
			}
			if gotVol != tt.wantVol {
				t.Errorf("parseRecord() volume = %v, want %v", gotVol, tt.wantVol)
			}
		})
	}
}

func TestFillMissingDays(t *testing.T) {
	// Create a dataset with a gap in the middle.
	records := []VolumeData{
		{
			Name:   "TEST",
			Date:   time.Date(2025, 3, 10, 0, 0, 0, 0, time.UTC),
			Volume: 100,
		},
		{
			Name:   "TEST",
			Date:   time.Date(2025, 3, 12, 0, 0, 0, 0, time.UTC),
			Volume: 200,
		},
	}
	filled := fillMissingDays(records, "TEST", testDate)

	// Now expecting 364 days total
	if len(filled) != 364 {
		t.Fatalf("expected 364 records, got %d", len(filled))
	}

	// Check that our specific test dates are present and correct
	datesToCheck := map[string]float64{
		"2025-03-10": 100,
		"2025-03-11": 0, // This was the missing day, should be filled with zero
		"2025-03-12": 200,
	}

	for _, rec := range filled {
		dateStr := rec.Date.Format("2006-01-02")
		if expectedVolume, exists := datesToCheck[dateStr]; exists {
			if rec.Volume != expectedVolume {
				t.Errorf("expected volume %v for date %s, got %v", expectedVolume, dateStr, rec.Volume)
			}
		}
	}

	// Check that the date range is correct
	firstDate := filled[0].Date
	lastDate := filled[len(filled)-1].Date
	expectedFirstDate := testDate.AddDate(0, 0, -363)

	if !firstDate.Equal(expectedFirstDate) {
		t.Errorf("expected first date %v, got %v", expectedFirstDate, firstDate)
	}

	if !lastDate.Equal(testDate) {
		t.Errorf("expected last date %v, got %v", testDate, lastDate)
	}
}

// --- Integration Tests ---

// TestCalculateRollingAverages_CoinMarketCap runs the full calculation
// for a known CoinMarketCap input and compares against expected output.
// The sample input and expected output are stored in testdata/coinmarketcap.
func TestCalculateRollingAverages_CoinMarketCap(t *testing.T) {
	defer setupTestTime()() // Force timeNow() to return 3/12/2025 during this test.

	testDataPath := getTestDataPath()
	inputPath := filepath.Join(testDataPath, "coinmarketcap", "MAID_3_13_2024-3_13_2025_historical_data_coinmarketcap.csv")
	expectedOutputPath := filepath.Join(testDataPath, "coinmarketcap", "expected_MAID_Trading_Average.csv")

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "MAID_Trading_Average.csv")

	if err := CalculateRollingAverages(inputPath, outputPath); err != nil {
		t.Fatalf("CalculateRollingAverages() error: %v", err)
	}

	compareCSVFiles(t, expectedOutputPath, outputPath)
}

// TestCalculateRollingAverages_CoinGecko runs the full calculation
// for a known CoinGecko input and compares against expected output.
// The sample input and expected output are stored in testdata/coingecko.
func TestCalculateRollingAverages_CoinGecko(t *testing.T) {
	defer setupTestTime()() // Force timeNow() to return 3/12/2025 during this test.

	testDataPath := getTestDataPath()
	inputPath := filepath.Join(testDataPath, "coingecko", "THC_usd-max.csv")
	expectedOutputPath := filepath.Join(testDataPath, "coingecko", "expected_THC_Trading_Average.csv")

	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "THC_Trading_Average.csv")

	if err := CalculateRollingAverages(inputPath, outputPath); err != nil {
		t.Fatalf("CalculateRollingAverages() error: %v", err)
	}

	compareCSVFiles(t, expectedOutputPath, outputPath)
}

// Edge case test: an empty file (only header) should result in an error.
func TestCalculateRollingAverages_EmptyFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "empty_test*.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write only a header (using CoinMarketCap format)
	header := "timeOpen;timeClose;timeHigh;timeLow;name;open;high;low;close;volume;marketCap;timestamp\n"
	if _, err := tmpFile.WriteString(header); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	outputPath := tmpFile.Name() + "_out.csv"
	err = CalculateRollingAverages(tmpFile.Name(), outputPath)
	if err == nil {
		t.Fatal("expected error for empty data file, got nil")
	}
}

// compareCSVFiles reads two CSV files and compares them record by record.
func compareCSVFiles(t *testing.T, expectedPath, actualPath string) {
	expectedFile, err := os.Open(expectedPath)
	if err != nil {
		t.Fatalf("error opening expected output file: %v", err)
	}
	defer expectedFile.Close()
	expectedRecords, err := readAllCSV(expectedPath)
	if err != nil {
		t.Fatalf("error reading expected output file: %v", err)
	}

	actualFile, err := os.Open(actualPath)
	if err != nil {
		t.Fatalf("error opening actual output file: %v", err)
	}
	defer actualFile.Close()
	actualRecords, err := readAllCSV(actualPath)
	if err != nil {
		t.Fatalf("error reading actual output file: %v", err)
	}

	if len(expectedRecords) != len(actualRecords) {
		t.Fatalf("expected %d records, got %d", len(expectedRecords), len(actualRecords))
	}

	// Compare each record field by field.
	for i := range expectedRecords {
		if len(expectedRecords[i]) != len(actualRecords[i]) {
			t.Errorf("record %d: expected %d fields, got %d", i, len(expectedRecords[i]), len(actualRecords[i]))
			continue
		}
		for j := range expectedRecords[i] {
			exp := strings.TrimSpace(expectedRecords[i][j])
			act := strings.TrimSpace(actualRecords[i][j])
			// For numeric fields, allow small differences.
			if numExp, err1 := strconv.ParseFloat(exp, 64); err1 == nil {
				if numAct, err2 := strconv.ParseFloat(act, 64); err2 == nil {
					if math.Abs(numExp-numAct) > 0.01 {
						t.Errorf("record %d, field %d: expected %v, got %v", i, j, numExp, numAct)
					}
					continue
				}
			}
			if exp != act {
				t.Errorf("record %d, field %d: expected %s, got %s", i, j, exp, act)
			}
		}
	}
}

// readAllCSV reads all records from a CSV file and filters out blank lines to ensure
// consistent comparison between files that may or may not have trailing newlines.
func readAllCSV(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	allRecords, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Filter out blank lines (lines where all fields are empty)
	var filteredRecords [][]string
	for _, record := range allRecords {
		hasContent := false
		for _, field := range record {
			if field != "" {
				hasContent = true
				break
			}
		}
		if hasContent {
			filteredRecords = append(filteredRecords, record)
		}
	}

	return filteredRecords, nil
}
