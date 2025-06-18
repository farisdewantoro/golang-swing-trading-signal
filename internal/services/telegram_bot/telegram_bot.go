package telegram_bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"

	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/services/trading_analysis"
	"golang-swing-trading-signal/internal/utils"
)

type TelegramBotService struct {
	bot           *telebot.Bot
	config        *config.TelegramConfig
	tradingConfig *config.TradingConfig
	logger        *logrus.Logger
	analyzer      *trading_analysis.Analyzer
	router        *gin.Engine
	ctx           context.Context
	cancel        context.CancelFunc
}

func NewTelegramBotService(cfg *config.TelegramConfig, tradingConfig *config.TradingConfig, logger *logrus.Logger, analyzer *trading_analysis.Analyzer, router *gin.Engine) (*TelegramBotService, error) {
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("telegram bot token is required")
	}

	// Always start with long polling to avoid webhook conflicts
	pref := telebot.Settings{
		Token:  cfg.BotToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		OnError: func(err error, c telebot.Context) {
			logger.WithError(err).Error("Telegram bot error")
		},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	service := &TelegramBotService{
		bot:           bot,
		config:        cfg,
		tradingConfig: tradingConfig,
		logger:        logger,
		analyzer:      analyzer,
		router:        router,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Register handlers
	service.registerHandlers()

	// Setup webhook routes
	service.setupWebhookRoutes()

	return service, nil
}

func (t *TelegramBotService) registerHandlers() {
	// Handle /start command
	t.bot.Handle("/start", t.handleStart)

	// Handle /help command
	t.bot.Handle("/help", t.handleHelp)

	// Handle stock analysis command
	t.bot.Handle("/analyze", t.handleAnalyze)

	// Handle position monitoring command
	t.bot.Handle("/monitor", t.handleMonitorPosition)

	// Handle buy list analysis command
	t.bot.Handle("/buylist", t.handleBuyList)

	// Handle text messages (stock symbols)
	t.bot.Handle(telebot.OnText, t.handleTextMessage)
}

func (t *TelegramBotService) setupWebhookRoutes() {
	// Webhook endpoint for Telegram updates
	t.router.POST("/telegram/webhook", t.handleWebhook)

	// Webhook setup endpoint
	t.router.POST("/telegram/set-webhook", t.setupWebhook)

	// Webhook info endpoint
	t.router.GET("/telegram/webhook-info", t.getWebhookInfo)
}

func (t *TelegramBotService) handleWebhook(c *gin.Context) {
	// Parse the update from Telegram
	var update telebot.Update
	if err := c.ShouldBindJSON(&update); err != nil {
		t.logger.WithError(err).Error("Failed to parse webhook update")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid update format"})
		return
	}

	// Process the update
	t.bot.ProcessUpdate(update)

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (t *TelegramBotService) setupWebhook(c *gin.Context) {
	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		t.logger.WithError(err).Error("Invalid webhook setup request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Set webhook URL directly using HTTP client to avoid telebot library issues
	client := &http.Client{Timeout: 10 * time.Second}

	// Create form data
	data := url.Values{}
	data.Set("url", request.URL)

	// Make request to Telegram API
	resp, err := client.PostForm(fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", t.config.BotToken), data)
	if err != nil {
		t.logger.WithError(err).Error("Failed to call Telegram API")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to call Telegram API",
			"message": err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.logger.WithError(err).Error("Failed to read response")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to read response",
			"message": err.Error(),
		})
		return
	}

	// Check if successful
	if resp.StatusCode == http.StatusOK {
		t.logger.WithField("webhook_url", request.URL).Info("Webhook set successfully")
		c.JSON(http.StatusOK, gin.H{
			"status":   "success",
			"message":  "Webhook set successfully",
			"url":      request.URL,
			"response": string(body),
		})
	} else {
		t.logger.WithError(fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))).Error("Failed to set webhook")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to set webhook",
			"message": fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		})
	}
}

func (t *TelegramBotService) getWebhookInfo(c *gin.Context) {
	info, err := t.bot.Webhook()
	if err != nil {
		t.logger.WithError(err).Error("Failed to get webhook info")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get webhook info",
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"webhook": gin.H{
			"url":             info.Listen,
			"max_connections": info.MaxConnections,
		},
	})
}

func (t *TelegramBotService) handleStart(c telebot.Context) error {
	message := `🚀 Welcome to Swing Trading Signal Bot!

I can help you analyze stocks and monitor your positions.

Commands available:
/analyze <symbol> - Analyze a specific stock (e.g., /analyze AAPL)
/monitor <symbol> <buy_price> <buy_date> <max_days> - Monitor a trading position (e.g., /monitor AAPL 150.00 2024-01-15 5)
/buylist - Analyze all stocks from config and show buy list
/help - Show this help message

You can also just send me a stock symbol (e.g., AAPL, GOOGL, TSLA)`

	return c.Send(message)
}

func (t *TelegramBotService) handleHelp(c telebot.Context) error {
	message := `📊 Swing Trading Signal Bot Help

Commands:
/analyze <symbol> <interval:optional> <period:optional> - Analyze a specific stock
/monitor <symbol> <buy_price> <buy_date> <max_days> <interval:optional> <period:optional> - Monitor a trading position
/buylist - Analyze all stocks from config and show buy list
/help - Show this help message

Examples:
/analyze BBCA 1d 2m
/analyze BBRI

/monitor ANTM 150.00 2024-01-15 5 1d 2m
/monitor BBRI 2800.00 2024-01-10 7

/buylist - Get buy list summary


The bot will provide:
• Technical analysis
• Risk/reward assessment
• Trading recommendations
• Confidence levels
• Position monitoring with P&L tracking
• Buy list analysis for all configured stocks
• Stock news summary analysis
`

	return c.Send(message)
}

func (t *TelegramBotService) handleAnalyze(c telebot.Context) error {
	args := c.Args()
	if len(args) == 0 {
		return c.Send("❌ Please provide a stock symbol.\nUsage: /analyze <symbol>\nExample: /analyze AAPL")
	}

	interval := "1d"
	if len(args) > 1 {
		interval = args[1]
	}

	period := "2m"
	if len(args) > 2 {
		period = args[2]
	}

	symbol := strings.ToUpper(args[0])
	return t.analyzeStock(c, symbol, interval, period)
}

func (t *TelegramBotService) handleTextMessage(c telebot.Context) error {
	text := strings.TrimSpace(c.Text())

	// Check if it's a command
	if strings.HasPrefix(text, "/") {
		return nil // Let other handlers deal with commands
	}

	return t.handleHelp(c)
}

func (t *TelegramBotService) analyzeStock(c telebot.Context, symbol string, interval string, period string) error {
	// Send initial message
	err := c.Send(fmt.Sprintf("🔍 Analyzing %s... Please wait.", symbol))
	if err != nil {
		t.logger.WithError(err).Error("Failed to send initial message")
		return err
	}

	// Run analysis in background goroutine
	go func() {
		// Check if context is cancelled before starting analysis
		select {
		case <-t.ctx.Done():
			t.logger.Info("Telegram bot shutting down, skipping analysis")
			return
		default:
		}

		// Perform analysis
		analysis, err := t.analyzer.AnalyzeStock(t.ctx, symbol, interval, period)
		if err != nil {
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to analyze stock")

			// Check if context is cancelled before sending error message
			select {
			case <-t.ctx.Done():
				t.logger.Info("Telegram bot shutting down, skipping error message")
				return
			default:
			}

			// Send error message
			err := c.Send(fmt.Sprintf("❌ Failed to analyze %s: %s", symbol, err.Error()))
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		// Check if context is cancelled before sending analysis
		select {
		case <-t.ctx.Done():
			t.logger.Info("Telegram bot shutting down, skipping analysis message")
			return
		default:
		}

		// Format analysis message
		analysisMessage := t.FormatAnalysisMessage(analysis)

		// Send the analysis results
		err = c.Send(analysisMessage, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML,
		})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send analysis message")
		}
	}()

	return nil
}

func (t *TelegramBotService) FormatAnalysisMessage(analysis *models.IndividualAnalysisResponse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📊 <b>Analysis for %s</b>\n", analysis.Symbol))
	sb.WriteString(fmt.Sprintf("📅 Date: %s\n", analysis.AnalysisDate.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("🎯 Signal: <b>%s</b>\n\n", analysis.Signal))

	// Technical Analysis Summary
	sb.WriteString("🔧 <b>Technical Analysis:</b>\n")
	sb.WriteString(fmt.Sprintf("Trend: %s (Strength: %s)\n", analysis.TechnicalAnalysis.Trend, analysis.TechnicalAnalysis.TrendStrength))
	sb.WriteString(fmt.Sprintf("EMA Signal: %s (9: $%.2f, 21: $%.2f)\n", analysis.TechnicalAnalysis.EMASignal, analysis.TechnicalAnalysis.EMA9, analysis.TechnicalAnalysis.EMA21))
	sb.WriteString(fmt.Sprintf("RSI: %.2f (%s)\n", analysis.TechnicalAnalysis.RSI, analysis.TechnicalAnalysis.RSISignal))
	sb.WriteString(fmt.Sprintf("MACD: %s | Momentum: %s\n", analysis.TechnicalAnalysis.MACDSignal, analysis.TechnicalAnalysis.Momentum))
	sb.WriteString(fmt.Sprintf("Volume: %s | Technical Score: %d/100\n\n", analysis.TechnicalAnalysis.VolumeTrend, analysis.TechnicalAnalysis.TechnicalScore))

	// News Summary
	sb.WriteString("📰 <b>News Summary Analysis:</b>\n")
	sb.WriteString(fmt.Sprintf("Confidence Score: %.2f\n", analysis.NewsSummary.ConfidenceScore))
	sb.WriteString(fmt.Sprintf("Sentiment: %s\n", analysis.NewsSummary.Sentiment))
	sb.WriteString(fmt.Sprintf("Impact: %s\n\n", analysis.NewsSummary.Impact))

	// Key Levels
	sb.WriteString("🎯 <b>Key Levels:</b>\n")
	sb.WriteString(fmt.Sprintf("Support: $%.2f | Resistance: $%.2f\n", analysis.TechnicalAnalysis.SupportLevel, analysis.TechnicalAnalysis.ResistanceLevel))
	if len(analysis.TechnicalAnalysis.KeySupportLevels) > 0 {
		sb.WriteString("Key Supports: ")
		for i, level := range analysis.TechnicalAnalysis.KeySupportLevels {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("$%.2f", level))
		}
		sb.WriteString("\n")
	}
	if len(analysis.TechnicalAnalysis.KeyResistanceLevels) > 0 {
		sb.WriteString("Key Resistances: ")
		for i, level := range analysis.TechnicalAnalysis.KeyResistanceLevels {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf("$%.2f", level))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// Recommendation
	sb.WriteString("💡 <b>Recommendation:</b>\n")
	sb.WriteString(fmt.Sprintf("Action: <b>%s</b>\n", analysis.Recommendation.Action))

	// Only show buy-related information when action is BUY
	if analysis.Recommendation.Action == "BUY" {
		if analysis.Recommendation.BuyPrice > 0 {
			sb.WriteString(fmt.Sprintf("Buy Price: $%.2f\n", analysis.Recommendation.BuyPrice))
		}
		if analysis.Recommendation.TargetPrice > 0 {
			sb.WriteString(fmt.Sprintf("Target Price: $%.2f\n", analysis.Recommendation.TargetPrice))
		}
		if analysis.Recommendation.CutLoss > 0 {
			sb.WriteString(fmt.Sprintf("Stop Loss: $%.2f\n", analysis.Recommendation.CutLoss))
		}
		if analysis.MaxHoldingPeriodDays > 0 {
			sb.WriteString(fmt.Sprintf("Max Holding Period: %d days\n", analysis.MaxHoldingPeriodDays))
		}
	}

	sb.WriteString(fmt.Sprintf("Confidence: %d%%\n\n", analysis.Recommendation.ConfidenceLevel))

	// Risk Analysis
	sb.WriteString("⚠️ <b>Risk Analysis:</b>\n")
	sb.WriteString(fmt.Sprintf("Risk Level: %s\n", analysis.RiskLevel))
	sb.WriteString(fmt.Sprintf("Potential Profit: %.2f%%\n", analysis.Recommendation.RiskRewardAnalysis.PotentialProfitPercentage))
	sb.WriteString(fmt.Sprintf("Potential Loss: %.2f%%\n", analysis.Recommendation.RiskRewardAnalysis.PotentialLossPercentage))
	sb.WriteString(fmt.Sprintf("Risk/Reward Ratio: %.2f\n", analysis.Recommendation.RiskRewardAnalysis.RiskRewardRatio))
	sb.WriteString(fmt.Sprintf("Success Probability: %d%%\n\n", analysis.Recommendation.RiskRewardAnalysis.SuccessProbability))

	// Technical Summary
	if analysis.TechnicalSummary.OverallSignal != "" {
		sb.WriteString("📋 <b>Summary:</b>\n")
		sb.WriteString(fmt.Sprintf("Overall Signal: %s\n", analysis.TechnicalSummary.OverallSignal))
		sb.WriteString(fmt.Sprintf("Volume Support: %s\n", analysis.TechnicalSummary.VolumeSupport))
		sb.WriteString(fmt.Sprintf("Confidence Level: %d%%\n", analysis.TechnicalSummary.ConfidenceLevel))

		if len(analysis.TechnicalSummary.KeyInsights) > 0 {
			sb.WriteString("Key Insights:\n")
			for _, insight := range analysis.TechnicalSummary.KeyInsights {
				sb.WriteString(fmt.Sprintf("• %s\n", insight))
			}
		}

		if len(analysis.NewsSummary.KeyIssues) > 0 {
			for _, issue := range analysis.NewsSummary.KeyIssues {
				sb.WriteString(fmt.Sprintf("• %s\n", issue))
			}
		}
		sb.WriteString("\n")
	}

	// Reasoning
	sb.WriteString("🧠 <b>Reasoning:</b>\n")
	sb.WriteString(analysis.Recommendation.Reasoning)

	return sb.String()
}

func (t *TelegramBotService) handleMonitorPosition(c telebot.Context) error {
	args := c.Args()
	if len(args) < 4 {
		return c.Send(`❌ Please provide all required parameters.
Usage: /monitor <symbol> <buy_price> <buy_date> <max_days> <interval :optional> <period_start :optional> <period_end :optional>
Example: /monitor AAPL 150.00 2024-01-15 5 1d 2m

Parameters:
- symbol: Stock symbol (e.g., AAPL)
- buy_price: Price when you bought the stock
- buy_date: Date when you bought (YYYY-MM-DD format)
- max_days: Maximum holding period in days`)
	}

	symbol := strings.ToUpper(args[0])

	// Parse buy price
	buyPrice, err := strconv.ParseFloat(args[1], 64)
	if err != nil {
		return c.Send("❌ Invalid buy price. Please provide a valid number.")
	}

	// Parse buy date
	buyDate, err := time.Parse("2006-01-02", args[2])
	if err != nil {
		return c.Send("❌ Invalid buy date. Please use YYYY-MM-DD format (e.g., 2024-01-15)")
	}

	// Parse max holding period
	maxDays, err := strconv.Atoi(args[3])
	if err != nil || maxDays <= 0 {
		return c.Send("❌ Invalid max holding period. Please provide a positive number.")
	}

	interval := "1d"

	if len(args) > 4 {
		interval = args[4]
	}

	period := "2m"
	if len(args) > 5 {
		period = args[5]
	}

	// Send initial message
	err = c.Send(fmt.Sprintf("🔍 Monitoring position for %s... Please wait.", symbol))
	if err != nil {
		t.logger.WithError(err).Error("Failed to send initial message")
		return err
	}

	// Run monitoring in background goroutine
	go func() {
		// Check if context is cancelled before starting monitoring
		select {
		case <-t.ctx.Done():
			t.logger.Info("Telegram bot shutting down, skipping position monitoring")
			return
		default:
		}

		// Create position monitoring request
		request := models.PositionMonitoringRequest{
			Symbol:               symbol,
			BuyPrice:             buyPrice,
			BuyTime:              buyDate,
			MaxHoldingPeriodDays: maxDays,
			Interval:             interval,
			Period:               period,
		}

		// Perform position monitoring
		analysis, err := t.analyzer.MonitorPosition(t.ctx, request)
		if err != nil {
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to monitor position")

			// Check if context is cancelled before sending error message
			select {
			case <-t.ctx.Done():
				t.logger.Info("Telegram bot shutting down, skipping error message")
				return
			default:
			}

			// Send error message
			err := c.Send(fmt.Sprintf("❌ Failed to monitor position for %s: %s", symbol, err.Error()))
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		// Check if context is cancelled before sending analysis
		select {
		case <-t.ctx.Done():
			t.logger.Info("Telegram bot shutting down, skipping analysis message")
			return
		default:
		}

		// Format position monitoring message
		analysisMessage := t.FormatPositionMonitoringMessage(analysis)

		// Send the position monitoring results
		err = c.Send(analysisMessage, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML,
		})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send position monitoring message")
		}
	}()

	return nil
}

func (t *TelegramBotService) SendPositionMonitoringNotification(position *models.PositionMonitoringResponse) error {
	if t.config.ChatID == "" {
		t.logger.Warn("Telegram chat ID not configured, skipping notification")
		return nil
	}

	chatID, err := strconv.ParseInt(t.config.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	message := t.FormatPositionMonitoringMessage(position)

	_, err = t.bot.Send(&telebot.Chat{ID: chatID}, message, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	})

	if err != nil {
		t.logger.WithError(err).Error("Failed to send position monitoring notification")
		return err
	}

	t.logger.WithField("symbol", position.Symbol).Info("Position monitoring notification sent")
	return nil
}

func (t *TelegramBotService) SendBulkPositionMonitoringNotification(positions []models.PositionMonitoringResponse) error {
	if t.config.ChatID == "" {
		t.logger.Warn("Telegram chat ID not configured, skipping notification")
		return nil
	}

	chatID, err := strconv.ParseInt(t.config.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	messages := t.FormatBulkPositionMonitoringMessage(positions)

	for _, message := range messages {
		_, err = t.bot.Send(&telebot.Chat{ID: chatID}, message, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML,
		})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send position monitoring notification")
			return err
		}
	}

	return nil
}

func (t *TelegramBotService) FormatPositionMonitoringMessage(position *models.PositionMonitoringResponse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📊 <b>Position Update: %s</b>\n", position.Symbol))
	sb.WriteString(fmt.Sprintf("💰 Buy: $%.2f | Current: $%.2f | P&L: %.2f%%\n", position.BuyPrice, position.CurrentPrice, position.PositionMetrics.UnrealizedPnLPercentage))
	sb.WriteString(fmt.Sprintf("📈 Age: %d days | Remaining: %d days\n\n", position.PositionAgeDays, position.PositionMetrics.DaysRemaining))

	// Recommendation
	sb.WriteString("💡 <b>Recommendation:</b>\n")
	sb.WriteString(fmt.Sprintf("Action: <b>%s</b>\n", position.Recommendation.Action))
	sb.WriteString(fmt.Sprintf("Reasoning: %s\n\n", position.Recommendation.Reasoning))

	// Technical Analysis
	sb.WriteString("🔧 <b>Technical Analysis:</b>\n")
	sb.WriteString(fmt.Sprintf("Trend: %s (Strength: %s)\n", position.Recommendation.TechnicalAnalysis.Trend, position.Recommendation.TechnicalAnalysis.TrendStrength))
	sb.WriteString(fmt.Sprintf("EMA: %s | RSI: %.1f (%s)\n", position.Recommendation.TechnicalAnalysis.EMASignal, position.Recommendation.TechnicalAnalysis.RSI, position.Recommendation.TechnicalAnalysis.RSISignal))
	sb.WriteString(fmt.Sprintf("MACD: %s | Volume: %s\n", position.Recommendation.TechnicalAnalysis.MACDSignal, position.Recommendation.TechnicalAnalysis.VolumeTrend))
	sb.WriteString(fmt.Sprintf("Support: $%.2f | Resistance: $%.2f\n", position.Recommendation.TechnicalAnalysis.SupportLevel, position.Recommendation.TechnicalAnalysis.ResistanceLevel))
	sb.WriteString(fmt.Sprintf("Technical Score: %d/100\n\n", position.Recommendation.TechnicalAnalysis.TechnicalScore))

	// News Summary
	sb.WriteString("📰 <b>News Summary:</b>\n")
	sb.WriteString(fmt.Sprintf("Confidence Score: %.2f\n", position.NewsSummary.ConfidenceScore))
	sb.WriteString(fmt.Sprintf("Sentiment: %s\n", position.NewsSummary.Sentiment))
	sb.WriteString(fmt.Sprintf("Impact: %s\n\n", position.NewsSummary.Impact))

	// Risk Analysis
	sb.WriteString("⚠️ <b>Risk Analysis:</b>\n")
	sb.WriteString(fmt.Sprintf("Current Profit: %.2f%%\n", position.Recommendation.RiskRewardAnalysis.CurrentProfitPercentage))
	sb.WriteString(fmt.Sprintf("Remaining Potential: %.2f%%\n", position.Recommendation.RiskRewardAnalysis.RemainingPotentialProfitPercentage))
	sb.WriteString(fmt.Sprintf("Risk/Reward Ratio: %.2f\n", position.Recommendation.RiskRewardAnalysis.RiskRewardRatio))
	sb.WriteString(fmt.Sprintf("Success Probability: %d%%\n\n", position.Recommendation.RiskRewardAnalysis.SuccessProbability))

	// Exit Strategy
	if position.Recommendation.RiskRewardAnalysis.ExitRecommendation.TargetExitPrice > 0 {
		sb.WriteString("🎯 <b>Exit Strategy:</b>\n")
		sb.WriteString(fmt.Sprintf("Target: $%.2f | Stop Loss: $%.2f\n",
			position.Recommendation.RiskRewardAnalysis.ExitRecommendation.TargetExitPrice,
			position.Recommendation.RiskRewardAnalysis.ExitRecommendation.StopLossPrice))
		if position.Recommendation.RiskRewardAnalysis.ExitRecommendation.TimeBasedExit != "" {
			if t, err := time.Parse(time.RFC3339, position.Recommendation.RiskRewardAnalysis.ExitRecommendation.TimeBasedExit); err == nil {
				sb.WriteString(fmt.Sprintf("Time Exit: %s\n", t.Format("2006-01-02 15:04:05")))
			}
		}
		if len(position.Recommendation.RiskRewardAnalysis.ExitRecommendation.ExitConditions) > 0 {
			sb.WriteString("Exit Conditions:\n")
			for i, condition := range position.Recommendation.RiskRewardAnalysis.ExitRecommendation.ExitConditions {
				if i < 2 { // Limit to 2 most important conditions
					sb.WriteString(fmt.Sprintf("• %s\n", condition))
				}
			}
		}
		sb.WriteString("\n")
	}

	// Summary
	if position.TechnicalSummary.OverallSignal != "" {
		sb.WriteString("📋 <b>Summary:</b>\n")
		sb.WriteString(fmt.Sprintf("Signal: %s | Confidence: %d%%\n", position.TechnicalSummary.OverallSignal, position.TechnicalSummary.ConfidenceLevel))
		if len(position.TechnicalSummary.KeyInsights) > 0 {
			sb.WriteString("Key Insights:\n")
			for _, insight := range position.TechnicalSummary.KeyInsights {
				sb.WriteString(fmt.Sprintf("• %s\n", insight))

			}
		}

		if len(position.NewsSummary.KeyIssues) > 0 {
			for _, issue := range position.NewsSummary.KeyIssues {
				sb.WriteString(fmt.Sprintf("• %s\n", issue))

			}
		}
	}

	return sb.String()
}

func (t *TelegramBotService) FormatBulkPositionMonitoringMessage(positions []models.PositionMonitoringResponse) []string {
	const maxLen = 4090
	var messages []string
	var currentMessage strings.Builder
	part := 1

	now := utils.TimeNowWIB()
	// Helper function to start a new message part with the correct header
	startNewPart := func() {
		currentMessage.Reset()
		var header string
		if part == 1 {
			header = "📊 <b>Position Update Harian </b>\n"
		} else {
			header = fmt.Sprintf("---*Lanjutan Position Update Harian Part %d*---\n\n", part)
		}
		currentMessage.WriteString(header)
		currentMessage.WriteString(utils.PrettyDate(now) + "\n\n")
	}

	// Start the first part
	startNewPart()

	for _, position := range positions {

		var entryBuilder strings.Builder
		entryBuilder.WriteString(fmt.Sprintf("💼 <b>$%s</b>\n", position.Symbol))
		entryBuilder.WriteString(fmt.Sprintf("💰 Buy: $%.2f | Current: $%.2f | P&L: %.2f%%\n", position.BuyPrice, position.CurrentPrice, position.PositionMetrics.UnrealizedPnLPercentage))
		entryBuilder.WriteString(fmt.Sprintf("📈 Age: %d days | Remaining: %d days\n", position.PositionAgeDays, position.PositionMetrics.DaysRemaining))
		entryBuilder.WriteString(fmt.Sprintf("🎯 TP: $%.2f | SL: $%.2f\n",
			position.Recommendation.RiskRewardAnalysis.ExitRecommendation.TargetExitPrice,
			position.Recommendation.RiskRewardAnalysis.ExitRecommendation.StopLossPrice))
		// Suggested Action with icon
		var actionIcon string
		switch strings.ToLower(position.Recommendation.Action) {
		case "buy":
			actionIcon = "🟢"
		case "sell":
			actionIcon = "🔴"
		default: // Hold, Neutral
			actionIcon = "🟡"
		}

		entryBuilder.WriteString(fmt.Sprintf("📌 Action: %s %s\n", actionIcon, position.Recommendation.Action))
		entryBuilder.WriteString(fmt.Sprintf("🧠 Success Probability: %d%%\n", position.Recommendation.RiskRewardAnalysis.SuccessProbability))
		entryBuilder.WriteString(fmt.Sprintf("🔍 Reasoning: %s\n\n\n", position.Recommendation.Reasoning))

		// Check if adding the new entry exceeds the max length. We assume a single entry doesn't exceed the limit.
		if currentMessage.Len()+len(entryBuilder.String()) > maxLen {
			// Finalize the current message and add it to the slice
			messages = append(messages, currentMessage.String())

			// Start a new part
			part++
			startNewPart()
		}

		// Add the entry to the current message
		currentMessage.WriteString(entryBuilder.String())
	}

	// Add the final message part to the slice
	messages = append(messages, currentMessage.String())

	return messages
}

func (t *TelegramBotService) Start() {
	t.logger.Info("Starting Telegram bot...")

	// If webhook URL is configured, set it up
	if t.config.WebhookURL != "" {
		t.logger.WithField("webhook_url", t.config.WebhookURL).Info("Setting up webhook...")

		// Set webhook URL directly using HTTP client to avoid telebot library issues
		client := &http.Client{Timeout: 10 * time.Second}

		// Create form data
		data := url.Values{}
		data.Set("url", t.config.WebhookURL)

		// Make request to Telegram API
		resp, err := client.PostForm(fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", t.config.BotToken), data)
		if err != nil {
			t.logger.WithError(err).Error("Failed to call Telegram API for webhook setup")
			return
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.logger.WithError(err).Error("Failed to read webhook response")
			return
		}

		// Check if successful
		if resp.StatusCode == http.StatusOK {
			t.logger.WithField("webhook_url", t.config.WebhookURL).Info("Webhook set successfully")
		} else {
			t.logger.WithError(fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))).Error("Failed to set webhook, falling back to long polling")
		}
	} else {
		t.logger.Info("No webhook URL configured, using long polling")
	}
}

func (t *TelegramBotService) Stop() {
	t.logger.Info("Stopping Telegram bot...")

	// Cancel the context to signal shutdown
	t.cancel()

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop the bot with timeout
	stopDone := make(chan error, 1)
	go func() {
		// Use a separate goroutine to avoid blocking
		t.bot.Stop()
		stopDone <- nil
	}()

	// Wait for bot to stop with timeout
	select {
	case <-stopDone:
		t.logger.Info("Telegram bot stopped successfully")
	case <-ctx.Done():
		t.logger.Warn("Timeout while stopping bot, forcing shutdown")
	}

	t.logger.Info("Telegram bot shutdown completed")
}

// handleBuyList handles /buylist command - analyzes all stocks and shows buy list
func (t *TelegramBotService) handleBuyList(c telebot.Context) error {
	// Send initial message with estimation
	startTime := time.Now()
	estimatedTime := time.Duration(len(t.tradingConfig.StockList)) * 3 * time.Second // Estimate 3 seconds per stock
	err := c.Send(fmt.Sprintf("🔍 Analyzing all stocks from configuration to generate buy list...\n⏱️ Estimated time: %s\nPlease wait.", formatDuration(estimatedTime)))
	if err != nil {
		t.logger.WithError(err).Error("Failed to send initial message")
		return err
	}

	// Run analysis in background goroutine
	go func() {
		// Check if context is cancelled before starting analysis
		select {
		case <-t.ctx.Done():
			t.logger.Info("Telegram bot shutting down, skipping buy list analysis")
			return
		default:
		}

		// Perform analysis on all stocks
		summary, err := t.analyzer.AnalyzeAllStocks(t.ctx, t.tradingConfig.StockList)
		if err != nil {
			t.logger.WithError(err).Error("Failed to analyze all stocks")

			// Check if context is cancelled before sending error message
			select {
			case <-t.ctx.Done():
				t.logger.Info("Telegram bot shutting down, skipping error message")
				return
			default:
			}

			// Send error message
			err := c.Send(fmt.Sprintf("❌ Failed to analyze stocks: %s", err.Error()))
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		// Check if context is cancelled before sending analysis
		select {
		case <-t.ctx.Done():
			t.logger.Info("Telegram bot shutting down, skipping buy list message")
			return
		default:
		}

		// Calculate actual time taken
		actualTime := time.Since(startTime)

		// Format buy list summary message
		summaryMessage := t.FormatBuyListSummaryMessage(summary, actualTime)

		// Send the buy list summary results
		err = c.Send(summaryMessage, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML,
		})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send buy list summary message")
		}

		// Check if context is cancelled before sending detailed list
		select {
		case <-t.ctx.Done():
			t.logger.Info("Telegram bot shutting down, skipping detailed stock list")
			return
		default:
		}

		// Send detailed stock list as second message
		if len(summary.BuyList) > 0 {
			detailedMessage := t.FormatDetailedStockListMessage(summary)
			err = c.Send(detailedMessage, &telebot.SendOptions{
				ParseMode: telebot.ModeHTML,
			})
			if err != nil {
				t.logger.WithError(err).Error("Failed to send detailed stock list message")
			}
		}
	}()

	return nil
}

// formatDuration formats duration in a human-readable format
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	}
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%d minutes %d seconds", minutes, seconds)
}

// FormatBuyListSummaryMessage formats the buy list analysis summary for Telegram
func (t *TelegramBotService) FormatBuyListSummaryMessage(summary *models.SummaryAnalysisResponse, analysisTime time.Duration) string {
	var sb strings.Builder

	sb.WriteString("📊 <b>BUY LIST ANALYSIS SUMMARY</b>\n")
	sb.WriteString(fmt.Sprintf("📅 Date: %s\n", summary.AnalysisDate.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("⏱️ Analysis Time: %s\n", formatDuration(analysisTime)))
	sb.WriteString(fmt.Sprintf("📈 Total Stocks: %d | Buy: %d | Hold: %d\n\n", summary.TotalStocks, summary.BuyCount, summary.HoldCount))

	sb.WriteString("📋 <b>MARKET OVERVIEW:</b>\n")
	sb.WriteString(fmt.Sprintf("Best Opportunity: %s\n", summary.Summary.BestOpportunity))
	sb.WriteString(fmt.Sprintf("Worst Opportunity: %s\n\n", summary.Summary.WorstOpportunity))

	// Buy List Summary
	if len(summary.BuyList) > 0 {
		// Show top 3 stocks with highest confidence
		sb.WriteString("🏆 <b>TOP RECOMMENDATIONS:</b>\n")
		for i, stock := range summary.BuyList {
			if i >= 3 { // Limit to top 3
				break
			}
			sb.WriteString(fmt.Sprintf("%d. <b>%s</b> - $%.2f (Confidence: %d%%)\n", i+1, stock.Symbol, stock.CurrentPrice, stock.Confidence))
		}
		sb.WriteString("\n📋 Detailed list will be sent in the next message...\n")
	} else {
		sb.WriteString("🟢 <b>RECOMMENDED BUY LIST:</b> No stocks recommended for buying at this time\n\n")
	}

	return sb.String()
}

// FormatDetailedStockListMessage formats the detailed stock list for Telegram
func (t *TelegramBotService) FormatDetailedStockListMessage(summary *models.SummaryAnalysisResponse) string {
	var sb strings.Builder

	sb.WriteString("📋 <b>DETAILED STOCK LIST</b>\n")
	sb.WriteString(fmt.Sprintf("📅 Analysis Date: %s\n\n", summary.AnalysisDate.Format("2006-01-02 15:04:05")))

	if len(summary.BuyList) > 0 {
		sb.WriteString("🟢 <b>RECOMMENDED BUY LIST:</b>\n\n")
		for i, stock := range summary.BuyList {
			sb.WriteString(fmt.Sprintf("%d. <b>%s</b> - $%.2f\n", i+1, stock.Symbol, stock.CurrentPrice))
			sb.WriteString(fmt.Sprintf("   💰 Buy: $%.2f | Target: $%.2f | Cut Loss: $%.2f\n", stock.BuyPrice, stock.TargetPrice, stock.StopLoss))
			sb.WriteString(fmt.Sprintf("   📈 Profit: %.2f%% | Risk Ratio: %.2f | Max Days: %d\n", stock.ProfitPercentage, stock.RiskRatio, stock.MaxHoldingDays))
			sb.WriteString(fmt.Sprintf("   🎯 Confidence: %d%% | Risk: %s\n\n", stock.Confidence, stock.RiskLevel))
		}
	} else {
		sb.WriteString("🟢 <b>RECOMMENDED BUY LIST:</b> No stocks recommended for buying at this time\n\n")
	}

	return sb.String()
}

// FormatBuyListMessage formats the buy list analysis for Telegram (keeping for backward compatibility)
func (t *TelegramBotService) FormatBuyListMessage(summary *models.SummaryAnalysisResponse) string {
	var sb strings.Builder

	sb.WriteString("📊 <b>BUY LIST ANALYSIS</b>\n")
	sb.WriteString(fmt.Sprintf("📅 Date: %s\n", summary.AnalysisDate.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("📈 Total Stocks: %d | Buy: %d | Hold: %d\n\n", summary.TotalStocks, summary.BuyCount, summary.HoldCount))

	sb.WriteString("📋 <b>MARKET OVERVIEW:</b>\n")
	sb.WriteString(fmt.Sprintf("Best Opportunity: %s\n", summary.Summary.BestOpportunity))
	sb.WriteString(fmt.Sprintf("Worst Opportunity: %s\n\n", summary.Summary.WorstOpportunity))

	return sb.String()
}
