package gemini_ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"

	"golang-swing-trading-signal/pkg/ratelimit"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
	"google.golang.org/genai"
)

type Client struct {
	config         *config.GeminiConfig
	client         *http.Client
	requestLimiter *rate.Limiter
	tokenLimiter   *ratelimit.TokenLimiter
	logger         *logrus.Logger
	geminiClient   *genai.Client
}

func NewClient(cfg *config.GeminiConfig, logger *logrus.Logger, geminiClient *genai.Client) *Client {
	secondsPerRequest := time.Minute / time.Duration(cfg.MaxRequestPerMinute)
	requestLimiter := rate.NewLimiter(rate.Every(secondsPerRequest), 1)

	tokenLimiter := ratelimit.NewTokenLimiter(cfg.MaxTokenPerMinute)

	return &Client{
		config: cfg,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		requestLimiter: requestLimiter,
		tokenLimiter:   tokenLimiter,
		logger:         logger,
		geminiClient:   geminiClient,
	}
}

func (c *Client) sendRequest(ctx context.Context, prompt string) (string, error) {
	contents := []*genai.Content{
		genai.NewContentFromText(prompt, "user"),
	}

	tokenCount, err := c.geminiClient.Models.CountTokens(ctx, c.config.Model, contents, nil)

	if err != nil {
		c.logger.Error("failed to count tokens", logrus.Fields{
			"error": err,
		})
		return "", fmt.Errorf("failed to count tokens: %w", err)
	}

	if reserve := c.requestLimiter.Reserve(); reserve.Delay() > 0 {
		c.logger.Warn("request limit exceeded", logrus.Fields{
			"remaining_tokens":   c.tokenLimiter.GetRemaining(),
			"token_available_at": utils.TimeNowWIB().Add(reserve.Delay()),
		})
	}

	if err := c.tokenLimiter.Wait(ctx, int(tokenCount.TotalTokens)); err != nil {
		return "", fmt.Errorf("failed to wait for token limit: %w", err)
	}
	if err := c.requestLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("failed to wait for request limit: %w", err)
	}

	if int(tokenCount.TotalTokens) > c.config.MaxTokenPerMinute/2 {
		c.logger.Warn("gemini ai token exceeded half limit", logrus.Fields{
			"remaining_tokens": tokenCount.TotalTokens,
			"max_tokens":       c.config.MaxTokenPerMinute,
		})
	}

	// Build request URL
	requestURL := fmt.Sprintf("%s/%s:generateContent?key=%s",
		c.config.BaseURL, c.config.Model, c.config.APIKey)

	// Build request body
	requestBody := models.GeminiAIRequest{
		GenerationConfig: &models.GeminiGenerationConfig{
			Temperature: c.config.RequestTemperature,
		},
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

	c.logger.Debug("Sending request to Gemini AI", logrus.Fields{
		"request_body": requestBody,
		"request_url":  requestURL,
	})

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewBuffer(jsonBody))
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
		return "", fmt.Errorf("gemini AI API returned status code %d: %s", resp.StatusCode, string(body))
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

	c.logger.Debug("Gemini AI response", logrus.Fields{
		"candidates": geminiResp.Candidates,
	})

	// Check if we have candidates
	if len(geminiResp.Candidates) == 0 {
		return "", fmt.Errorf("no response candidates from Gemini AI")
	}

	candidate := geminiResp.Candidates[0]
	if len(candidate.Content.Parts) == 0 {
		return "", fmt.Errorf("no content parts in Gemini AI response")
	}

	if strings.ToLower(candidate.FinishReason) != "stop" {
		return "", fmt.Errorf("gemini AI response did not finish: %s", candidate.FinishReason)
	}

	return candidate.Content.Parts[0].Text, nil
}

func (c *Client) buildIndividualAnalysisPrompt(
	ctx context.Context,
	symbol string,
	ohlcvData []models.OHLCVData,
	dataInfo models.DataInfo,
	summary *models.StockNewsSummaryEntity,
) string {
	// Convert OHLCV data to JSON string
	ohlcvJSON, _ := json.Marshal(ohlcvData)

	// Ringkasan sentimen dari berita
	newsSummaryText := ""
	if summary != nil {
		newsSummaryText = fmt.Sprintf(`
### INPUT BERITA TERKINI
Berikut adalah ringkasan berita untuk saham %s selama periode %s hingga %s:
- Sentimen utama: %s
- Dampak terhadap harga: %s
- Key issues: %s
- Ringkasan singkat: %s
- Confidence score: %.2f
- Saran tindakan: %s
- Alasan: %s

**Gunakan informasi ini sebagai konteks eksternal saat menganalisis data teknikal.**
`,
			summary.StockCode,
			summary.SummaryStart.Format("2006-01-02"),
			summary.SummaryEnd.Format("2006-01-02"),
			summary.SummarySentiment,
			summary.SummaryImpact,
			strings.Join(summary.KeyIssues, ", "),
			summary.ShortSummary,
			summary.SummaryConfidenceScore,
			summary.SuggestedAction,
			summary.Reasoning,
		)
	}

	prompt := fmt.Sprintf(`
### PERAN ANDA
Anda adalah analis teknikal profesional dengan pengalaman lebih dari 10 tahun di pasar saham Indonesia. Tugas Anda adalah melakukan analisis teknikal dan memberikan sinyal trading **swing jangka pendek (1-5 hari)** berdasarkan data harga (OHLC) dan berita pasar untuk saham %s.

### TUJUAN
Berikan rekomendasi trading dalam format JSON berdasarkan:
- Analisis tren teknikal dan indikator (EMA, RSI, MACD, Bollinger Bands, volume, candlestick)
- Struktur pasar, support/resistance
- Konteks berita terbaru
- Manajemen risiko ketat: Hanya berikan sinyal **BUY** jika **risk/reward ratio ≥ 1:3**

%s


### INPUT DATA HARGA (OHLC %s terakhir)
(Data OHLC seperti sebelumnya, tidak perlu diubah di sini) :
%s

### HARGA PASAR SAAT INI
%.2f (ini adalah harga pasar saat ini)

### KRITERIA ANALISIS TEKNIKAL
Analisis teknikal yang diperlukan:
1. Trend: BULLISH/BEARISH/SIDEWAYS
2. Technical indicators:
   - EMA signal (BULLISH, BEARISH, NEUTRAL)
   - RSI signal (OVERBOUGHT, OVERSOLD, NEUTRAL)
   - MACD signal (BULLISH, BEARISH, NEUTRAL)
   - Bollinger Bands position (UPPER/MIDDLE/LOWER)
3. Support dan resistance levels
4. Volume trend (HIGH/NORMAL/LOW) dan momentum
5. Candlestick pattern terbaru
6. Technical score (0-100)

### PANDUAN MANAJEMEN RISIKO
- Berikan **BUY signal** hanya jika:
  - Risk/reward ratio ≥ 1:3
  - Trend, indikator, dan volume mendukung
- Cut loss berdasarkan support kuat
- Target price harus realistis dan berdasarkan resistance  
- Maksimal holding 1-5 hari
- Ulangi analisis jika syarat tidak terpenuhi dan output sinyal: HOLD

### FORMAT OUTPUT (JSON):
{
  "symbol": "%s",
  "analysis_date": "%s",
  "signal": "BUY|HOLD",
  "max_holding_period_days": (1 sampai 5 hari),
  "technical_analysis": {
    "trend": "BULLISH|BEARISH|SIDEWAYS",
    "short_term_trend": "BULLISH|BEARISH|SIDEWAYS",
    "medium_term_trend": "BULLISH|BEARISH|SIDEWAYS",
    "ema_signal": "BULLISH|BEARISH|NEUTRAL",
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
    "action": "BUY|HOLD",
    "buy_price": (Harga pembelian),
    "target_price": (Harga target - risk_reward_ratio ≥ 1:3),
    "cut_loss": (Harga cut loss),
    "confidence_level": (Confidence level 0-100),
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
    "confidence_level": (Confidence level 0-100),
    "key_insights": [
      "Trend bullish dengan volume mendukung",
      "Technical indicators positif",
      "Support dan resistance teridentifikasi",
      "Risk/reward ratio menguntungkan"
    ]
  },
  "news_summary":{ (JIKA ADA NEWS SUMMARY)
    "confidence_score": (Confidence score 0.0 - 1.0),
    "sentiment": "positive, negative, neutral, mixed",
    "impact": "bullish, bearish, sideways"
    "key_issues": ["issue1", "issue2", "issue3"]
  }
}`, symbol, newsSummaryText, dataInfo.Range, string(ohlcvJSON), dataInfo.MarketPrice, symbol, time.Now().Format("2006-01-02T15:04:05-07:00"))

	return prompt
}

func (c *Client) buildPositionMonitoringPrompt(ctx context.Context,
	request models.PositionMonitoringRequest,
	ohlcvData []models.OHLCVData,
	dataInfo models.DataInfo,
	summary *models.StockNewsSummaryEntity,
) string {
	// Convert OHLCV data to JSON string
	ohlcvJSON, _ := json.Marshal(ohlcvData)

	// Ringkasan sentimen dari berita
	newsSummaryText := ""
	if summary != nil {
		newsSummaryText = fmt.Sprintf(`Berikut adalah ringkasan sentimen berita untuk saham %s selama periode %s hingga %s:

- Sentimen utama: %s
- Dampak terhadap harga: %s
- Key issues: %s
- Ringkasan singkat: %s
- Confidence score: %.2f
- Saran tindakan: %s
- Alasan: %s

Gunakan ringkasan ini untuk mempertimbangkan konteks eksternal (berita) dalam analisis teknikal berikut.
`,
			summary.StockCode,
			summary.SummaryStart.Format("2006-01-02"),
			summary.SummaryEnd.Format("2006-01-02"),
			summary.SummarySentiment,
			summary.SummaryImpact,
			strings.Join(summary.KeyIssues, ", "),
			summary.ShortSummary,
			summary.SummaryConfidenceScore,
			summary.SuggestedAction,
			summary.Reasoning,
		)
	}

	// Calculate remaining holding period
	positionAgeDays := int(time.Since(request.BuyTime).Hours() / 24)
	remainingDays := request.MaxHoldingPeriodDays - positionAgeDays
	if remainingDays < 0 {
		remainingDays = 0
	}

	prompt := fmt.Sprintf(`Anda adalah analis teknikal saham Indonesia yang ahli dalam swing trading. Analisis posisi trading yang sedang berjalan dan berikan rekomendasi HOLD/SELL/CUT_LOSS.
%s

Data posisi trading:
- Symbol: %s
- Buy Price: %.2f
- Buy Time: %s
- Max Holding Period: %d days
- Position Age: %d days
- Remaining Days: %d days


Data OHLC %s:
%s

Current Market Price: %.2f (ini adalah harga pasar saat ini)

Analisis yang diperlukan:
1. Hitung current profit/loss dan percentage
2. Analisis trend (short-term dan medium-term)
3. Technical indicators: EMA, RSI, MACD, Bollinger Bands
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
- Pertimbangkan Data Ringkasan Analisa Berita yang diberikan (JIKA ADA NEWS SUMMARY)
- Ulangi analisis Anda jika risk/reward tidak memenuhi. Jangan berikan sinyal BUY jika potensi kerugian lebih besar daripada potensi keuntungan. Ketatkan logika manajemen risiko seperti layaknya seorang trader profesional.


Return response dalam format JSON:
{
  "symbol": "%s",
  "max_holding_period_days": %d,
  "recommendation": {
    "action": "HOLD|SELL|CUT_LOSS",
    "reasoning": "Analisis teknikal menunjukkan momentum bullish dengan volume mendukung. EMA 9 di atas EMA 21, RSI 68.5 netral-positif, MACD bullish. Support 8950, resistance 9200. Saham masih dalam trend yang diharapkan dengan potential profit menarik. Dengan sisa %d hari, masih ada waktu untuk mencapai target.",
    "technical_analysis": {
      "trend": "BULLISH",
      "short_term_trend": "BULLISH",
      "medium_term_trend": "BULLISH",
      "ema_signal": "BULLISH",
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
      "risk_reward_ratio": 2.67 (PENTING UNTUK MENGGUNAKAN risk-reward ratio ≥ 1:3),
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
  },
  "news_summary":{ (JIKA ADA NEWS SUMMARY)
    "confidence_score": 0.0 - 1.0,
    "sentiment": "positive, negative, neutral, mixed",
    "impact": "bullish, bearish, sideways"
    "key_issues": ["issue1", "issue2", "issue3"]
  }
}`, request.Symbol, newsSummaryText, request.BuyPrice, request.BuyTime.Format("2006-01-02T15:04:05-07:00"),
		request.MaxHoldingPeriodDays, positionAgeDays, remainingDays, dataInfo.Range, string(ohlcvJSON),
		dataInfo.MarketPrice, remainingDays, request.Symbol, request.MaxHoldingPeriodDays,
		remainingDays, remainingDays, remainingDays, remainingDays, remainingDays)

	return prompt
}

// extractJSONFromText extracts JSON object from text that may contain additional content
func extractJSONFromText(ctx context.Context, text string) (string, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start == -1 || end == -1 || start > end {
		return "", fmt.Errorf("no JSON object found in response: %s", text)
	}
	return text[start : end+1], nil
}
