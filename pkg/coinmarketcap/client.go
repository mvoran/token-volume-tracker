package coinmarketcap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"token-volume-tracker/pkg/config"
)

const (
	baseURL = "https://pro-api.coinmarketcap.com/v1"
)

// Client represents a CoinMarketCap API client
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// VolumeData represents the trading volume data for a specific day
type VolumeData struct {
	Date   time.Time
	Volume float64
}

// NewClient creates a new CoinMarketCap API client
func NewClient(cfg *config.Config) (*Client, error) {
	if cfg.CMCApiKey == "" {
		return nil, fmt.Errorf("CoinMarketCap API key not set in config")
	}

	return &Client{
		apiKey: cfg.CMCApiKey,
		httpClient: &http.Client{
			Timeout: time.Second * 10,
		},
	}, nil
}

// GetHistoricalVolume fetches historical volume data for a given token
func (c *Client) GetHistoricalVolume(symbol string, days int) ([]VolumeData, error) {
	// Calculate the start and end dates
	end := time.Now()
	start := end.AddDate(0, 0, -days)

	// Build the API URL
	url := fmt.Sprintf("%s/cryptocurrency/quotes/historical", baseURL)

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	req.Header.Set("X-CMC_PRO_API_KEY", c.apiKey)
	req.Header.Set("Accept", "application/json")

	// Add query parameters
	q := req.URL.Query()
	q.Add("symbol", symbol)
	q.Add("time_start", start.Format(time.RFC3339))
	q.Add("time_end", end.Format(time.RFC3339))
	q.Add("interval", "1d")
	req.URL.RawQuery = q.Encode()

	// Make the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %s", resp.Status)
	}

	// Parse response
	var result struct {
		Data []struct {
			Timestamp time.Time `json:"timestamp"`
			Quote     struct {
				USD struct {
					Volume24h float64 `json:"volume_24h"`
				} `json:"USD"`
			} `json:"quote"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	// Convert to VolumeData slice
	volumeData := make([]VolumeData, len(result.Data))
	for i, d := range result.Data {
		volumeData[i] = VolumeData{
			Date:   d.Timestamp,
			Volume: d.Quote.USD.Volume24h,
		}
	}

	return volumeData, nil
}
