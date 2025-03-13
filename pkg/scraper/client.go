package scraper

import (
	"context"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// Client represents a scraper client
type Client struct{}

// VolumeData represents the trading volume data for a specific day
type VolumeData struct {
	Date   time.Time
	Volume float64
}

// NewClient creates a new scraper client
func NewClient() *Client {
	return &Client{}
}

// GetHistoricalVolume fetches historical volume data for a given token
// If endDate is provided, it will fetch data up to that date; otherwise, it uses the current date
func (c *Client) GetHistoricalVolume(token string, days int, endDate ...time.Time) ([]VolumeData, error) {
	// Determine end date
	var end time.Time
	if len(endDate) > 0 {
		end = endDate[0]
	} else {
		end = time.Now()
	}

	// Format dates for URL
	endDateStr := end.Format("20060102")
	startDate := end.AddDate(0, 0, -days)
	startDateStr := startDate.Format("20060102")

	// Try to map token symbols to their CoinMarketCap slug
	tokenToSlug := map[string]string{
		"MAID": "maidsafecoin",
		"BTC":  "bitcoin",
		"ETH":  "ethereum",
		// Add more mappings as needed
	}

	// Get the correct slug for the URL
	slug, ok := tokenToSlug[strings.ToUpper(token)]
	if !ok {
		// If not found in mapping, use lowercase token as fallback
		slug = strings.ToLower(token)
		fmt.Printf("No known slug mapping for %s, using %s as fallback\n", token, slug)
	}

	// Check if the CSV file already exists in the downloads directory
	downloadDir := "downloads"
	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		if err := os.MkdirAll(downloadDir, 0755); err != nil {
			return nil, fmt.Errorf("error creating downloads directory: %w", err)
		}
	}

	// Try downloading directly from CoinMarketCap first
	fmt.Println("Attempting to use manual download approach (recommended)...")
	volumeData, err := c.manualDownloadApproach(token, slug, startDateStr, endDateStr)
	if err == nil && len(volumeData) > 0 {
		fmt.Printf("Successfully retrieved %d records via manual download approach\n", len(volumeData))
		return volumeData, nil
	}
	fmt.Printf("Manual download approach failed: %v\n", err)

	// Fallback to scraping if manual download fails
	manualURL := fmt.Sprintf("https://coinmarketcap.com/currencies/%s/historical-data/", slug)
	fmt.Printf("Could not scrape data from CoinMarketCap. This is likely due to anti-scraping measures.\n")
	fmt.Printf("Please manually download the data from: %s\n", manualURL)
	fmt.Printf("Select the date range from %s to %s, and then click 'Download CSV'\n",
		startDate.Format("2006-01-02"), end.Format("2006-01-02"))

	// Return the error so the user knows manual intervention is required
	return nil, fmt.Errorf("automatic data retrieval failed, please download manually from %s", manualURL)
}

// manualDownloadApproach attempts to download the CSV directly from CoinMarketCap
func (c *Client) manualDownloadApproach(token, slug, startDateStr, endDateStr string) ([]VolumeData, error) {
	// Create a temporary directory for downloads
	tempDir, err := ioutil.TempDir("", "cmc-downloads")
	if err != nil {
		return nil, fmt.Errorf("error creating temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Setup chromedp with more options to avoid detection
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false), // Change to false for debugging - you can see what's happening
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("disable-web-security", true),
		chromedp.Flag("disable-extensions", true),
		// Set download directory
		chromedp.Flag("download.default_directory", tempDir),
		// Fake a regular user agent
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"),
	)

	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancel()

	// Create browser context with verbose logging
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
		chromedp.WithDebugf(log.Printf),
	)
	defer cancel()

	// Set a longer timeout (2 minutes)
	ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	// Enable network event handling
	if err := chromedp.Run(ctx, network.Enable()); err != nil {
		return nil, fmt.Errorf("error enabling network: %w", err)
	}

	// Navigate to CoinMarketCap's historical data page
	historyURL := fmt.Sprintf("https://coinmarketcap.com/currencies/%s/historical-data/", slug)
	fmt.Printf("Navigating to %s\n", historyURL)

	// The process is:
	// 1. Navigate to the historical data page
	// 2. Wait for the date picker to be available
	// 3. Click on the date picker
	// 4. Set the date range
	// 5. Click apply
	// 6. Wait for the data to load
	// 7. Click the download button

	fmt.Println("Opening browser to download data (this might take a moment)...")
	err = chromedp.Run(ctx,
		// Navigate to the page
		chromedp.Navigate(historyURL),

		// Set desktop viewport
		emulation.SetDeviceMetricsOverride(1920, 1080, 1.0, false),

		// Wait for page to load
		chromedp.Sleep(5*time.Second),

		// Take screenshot for debugging
		chromedp.CaptureScreenshot(&[]byte{}),

		// Handle cookie consent if present
		chromedp.ActionFunc(func(ctx context.Context) error {
			// Try to click various cookie consent buttons
			cookieSelectors := []string{
				".cmc-cookie-policy-banner__close",
				"#onetrust-accept-btn-handler",
				"button[aria-label='Accept all cookies']",
			}

			for _, selector := range cookieSelectors {
				// Check if the element exists before trying to click it
				var nodes []*cdp.Node
				if err := chromedp.Nodes(selector, &nodes).Do(ctx); err == nil && len(nodes) > 0 {
					return chromedp.Click(selector).Do(ctx)
				}
			}
			return nil
		}),

		// Wait for page to adjust after cookie banner
		chromedp.Sleep(2*time.Second),

		// Take another screenshot
		chromedp.CaptureScreenshot(&[]byte{}),
	)

	if err != nil {
		return nil, fmt.Errorf("error in initial page navigation: %w", err)
	}

	// At this point, we should be at the historical data page
	// For CoinMarketCap, manual download is the most reliable method
	fmt.Println("Please manually download historical data from the opened browser window.")
	fmt.Println("1. Set the date range")
	fmt.Println("2. Click 'Download CSV'")
	fmt.Println("3. Once download is complete, you can close the browser")

	// Wait for user to manually download data
	fmt.Println("Press Enter after manually downloading the CSV file...")

	// Instead of waiting for user input, we'll wait for a fixed time
	fmt.Println("Waiting 30 seconds for manual download...")
	time.Sleep(30 * time.Second)

	// Manual download process completed
	fmt.Println("Manual download process complete.")

	// Fallback to instructions if download wasn't successful
	return nil, fmt.Errorf("automated download not possible, manual download required")
}

// loadCsvFile loads volume data from a CSV file
func loadCsvFile(filePath string) ([]VolumeData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV file: %w", err)
	}

	if len(records) <= 1 {
		return nil, fmt.Errorf("no data records found in CSV file")
	}

	var volumeData []VolumeData
	// Skip header row
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 2 {
			continue
		}

		// The format of the CSV can vary, but typically:
		// - 0th column contains the date
		// - Some column contains the volume

		// Try to find the volume column - it might be titled "Volume" or "Volume(USD)" etc.
		volumeColIndex := -1
		dateColIndex := 0

		// First check the header row to find volume
		if i == 1 {
			for j, col := range records[0] {
				colLower := strings.ToLower(col)
				if strings.Contains(colLower, "volume") {
					volumeColIndex = j
					break
				}
			}

			// If volume column not found, assume it's the 5th or 6th column
			if volumeColIndex == -1 && len(record) >= 6 {
				volumeColIndex = 5
			}
		}

		// Get date and volume from appropriate columns
		dateStr := record[dateColIndex]
		volumeStr := record[volumeColIndex]

		// Parse date
		date, err := parseDate(dateStr)
		if err != nil {
			fmt.Printf("Warning: Could not parse date '%s': %v\n", dateStr, err)
			continue
		}

		// Parse volume
		volume, err := parseVolume(volumeStr)
		if err != nil {
			fmt.Printf("Warning: Could not parse volume '%s': %v\n", volumeStr, err)
			continue
		}

		volumeData = append(volumeData, VolumeData{
			Date:   date,
			Volume: volume,
		})
	}

	return volumeData, nil
}

// stripTags removes HTML tags from a string
func stripTags(html string) string {
	// Remove tags
	tagRegex := regexp.MustCompile(`<[^>]*>`)
	text := tagRegex.ReplaceAllString(html, "")
	// Trim spaces
	return strings.TrimSpace(text)
}

// parseDate attempts to convert date strings from CoinMarketCap to time.Time
func parseDate(dateStr string) (time.Time, error) {
	// CoinMarketCap uses formats like "Apr 01, 2023" or "2023-04-01"
	formats := []string{
		"Jan 02, 2006",
		"Jan 2, 2006",
		"2006-01-02",
		"Jan 02 2006",
		"Jan 2 2006",
		"01/02/2006",
		"1/2/2006",
		"1/2/06",
		"2006/01/02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("could not parse date: %s", dateStr)
}

// parseVolume attempts to convert volume strings to float64
func parseVolume(volumeStr string) (float64, error) {
	// If the string is empty or just contains dashes/other non-numeric indicators, return 0
	if volumeStr == "" || volumeStr == "--" || volumeStr == "-" || volumeStr == "n/a" {
		return 0, nil
	}

	// Remove currency symbols, commas, etc.
	re := regexp.MustCompile(`[^\d.]`)
	numStr := re.ReplaceAllString(volumeStr, "")

	// If after cleaning we have an empty string, return 0
	if numStr == "" {
		return 0, nil
	}

	// Parse as float
	return strconv.ParseFloat(numStr, 64)
}
