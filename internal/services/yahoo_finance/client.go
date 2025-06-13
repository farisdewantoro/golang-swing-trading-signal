package yahoo_finance

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/models"
)

type Client struct {
	config *config.YahooFinanceConfig
	client *http.Client
}

func NewClient(cfg *config.YahooFinanceConfig) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// OHLCDataWithInfo contains OHLC data and metadata about the data
type OHLCDataWithInfo struct {
	Data     []models.OHLCVData
	DataInfo models.DataInfo
}

func (c *Client) GetOHLCData(symbol string, period1, period2 int64) (*OHLCDataWithInfo, error) {
	// Add .JK suffix for Indonesian stocks
	indonesianSymbol := symbol + ".JK"

	// Build URL with query parameters
	baseURL := c.config.BaseURL + "/" + indonesianSymbol
	params := url.Values{}
	params.Add("period1", fmt.Sprintf("%d", period1))
	params.Add("period2", fmt.Sprintf("%d", period2))
	params.Add("interval", "1d")
	params.Add("includePrePost", "false")
	params.Add("events", "div,split")

	requestURL := baseURL + "?" + params.Encode()

	// Create HTTP request
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers to mimic browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Referer", "https://finance.yahoo.com/")

	// Make HTTP request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data from Yahoo Finance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Yahoo Finance API returned status: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle gzip compression
	if resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(io.NopCloser(io.NewSectionReader(bytes.NewReader(body), 0, int64(len(body)))))
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()

		body, err = io.ReadAll(reader)
		if err != nil {
			return nil, fmt.Errorf("failed to decompress gzip response: %w", err)
		}
	}

	// Parse JSON response
	var yahooResp models.YahooFinanceResponse
	if err := json.Unmarshal(body, &yahooResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Yahoo Finance response: %w", err)
	}

	// Check for API errors
	if yahooResp.Chart.Error != nil {
		return nil, fmt.Errorf("Yahoo Finance API error: %v", yahooResp.Chart.Error)
	}

	// Check if we have results
	if len(yahooResp.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data returned for symbol: %s", symbol)
	}

	result := yahooResp.Chart.Result[0]
	if len(result.Indicators.Quote) == 0 {
		return nil, fmt.Errorf("no quote data available for symbol: %s", symbol)
	}

	quote := result.Indicators.Quote[0]

	// Convert to OHLCVData format
	var ohlcvData []models.OHLCVData
	for i, timestamp := range result.Timestamp {
		// Skip if any required data is missing
		if i >= len(quote.Open) || i >= len(quote.High) || i >= len(quote.Low) ||
			i >= len(quote.Close) || i >= len(quote.Volume) {
			continue
		}

		// Skip if any value is 0 (missing data)
		if quote.Open[i] == 0 || quote.High[i] == 0 || quote.Low[i] == 0 ||
			quote.Close[i] == 0 {
			continue
		}

		ohlcvData = append(ohlcvData, models.OHLCVData{
			Timestamp: timestamp,
			Open:      quote.Open[i],
			High:      quote.High[i],
			Low:       quote.Low[i],
			Close:     quote.Close[i],
			Volume:    quote.Volume[i],
		})
	}

	if len(ohlcvData) == 0 {
		return nil, fmt.Errorf("no valid OHLCV data found for symbol: %s", symbol)
	}

	// Create data info
	startDate := time.Unix(period1, 0)
	endDate := time.Unix(period2, 0)

	dataInfo := models.DataInfo{
		Interval:   "1d",
		Range:      fmt.Sprintf("%d days", int(endDate.Sub(startDate).Hours()/24)),
		StartDate:  startDate,
		EndDate:    endDate,
		DataPoints: len(ohlcvData),
		Source:     "Yahoo Finance API",
	}

	return &OHLCDataWithInfo{
		Data:     ohlcvData,
		DataInfo: dataInfo,
	}, nil
}

// GetRecentOHLCData gets the last 60 days of OHLC data
func (c *Client) GetRecentOHLCData(symbol string) (*OHLCDataWithInfo, error) {
	wib, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return nil, fmt.Errorf("failed to load WIB timezone, using UTC: %v", err)

	}
	now := time.Now().In(wib)
	period2 := now.Unix()
	period1 := now.AddDate(0, 0, -60).Unix() // 60 days ago

	return c.GetOHLCData(symbol, period1, period2)
}

// GetLatestOHLCData gets the most recent OHLC data
func (c *Client) GetLatestOHLCData(symbol string) (*models.OHLCVData, error) {
	ohlcvDataWithInfo, err := c.GetRecentOHLCData(symbol)
	if err != nil {
		return nil, err
	}

	if len(ohlcvDataWithInfo.Data) == 0 {
		return nil, fmt.Errorf("no OHLCV data found for symbol: %s", symbol)
	}

	// Return the most recent data (last element)
	latest := ohlcvDataWithInfo.Data[len(ohlcvDataWithInfo.Data)-1]
	return &latest, nil
}
