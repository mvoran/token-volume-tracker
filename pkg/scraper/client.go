package scraper

import (
	"fmt"
	"time"
)

// Client represents a scraper client
type Client struct {
}

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
func (c *Client) GetHistoricalVolume(token string, days int) ([]VolumeData, error) {
	return nil, fmt.Errorf("not implemented")
}
