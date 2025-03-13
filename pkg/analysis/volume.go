// Package analysis provides functionality for analyzing token trading volume data.
// It supports both CoinGecko and CoinMarketCap data formats and calculates various
// metrics including rolling averages, low volume days, and historical highs.

package analysis

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// timeNow is a variable that can be overridden for testing
var timeNow = time.Now

// VolumeData represents a single day's trading data and calculated metrics.
// All monetary values are in USD.
type VolumeData struct {
	Name                 string    // Token identifier (e.g., "THC", "MAID")
	Date                 time.Time // Date of the trading data
	Volume               float64   // Daily trading volume in USD
	DayAvg30             float64   // 30-day rolling average volume
	DayAvg90             float64   // 90-day rolling average volume
	DayAvg180            float64   // 180-day rolling average volume
	LowVolumeDays30      int       // Number of days with volume <= $1 in last 30 days
	LowVolumeDays90      int       // Number of days with volume <= $1 in last 90 days
	LowVolumeDays180     int       // Number of days with volume <= $1 in last 180 days
	High30               float64   // Highest 30-day average volume seen
	High90               float64   // Highest 90-day average volume seen
	High180              float64   // Highest 180-day average volume seen
	ChangeFromHighAvg30  float64   // Percentage change from highest 30-day average
	ChangeFromHighAvg90  float64   // Percentage change from highest 90-day average
	ChangeFromHighAvg180 float64   // Percentage change from highest 180-day average
}

// DataSource represents the source of the trading data.
// Different sources have different CSV formats that need to be handled appropriately.
type DataSource int

const (
	CoinMarketCap DataSource = iota // Data from CoinMarketCap (semicolon-separated, RFC3339Nano timestamps)
	CoinGecko                       // Data from CoinGecko (comma-separated, custom timestamp format)
	Unknown                         // Unknown data source
)

// detectDataSource determines whether the data is from CoinMarketCap or CoinGecko
// based on the CSV header format. This allows automatic handling of different data sources.
// CRITICAL: This affects how timestamps and volumes are parsed. Do not modify without testing both formats.
func detectDataSource(reader *csv.Reader) (DataSource, error) {
	header, err := reader.Read()
	if err != nil {
		return CoinMarketCap, fmt.Errorf("error reading header: %v", err)
	}

	// CoinGecko format has exactly 4 columns with specific headers
	if len(header) == 4 && header[0] == "snapped_at" && header[3] == "total_volume" {
		return CoinGecko, nil
	}

	// Default to CoinMarketCap if not CoinGecko
	return CoinMarketCap, nil
}

// parseRecord parses a record based on the data source, extracting timestamp and volume.
// CRITICAL: Each source has different date formats and column positions:
// - CoinGecko: "YYYY-MM-DD HH:mm:ss UTC" format, volume in column 4
// - CoinMarketCap: RFC3339Nano format, volume in column 10
func parseRecord(record []string, source DataSource) (time.Time, float64, error) {
	var timestamp time.Time
	var volume float64
	var err error

	switch source {
	case CoinGecko:
		// Parse CoinGecko timestamp (YYYY-MM-DD HH:mm:ss UTC)
		timestamp, err = time.Parse("2006-01-02 15:04:05 MST", record[0])
		if err != nil {
			return time.Time{}, 0, fmt.Errorf("error parsing timestamp: %v", err)
		}
		volume, err = strconv.ParseFloat(record[3], 64) // total_volume is in column 4
		if err != nil {
			return time.Time{}, 0, fmt.Errorf("error parsing volume: %v", err)
		}

	case CoinMarketCap:
		// Parse CoinMarketCap timestamp (RFC3339Nano)
		timestamp, err = time.Parse(time.RFC3339Nano, strings.Trim(record[0], "\""))
		if err != nil {
			return time.Time{}, 0, fmt.Errorf("error parsing timestamp: %v", err)
		}
		volume, err = strconv.ParseFloat(strings.TrimSpace(record[9]), 64)
		if err != nil {
			return time.Time{}, 0, fmt.Errorf("error parsing volume: %v", err)
		}
	}

	return timestamp, volume, nil
}

// fillMissingDays ensures there is a record for every day in the date range.
// CRITICAL: This function:
// 1. Fills gaps in the data with zero volume
// 2. Extends the data up to today's date if the source data ends earlier
// 3. Preserves the original volume data for dates that exist
func fillMissingDays(records []VolumeData, name string) []VolumeData {
	if len(records) == 0 {
		return records
	}

	// Find the date range
	sort.Slice(records, func(i, j int) bool {
		return records[i].Date.Before(records[j].Date)
	})

	// Create a map of existing dates for quick lookup
	dateMap := make(map[string]float64)
	for _, record := range records {
		dateKey := record.Date.Format("2006-01-02")
		dateMap[dateKey] = record.Volume
	}

	// Create a complete list of records with all dates
	var completeRecords []VolumeData
	currentDate := records[0].Date
	today := timeNow().Truncate(24 * time.Hour)
	endDate := today // Include today in the output

	for !currentDate.After(endDate) {
		dateKey := currentDate.Format("2006-01-02")
		volume, exists := dateMap[dateKey]
		if !exists {
			volume = 0
		}

		completeRecords = append(completeRecords, VolumeData{
			Name:   name,
			Date:   currentDate,
			Volume: volume,
		})

		currentDate = currentDate.AddDate(0, 0, 1)
	}

	return completeRecords
}

// CalculateRollingAverages reads trading data from a CSV file and calculates rolling averages.
// CRITICAL: This function handles several important aspects:
// 1. Detects and handles different data source formats
// 2. Filters data to only include the last 365 days
// 3. Fills in missing days with zero volume
// 4. Calculates rolling averages and other metrics
// 5. Tracks historical highs and changes from those highs
func CalculateRollingAverages(inputFile, outputFile string) error {
	// Extract name from input file name (part before first underscore)
	baseName := filepath.Base(inputFile)
	name := strings.Split(baseName, "_")[0]

	// Read input file
	input, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("error opening input file: %v", err)
	}
	defer input.Close()

	// Create CSV reader
	reader := csv.NewReader(input)

	// Try to detect the format
	source, err := detectDataSource(reader)
	if err != nil {
		return err
	}

	// Set delimiter based on source
	if source == CoinMarketCap {
		reader.Comma = ';'
		reader.LazyQuotes = true
		reader.FieldsPerRecord = -1 // Allow variable number of fields
	}

	// Read all records and store in memory
	var records []VolumeData
	today := timeNow().UTC().Truncate(24 * time.Hour)
	fmt.Printf("Today is: %v\n", today)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading record: %v", err)
		}

		// Parse timestamp and volume based on source
		timestamp, volume, err := parseRecord(record, source)
		if err != nil {
			return err
		}

		fmt.Printf("Read record: date=%v volume=%v\n", timestamp, volume)

		// Skip future dates
		if timestamp.After(today) {
			fmt.Printf("Skipping future date: %v\n", timestamp)
			continue
		}

		records = append(records, VolumeData{
			Name:   name,
			Date:   timestamp.Truncate(24 * time.Hour),
			Volume: volume,
		})
	}

	if len(records) == 0 {
		return fmt.Errorf("no valid records found in input file")
	}

	fmt.Printf("Initial records: %d\n", len(records))
	for _, r := range records {
		fmt.Printf("  %v: %v\n", r.Date, r.Volume)
	}

	// Sort records by date (oldest first)
	sort.Slice(records, func(i, j int) bool {
		return records[i].Date.Before(records[j].Date)
	})

	// Keep only the last 365 days of data
	cutoffDate := today.AddDate(-1, 0, 0)
	var limitedRecords []VolumeData
	for _, record := range records {
		if !record.Date.Before(cutoffDate) {
			limitedRecords = append(limitedRecords, record)
		}
	}
	records = limitedRecords

	fmt.Printf("After limiting to 365 days: %d records\n", len(records))
	for _, r := range records {
		fmt.Printf("  %v: %v\n", r.Date, r.Volume)
	}

	// Fill in any missing days with zero volume
	records = fillMissingDays(records, name)

	fmt.Printf("After filling missing days: %d records\n", len(records))
	for _, r := range records {
		fmt.Printf("  %v: %v\n", r.Date, r.Volume)
	}

	// Track highest averages seen
	var highestAvg30, highestAvg90, highestAvg180 float64

	// Calculate rolling averages, low volume days, and track highest averages
	for i := 0; i < len(records); i++ {
		// Skip future dates in calculations
		if records[i].Date.After(today) {
			continue
		}

		// 30-day window
		sum := 0.0
		lowVolumeDays := 0
		daysInWindow := 0

		// Count backwards from current day
		for j := 0; j < 30 && i-j >= 0; j++ {
			vol := records[i-j].Volume
			sum += vol
			if vol <= 1.0 {
				lowVolumeDays++
			}
			daysInWindow++
		}

		// Calculate average using actual number of days in window
		avg := sum / float64(daysInWindow)
		records[i].DayAvg30 = avg
		records[i].LowVolumeDays30 = lowVolumeDays

		// Update highest average if needed
		if avg > highestAvg30 {
			highestAvg30 = avg
		}
		records[i].High30 = highestAvg30

		// Calculate change from high
		if highestAvg30 > 0 {
			records[i].ChangeFromHighAvg30 = ((avg - highestAvg30) / highestAvg30) * 100
		}

		// 90-day window
		if i >= 89 {
			sum := 0.0
			lowVolumeDays := 0
			daysInWindow := 0
			for j := 0; j < 90 && i-j >= 0; j++ {
				vol := records[i-j].Volume
				sum += vol
				if vol <= 1.0 {
					lowVolumeDays++
				}
				daysInWindow++
			}
			avg := sum / float64(daysInWindow)
			records[i].DayAvg90 = avg
			records[i].LowVolumeDays90 = lowVolumeDays
			if avg > highestAvg90 {
				highestAvg90 = avg
			}
			if highestAvg90 > 0 {
				records[i].ChangeFromHighAvg90 = ((avg - highestAvg90) / highestAvg90) * 100
			}
			records[i].High90 = highestAvg90
		}

		// 180-day window
		if i >= 179 {
			sum := 0.0
			lowVolumeDays := 0
			daysInWindow := 0
			for j := 0; j < 180 && i-j >= 0; j++ {
				vol := records[i-j].Volume
				sum += vol
				if vol <= 1.0 {
					lowVolumeDays++
				}
				daysInWindow++
			}
			avg := sum / float64(daysInWindow)
			records[i].DayAvg180 = avg
			records[i].LowVolumeDays180 = lowVolumeDays
			if avg > highestAvg180 {
				highestAvg180 = avg
			}
			if highestAvg180 > 0 {
				records[i].ChangeFromHighAvg180 = ((avg - highestAvg180) / highestAvg180) * 100
			}
			records[i].High180 = highestAvg180
		}
	}

	// Create output file
	output, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer output.Close()

	// Create CSV writer
	writer := csv.NewWriter(output)
	writer.Comma = ',' // Use comma as separator for output

	// Write header
	header := []string{
		"Date",
		"Volume",
		"30DayAvg",
		"90DayAvg",
		"180DayAvg",
		"LowVolumeDays30",
		"LowVolumeDays90",
		"LowVolumeDays180",
		"High30",
		"High90",
		"High180",
		"ChangeFromHighAvg30",
		"ChangeFromHighAvg90",
		"ChangeFromHighAvg180",
	}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error writing header: %v", err)
	}

	// Write records in reverse order (newest first)
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		// Skip future dates in output
		if record.Date.After(today) {
			continue
		}
		// For today's date, use the actual record
		if record.Date.Equal(today) {
			record = records[i]
		}
		row := []string{
			record.Date.Format("2006-01-02"),
			fmt.Sprintf("%.2f", record.Volume),
			fmt.Sprintf("%.2f", record.DayAvg30),
			fmt.Sprintf("%.2f", record.DayAvg90),
			fmt.Sprintf("%.2f", record.DayAvg180),
			fmt.Sprintf("%d", record.LowVolumeDays30),
			fmt.Sprintf("%d", record.LowVolumeDays90),
			fmt.Sprintf("%d", record.LowVolumeDays180),
			fmt.Sprintf("%.2f", record.High30),
			fmt.Sprintf("%.2f", record.High90),
			fmt.Sprintf("%.2f", record.High180),
			fmt.Sprintf("%.2f", record.ChangeFromHighAvg30),
			fmt.Sprintf("%.2f", record.ChangeFromHighAvg90),
			fmt.Sprintf("%.2f", record.ChangeFromHighAvg180),
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("error writing record: %v", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return fmt.Errorf("error flushing writer: %v", err)
	}

	return nil
}

// ProcessAllFiles processes all CSV files in the downloads directory and generates analysis files.
// For each CSV file:
// 1. Extracts the token name from the filename
// 2. Creates a corresponding output file in the output directory
// 3. Processes the data using CalculateRollingAverages
// 4. Continues processing remaining files even if one fails
//
// The function expects CSV files to follow the naming convention:
// - CoinMarketCap: TOKEN_DATE_RANGE_historical_data_coinmarketcap.csv
// - CoinGecko: TOKEN_usd-max.csv
func ProcessAllFiles(downloadsDir, outputDir string) error {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	// Read all files in downloads directory
	entries, err := os.ReadDir(downloadsDir)
	if err != nil {
		return fmt.Errorf("error reading downloads directory: %v", err)
	}

	// Process each CSV file
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".csv") {
			inputFile := filepath.Join(downloadsDir, entry.Name())

			// Extract name from file and create output filename
			name := strings.Split(entry.Name(), "_")[0]
			outputFile := filepath.Join(outputDir, fmt.Sprintf("%s_Trading_Average.csv", name))

			// Process the file
			fmt.Printf("Processing %s...\n", entry.Name())
			if err := CalculateRollingAverages(inputFile, outputFile); err != nil {
				fmt.Printf("Error processing %s: %v\n", entry.Name(), err)
				continue // Continue with next file even if this one fails
			}
			fmt.Printf("Successfully processed %s\n", entry.Name())
		}
	}

	return nil
}
