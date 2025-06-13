package gemini_ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/models"
)

type Client struct {
	config *config.GeminiConfig
	client *http.Client
}

func NewClient(cfg *config.GeminiConfig) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *Client) AnalyzeStock(symbol string, ohlcvData []models.OHLCVData, dataInfo models.DataInfo) (*models.IndividualAnalysisResponse, error) {
	prompt := c.buildIndividualAnalysisPrompt(symbol, ohlcvData, dataInfo)

	response, err := c.sendRequest(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get analysis from Gemini AI: %w", err)
	}

	jsonStr, err := extractJSONFromText(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from Gemini AI response: %w", err)
	}
	var analysis models.IndividualAnalysisResponse
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini AI response: %w", err)
	}

	// Set analysis date
	analysis.AnalysisDate = time.Now()

	return &analysis, nil
}

func (c *Client) MonitorPosition(request models.PositionMonitoringRequest, ohlcvData []models.OHLCVData, dataInfo models.DataInfo) (*models.PositionMonitoringResponse, error) {
	prompt := c.buildPositionMonitoringPrompt(request, ohlcvData, dataInfo)

	response, err := c.sendRequest(prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get position analysis from Gemini AI: %w", err)
	}

	jsonStr, err := extractJSONFromText(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from Gemini AI response: %w", err)
	}

	// Parse the response
	var analysis models.PositionMonitoringResponse
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini AI response: %w", err)
	}

	// Calculate position_age_days manually
	positionAgeDays := int(time.Since(request.BuyTime).Hours() / 24)
	analysis.PositionAgeDays = positionAgeDays

	return &analysis, nil
}

func (c *Client) sendRequest(prompt string) (string, error) {
	// Build request URL
	requestURL := fmt.Sprintf("%s/%s:generateContent?key=%s",
		c.config.BaseURL, c.config.Model, c.config.APIKey)

	// Build request body
	requestBody := models.GeminiAIRequest{
		Contents: []models.GeminiContent{
			{
				Parts: []models.GeminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Gemini AI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Gemini AI API returned status code %d: %s", resp.StatusCode, string(body))
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse response
	var geminiResp models.GeminiAIResponse
	if err := json.Unmarshal(body, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal Gemini AI response: %w", err)
	}

	// Check if we have candidates
	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("no response candidates from Gemini AI")
	}

	candidate := geminiResp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("no content parts in Gemini AI response")
	}

	return candidate.Content.Parts[0].Text, nil
}

func (c *Client) buildIndividualAnalysisPrompt(symbol string, ohlcvData []models.OHLCVData, dataInfo models.DataInfo) string {
	// Convert OHLCV data to JSON string
	ohlcvJSON, _ := json.Marshal(ohlcvData)

	prompt := fmt.Sprintf(`Anda adalah analis teknikal saham Indonesia yang ahli. Analisis data OHLC berikut untuk saham %s dan berikan rekomendasi trading swing (1-5 hari).

Data OHLC %s:
%s

Analisis teknikal yang diperlukan:
1. Trend: BULLISH/BEARISH/SIDEWAYS (short-term 1-3 hari, medium-term 1-2 minggu)
2. Technical indicators:
   - EMA 9/21 signal (BULLISH jika EMA9 > EMA21, BEARISH jika EMA9 < EMA21, NEUTRAL)
   - RSI 14 signal (OVERBOUGHT >70, OVERSOLD <30, NEUTRAL 30-70)
   - MACD signal (BULLISH jika MACD > Signal, BEARISH jika MACD < Signal, NEUTRAL)
   - Bollinger Bands position (UPPER/MIDDLE/LOWER)
3. Support dan resistance levels
4. Volume trend (HIGH/NORMAL/LOW) dan momentum
5. Candlestick pattern terbaru
6. Technical score (0-100)

KRITERIA PENTING:
- BUY signal hanya jika risk-reward ratio ≥ 1:3
- Target price harus realistis berdasarkan resistance levels
- Cut loss berdasarkan support levels yang kuat
- Max holding period 1-7 hari berdasarkan trend strength

Return response dalam format JSON:
{
  "symbol": "%s",
  "analysis_date": "%s",
  "signal": "BUY|HOLD|SELL",
  "max_holding_period_days": 5,
  "technical_analysis": {
    "trend": "BULLISH|BEARISH|SIDEWAYS",
    "short_term_trend": "BULLISH|BEARISH|SIDEWAYS",
    "medium_term_trend": "BULLISH|BEARISH|SIDEWAYS",
    "ema_9": 8750,
    "ema_21": 8700,
    "ema_signal": "BULLISH|BEARISH|NEUTRAL",
    "rsi": 65.5,
    "rsi_signal": "OVERBOUGHT|OVERSOLD|NEUTRAL",
    "macd_signal": "BULLISH|BEARISH|NEUTRAL",
    "bollinger_bands_position": "UPPER|MIDDLE|LOWER",
    "support_level": 8500,
    "resistance_level": 9200,
    "key_support_levels": [8500, 8400, 8300],
    "key_resistance_levels": [9200, 9300, 9400],
    "volume_trend": "HIGH|NORMAL|LOW",
    "volume_confirmation": "POSITIVE|NEGATIVE|NEUTRAL",
    "momentum": "STRONG|MODERATE|WEAK",
    "candlestick_pattern": "BULLISH|BEARISH|NEUTRAL",
    "market_structure": "UPTREND|DOWNTREND|SIDEWAYS",
    "trend_strength": "STRONG|MODERATE|WEAK",
    "breakout_potential": "HIGH|MEDIUM|LOW",
    "technical_score": 85
  },
  "recommendation": {
    "action": "BUY|HOLD|SELL",
    "buy_price": 8750,
    "target_price": 9200,
    "cut_loss": 8400,
    "confidence_level": 85,
    "reasoning": "Analisis teknikal menunjukkan momentum bullish dengan volume mendukung. EMA 9 di atas EMA 21, RSI 65.5 netral-positif, MACD bullish. Support 8500, resistance 9200. Risk/reward ratio menguntungkan.",
    "risk_reward_analysis": {
      "potential_profit": 450,
      "potential_profit_percentage": 5.14,
      "potential_loss": 350,
      "potential_loss_percentage": 4.0,
      "risk_reward_ratio": 1.29,
      "risk_level": "LOW|MEDIUM|HIGH",
      "expected_holding_period": "3-5 days",
      "success_probability": 75
    }
  },
  "risk_level": "LOW|MEDIUM|HIGH",
  "technical_summary": {
    "overall_signal": "BULLISH",
    "trend_strength": "STRONG",
    "volume_support": "HIGH",
    "momentum": "POSITIVE",
    "risk_level": "LOW",
    "confidence_level": 85,
    "key_insights": [
      "Trend bullish dengan volume mendukung",
      "Technical indicators positif",
      "Support dan resistance teridentifikasi",
      "Risk/reward ratio menguntungkan"
    ]
  }
}`, symbol, dataInfo.Range, string(ohlcvJSON), symbol, time.Now().Format("2006-01-02T15:04:05-07:00"))

	return prompt
}

func (c *Client) buildPositionMonitoringPrompt(request models.PositionMonitoringRequest, ohlcvData []models.OHLCVData, dataInfo models.DataInfo) string {
	// Convert OHLCV data to JSON string
	ohlcvJSON, _ := json.Marshal(ohlcvData)

	// Calculate remaining holding period
	positionAgeDays := int(time.Since(request.BuyTime).Hours() / 24)
	remainingDays := request.MaxHoldingPeriodDays - positionAgeDays
	if remainingDays < 0 {
		remainingDays = 0
	}

	prompt := fmt.Sprintf(`Anda adalah analis teknikal saham Indonesia yang ahli dalam swing trading. Analisis posisi trading yang sedang berjalan dan berikan rekomendasi HOLD/SELL/CUT_LOSS.

Data posisi trading:
- Symbol: %s
- Buy Price: %.2f
- Buy Time: %s
- Max Holding Period: %d days
- Position Age: %d days
- Remaining Days: %d days

Data OHLC %s:
%s

Analisis yang diperlukan:
1. Hitung current profit/loss dan percentage
2. Analisis trend (short-term dan medium-term)
3. Technical indicators: EMA 9/21, RSI 14, MACD, Bollinger Bands
4. Support dan resistance levels
5. Volume trend dan momentum
6. Candlestick patterns
7. Evaluasi apakah masih dalam trend yang diharapkan
8. Hitung remaining potential profit dan risk

KRITERIA PENTING:
- HOLD hanya jika risk-reward ratio ≥ 1:3 dan masih ada potential profit signifikan
- SELL jika trend berubah atau technical indicators memburuk
- CUT_LOSS jika risk meningkat atau target tidak realistis dalam sisa %d hari
- Evaluasi apakah target price masih realistis dalam sisa waktu

Return response dalam format JSON:
{
  "symbol": "%s",
  "current_price": 9100,
  "max_holding_period_days": %d,
  "recommendation": {
    "action": "HOLD|SELL|CUT_LOSS",
    "reasoning": "Analisis teknikal menunjukkan momentum bullish dengan volume mendukung. EMA 9 di atas EMA 21, RSI 68.5 netral-positif, MACD bullish. Support 8950, resistance 9200. Saham masih dalam trend yang diharapkan dengan potential profit menarik. Dengan sisa %d hari, masih ada waktu untuk mencapai target.",
    "technical_analysis": {
      "trend": "BULLISH",
      "short_term_trend": "BULLISH",
      "medium_term_trend": "BULLISH",
      "ema_9": 9080,
      "ema_21": 9020,
      "ema_signal": "BULLISH",
      "rsi": 68.5,
      "rsi_signal": "NEUTRAL",
      "macd_signal": "BULLISH",
      "bollinger_bands_position": "MIDDLE",
      "support_level": 8950,
      "resistance_level": 9200,
      "key_support_levels": [8950, 8900, 8850],
      "key_resistance_levels": [9200, 9250, 9300],
      "volume_trend": "HIGH",
      "volume_confirmation": "POSITIVE",
      "momentum": "STRONG",
      "candlestick_pattern": "BULLISH",
      "market_structure": "UPTREND",
      "trend_strength": "STRONG",
      "breakout_potential": "HIGH",
      "technical_score": 85
    },
    "risk_reward_analysis": {
      "current_profit": 100,
      "current_profit_percentage": 1.11,
      "remaining_potential_profit": 400,
      "remaining_potential_profit_percentage": 4.44,
      "current_risk": 150,
      "current_risk_percentage": 1.67,
      "risk_reward_ratio": 2.67,
      "risk_level": "LOW",
      "days_remaining": %d,
      "success_probability": 85,
      "exit_recommendation": {
        "target_exit_price": 9500,
        "stop_loss_price": 8950,
        "time_based_exit": "2025-01-16T09:00:00+07:00",
        "exit_reasoning": "Target exit di resistance 9500 dengan stop loss di support 8950. Time-based exit jika tidak mencapai target dalam %d hari tersisa.",
        "exit_conditions": [
          "Mencapai target price 9500",
          "Stop loss di 8950",
          "Time-based exit dalam %d hari tersisa",
          "Trend reversal signal"
        ]
      }
    }
  },
  "position_metrics": {
    "unrealized_pnl": 100,
    "unrealized_pnl_percentage": 1.11,
    "days_remaining": %d,
    "risk_assessment": "LOW",
    "position_health": "HEALTHY",
    "trend_alignment": "POSITIVE",
    "volume_support": "STRONG"
  },
  "technical_summary": {
    "overall_signal": "BULLISH",
    "trend_strength": "STRONG",
    "volume_support": "HIGH",
    "momentum": "POSITIVE",
    "risk_level": "LOW",
    "confidence_level": 85,
    "key_insights": [
      "Trend bullish dengan volume mendukung",
      "Technical indicators positif",
      "Support dan resistance teridentifikasi",
      "Risk/reward ratio menguntungkan"
    ]
  }
}`, request.Symbol, request.BuyPrice, request.BuyTime.Format("2006-01-02T15:04:05-07:00"),
		request.MaxHoldingPeriodDays, positionAgeDays, remainingDays, dataInfo.Range, string(ohlcvJSON),
		remainingDays, request.Symbol, request.MaxHoldingPeriodDays,
		remainingDays, remainingDays, remainingDays, remainingDays, remainingDays)

	return prompt
}

// extractJSONFromText extracts JSON object from text that may contain additional content
func extractJSONFromText(text string) (string, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || start > end {
		return "", fmt.Errorf("no JSON object found in response: %s", text)
	}
	return text[start : end+1], nil
}
