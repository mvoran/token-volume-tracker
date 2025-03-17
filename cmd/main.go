package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"token-volume-tracker/pkg/analysis"
	"token-volume-tracker/pkg/scraper"
	"token-volume-tracker/pkg/utils"
)

// Base directory for all data relative to project root
const dataDirName = "Token Volume Tracker Data"

func main() {
	// Get project root directory
	root, err := utils.GetProjectRoot()
	if err != nil {
		fmt.Printf("Error getting project root: %v\n", err)
		os.Exit(1)
	}

	// Define command line flags
	fetchCmd := flag.NewFlagSet("fetch", flag.ExitOnError)
	analyzeCmd := flag.NewFlagSet("analyze", flag.ExitOnError)

	// Fetch command flags
	tokenID := fetchCmd.String("token", "", "Token symbol (e.g., 'CELO')")
	days := fetchCmd.Int("days", 7, "Number of days of historical data to fetch")

	// Analyze command flags
	inputFile := analyzeCmd.String("input", "", "Input CSV file to analyze (if empty, processes all files in Download directory)")

	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Expected 'fetch' or 'analyze' subcommand")
		os.Exit(1)
	}

	// Get the data base path
	dataBasePath := filepath.Join(filepath.Dir(root), dataDirName)

	switch os.Args[1] {
	case "fetch":
		fetchCmd.Parse(os.Args[2:])
		if *tokenID == "" {
			fmt.Println("Error: token symbol is required")
			fetchCmd.PrintDefaults()
			os.Exit(1)
		}
		handleFetch(*tokenID, *days, dataBasePath)

	case "analyze":
		analyzeCmd.Parse(os.Args[2:])
		handleAnalyze(*inputFile, dataBasePath)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Expected 'fetch' or 'analyze' subcommand")
		os.Exit(1)
	}
}

func handleFetch(token string, days int, dataBasePath string) {
	// Create download directory if it doesn't exist
	downloadDir := filepath.Join(dataBasePath, "Download")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		fmt.Printf("Error creating download directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize scraper client
	client := scraper.NewClient()

	// If requesting a full year, adjust days to 364 due to source data limitation
	if days >= 365 {
		days = 364
	}

	// Ensure we only request data up to yesterday (last full day)
	endDate := time.Now().AddDate(0, 0, -1) // Yesterday

	fmt.Printf("Fetching %d days of historical volume data for %s (up to %s)...\n",
		days, token, endDate.Format("2006-01-02"))

	volumeData, err := client.GetHistoricalVolume(token, days, endDate)
	if err != nil {
		fmt.Printf("Error fetching data: %v\n", err)
		os.Exit(1)
	}

	// Create output file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	outputFile := filepath.Join(downloadDir, fmt.Sprintf("%s_volume_%s.csv", token, timestamp))
	if err := writeCSV(outputFile, volumeData); err != nil {
		fmt.Printf("Error writing data: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully wrote data to %s\n", outputFile)
}

func handleAnalyze(inputFile string, dataBasePath string) {
	downloadDir := filepath.Join(dataBasePath, "Download")
	finalDir := filepath.Join(dataBasePath, "Final")

	// Ensure the Final directory exists
	if err := os.MkdirAll(finalDir, 0755); err != nil {
		fmt.Printf("Error creating final directory: %v\n", err)
		os.Exit(1)
	}

	if inputFile != "" {
		// Process single file
		name := strings.Split(filepath.Base(inputFile), "_")[0]
		outputFile := filepath.Join(finalDir, fmt.Sprintf("%s_Token_Analysis.csv", name))

		fmt.Printf("Calculating rolling averages for %s...\n", inputFile)
		if err := analysis.CalculateRollingAverages(inputFile, outputFile); err != nil {
			fmt.Printf("Error calculating averages: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully wrote analysis to %s\n", outputFile)
	} else {
		// Process all files in Download directory
		fmt.Println("Processing all files in Download directory...")
		if err := analysis.ProcessAllFiles(downloadDir, finalDir); err != nil {
			fmt.Printf("Error processing files: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully processed all files")
	}
}

func writeCSV(filename string, data []scraper.VolumeData) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating output file: %v", err)
	}
	defer file.Close()

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{"Date", "Volume"}
	if err := writer.Write(header); err != nil {
		return fmt.Errorf("error writing header: %v", err)
	}

	// Write data
	for _, d := range data {
		record := []string{
			d.Date.Format("2006-01-02"),
			fmt.Sprintf("%.2f", d.Volume),
		}
		if err := writer.Write(record); err != nil {
			return fmt.Errorf("error writing record: %v", err)
		}
	}

	return nil
}
