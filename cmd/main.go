package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"token-volume-tracker/pkg/analysis"
	"token-volume-tracker/pkg/scraper"
)

// Base directory for all data
const dataBasePath = "/Users/mikev/Library/Mobile Documents/com~apple~CloudDocs/Personal/Software Development/Token Volume Tracker Data"

func main() {
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

	switch os.Args[1] {
	case "fetch":
		fetchCmd.Parse(os.Args[2:])
		if *tokenID == "" {
			fmt.Println("Error: token symbol is required")
			fetchCmd.PrintDefaults()
			os.Exit(1)
		}
		handleFetch(*tokenID, *days)

	case "analyze":
		analyzeCmd.Parse(os.Args[2:])
		handleAnalyze(*inputFile)

	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		fmt.Println("Expected 'fetch' or 'analyze' subcommand")
		os.Exit(1)
	}
}

func handleFetch(token string, days int) {
	// Create download directory if it doesn't exist
	downloadDir := filepath.Join(dataBasePath, "Download")
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		fmt.Printf("Error creating download directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize scraper client
	client := scraper.NewClient()

	// Fetch historical volume data
	fmt.Printf("Fetching %d days of historical volume data for %s...\n", days, token)
	volumeData, err := client.GetHistoricalVolume(token, days)
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

func handleAnalyze(inputFile string) {
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
		outputFile := filepath.Join(finalDir, fmt.Sprintf("%s_Trading_Average.csv", name))

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

	// TODO: Implement CSV writing
	return nil
}
