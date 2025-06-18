package telegram_bot

import (
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) registerHandlers() {
	// Command handlers
	t.bot.Handle("/start", t.handleStart)
	t.bot.Handle("/help", t.handleHelp)
	t.bot.Handle("/analyze", t.handleAnalyze)
	t.bot.Handle("/buylist", t.handleBuyList)
	t.bot.Handle("/setposition", t.handleSetPosition)
	t.bot.Handle("/cancel", t.handleCancel)

	// Callback handlers for inline buttons
	t.bot.Handle(&telebot.Btn{Unique: "btn_general_analysis"}, t.handleGeneralAnalysis)
	t.bot.Handle(&telebot.Btn{Unique: "btn_position_analysis"}, t.handlePositionAnalysis)

	// Handle incoming text messages for conversations
	t.bot.Handle(telebot.OnText, t.handleConversation)

	// Handle webhook setup
	t.router.POST("/telegram/webhook", func(c *gin.Context) {
		var update telebot.Update
		if err := c.ShouldBindJSON(&update); err != nil {
			t.logger.WithError(err).Error("Cannot bind JSON")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		t.bot.ProcessUpdate(update)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
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
/analyze <symbol> - Analyze a specific stock (e.g., /analyze BBCA)
/buylist - Analyze all stocks from config and show buy list
/setposition - Set a new trading position interactively
/help - Show this help message

You can also just send me a stock symbol (e.g., BBCA, ANTM, BBRI)`

	return c.Send(message)
}

func (t *TelegramBotService) handleAnalyze(c telebot.Context) error {
	userID := c.Sender().ID

	// If user is already in a conversation, cancel it first
	if _, inConversation := t.userStates[userID]; inConversation {
		t.handleCancel(c)
	}

	// Start a new conversation for analysis
	t.userStates[userID] = StateWaitingAnalyzeSymbol
	t.userAnalysisPositionData[userID] = &analysisPositionData{} // Reuse this to store the symbol

	return c.Send("Silakan masukkan simbol saham yang ingin Anda analisis (contoh: BBCA, ANTM).")
}

func (t *TelegramBotService) handleHelp(c telebot.Context) error {
	helpMessage := `
<b>Welcome to the Swing Trading Signal Bot!</b>

Here are the available commands:
/start - Start interacting with the bot
/help - Show this help message
/analyze [symbol] - Analyze a specific stock symbol (e.g., /analyze ANTM)
/buylist - Get a list of recommended stocks to buy
/setposition - Set a new trading position interactively
/cancel - Cancel the current operation (like /setposition)

You can also send a stock symbol directly (e.g., ANTM) to get a quick analysis.

<b>Analysis Parameters:</b>
The bot uses the following default parameters for analysis:
- <b>Interval</b>: 1 day
- <b>Period</b>: 3 months

<b>How it works:</b>
The bot analyzes stock data based on technical indicators to identify potential swing trading opportunities. It provides buy signals, target prices, and stop-loss levels.

For more information, please contact the administrator.
`
	return c.Send(helpMessage, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	})
}

func (t *TelegramBotService) handleSetPosition(c telebot.Context) error {
	userID := c.Sender().ID
	t.userStates[userID] = StateWaitingBuyPrice
	t.userPositionData[userID] = &positionData{}
	t.logger.Infof("Starting /setposition for user %d", userID)
	return c.Send("Silakan masukkan Buy Price (contoh: 150.75):", &telebot.SendOptions{})
}

func (t *TelegramBotService) handleCancel(c telebot.Context) error {
	userID := c.Sender().ID

	// Check if user is in any conversation state
	if state, ok := t.userStates[userID]; ok && state != StateIdle {
		// Reset user state
		delete(t.userStates, userID)

		// Clear data
		if state >= StateWaitingAnalysisPositionSymbol && state <= StateWaitingAnalysisPositionPeriod {
			delete(t.userAnalysisPositionData, userID)
		} else if state > StateIdle && state < StateWaitingAnalysisPositionSymbol {
			delete(t.userPositionData, userID)
		}

		return c.Send("✅ Percakapan dibatalkan.")
	}

	return c.Send("🤷‍♀️ Tidak ada percakapan aktif yang bisa dibatalkan.")
}

func (t *TelegramBotService) handleConversation(c telebot.Context) error {
	userID := c.Sender().ID

	state, ok := t.userStates[userID]
	if !ok || state == StateIdle {
		// This should not be treated as a conversation.
		// Let the generic text handler deal with it.
		return t.handleTextMessage(c)
	}

	// Route to the correct conversation handler based on state range
	switch {
	case state >= StateWaitingGeneralAnalysisInterval && state <= StateWaitingGeneralAnalysisPeriod:
		return t.handleGeneralAnalysisConversation(c)
	case state >= StateWaitingAnalysisPositionSymbol && state <= StateWaitingAnalysisPositionPeriod:
		return t.handleAnalyzePositionConversation(c)
	case state >= StateWaitingSymbol && state <= StateWaitingAlertPrice:
		return t.handleSetPositionConversation(c)
	case state >= StateWaitingAnalyzeSymbol && state <= StateWaitingAnalysisType:
		return t.handleNewAnalyzeConversation(c)
	default:
		// If no specific conversation is matched, maybe it's a dangling state.
		delete(t.userStates, userID)
		return c.Send("Sepertinya Anda tidak sedang dalam percakapan aktif. Gunakan /help untuk melihat perintah yang tersedia.")
	}
}

func (t *TelegramBotService) handleGeneralAnalysis(c telebot.Context) error {
	userID := c.Sender().ID

	// Acknowledge the callback first
	if err := c.Respond(&telebot.CallbackResponse{Text: "Baik, melanjutkan ke analisis umum..."}); err != nil {
		t.logger.WithError(err).Warn("Failed to respond to general analysis callback")
	}

	// Update state to start collecting interval
	t.userStates[userID] = StateWaitingGeneralAnalysisInterval

	// The symbol is already in userAnalysisPositionData from the initial /analyze command

	// Ask for interval
	return c.Send("Silakan masukkan interval analisis (contoh: 1d, 1h). Kosongkan untuk default (1d).")
}

func (t *TelegramBotService) handleGeneralAnalysisConversation(c telebot.Context) error {
	userID := c.Sender().ID
	text := c.Text()
	state := t.userStates[userID]

	data, ok := t.userAnalysisPositionData[userID]
	if !ok {
		delete(t.userStates, userID)
		return c.Send("Terjadi kesalahan internal (data analisis tidak ditemukan), silakan mulai lagi dengan /analyze.")
	}

	switch state {
	case StateWaitingGeneralAnalysisInterval:
		interval := text
		if interval == "" {
			interval = "1d" // default
		}
		data.Interval = interval
		t.userStates[userID] = StateWaitingGeneralAnalysisPeriod
		return c.Send("Interval diterima. Silakan masukkan periode analisis (contoh: 3mo, 1y). Kosongkan untuk default (3mo).")

	case StateWaitingGeneralAnalysisPeriod:
		period := text
		if period == "" {
			period = "3mo" // default
		}
		data.Period = period

		// All data collected, run analysis
		c.Send(fmt.Sprintf("✅ Data diterima. Memulai analisis untuk %s dengan interval %s dan periode %s.", data.Symbol, data.Interval, data.Period))

		// Clean up state
		delete(t.userStates, userID)
		delete(t.userAnalysisPositionData, userID)

		// Execute analysis
		return t.analyzeStock(c, data.Symbol, data.Interval, data.Period)
	}

	return nil
}

func (t *TelegramBotService) handleSetPositionConversation(c telebot.Context) error {
	userID := c.Sender().ID
	text := c.Text()
	state := t.userStates[userID] // We already know the state exists

	data, data_ok := t.userPositionData[userID]
	if !data_ok {
		// Should not happen, but as a safeguard
		delete(t.userStates, userID)
		return c.Send("Terjadi kesalahan internal (data posisi tidak ditemukan), silakan coba lagi dengan /setposition.")
	}

	switch state {
	case StateWaitingSymbol:
		data.Symbol = strings.ToUpper(text)
		t.userStates[userID] = StateWaitingBuyPrice
		return c.Send(fmt.Sprintf("Simbol %s diterima. Silakan masukkan harga beli:", data.Symbol))

	case StateWaitingBuyPrice:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga beli tidak valid. Silakan masukkan angka (contoh: 150.5).")
		}
		data.BuyPrice = price
		t.userStates[userID] = StateWaitingBuyDate
		return c.Send("Harga beli diterima. Silakan masukkan tanggal beli (format: YYYY-MM-DD):")

	case StateWaitingBuyDate:
		_, err := time.Parse("2006-01-02", text)
		if err != nil {
			return c.Send("Format tanggal tidak valid. Silakan gunakan format YYYY-MM-DD.")
		}
		data.BuyDate = text
		t.userStates[userID] = StateWaitingTakeProfit
		return c.Send("Tanggal beli diterima. Silakan masukkan harga take profit (contoh: 180.0):")

	case StateWaitingTakeProfit:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga take profit tidak valid. Silakan masukkan angka.")
		}
		data.TakeProfit = price
		t.userStates[userID] = StateWaitingStopLoss
		return c.Send("Harga take profit diterima. Silakan masukkan harga stop loss (contoh: 140.0):")

	case StateWaitingStopLoss:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga stop loss tidak valid. Silakan masukkan angka.")
		}
		data.StopLoss = price
		t.userStates[userID] = StateWaitingMaxHolding
		return c.Send("Harga stop loss diterima. Silakan masukkan maksimal hari hold (contoh: 10):")

	case StateWaitingMaxHolding:
		intVal, err := strconv.Atoi(text)
		if err != nil || intVal <= 0 {
			return c.Send("Format maksimal hari hold tidak valid. Silakan masukkan angka bulat positif.")
		}
		data.MaxHolding = intVal
		t.userStates[userID] = StateWaitingAlertPrice
		return c.Send("Maksimal hari hold diterima. Apakah Anda ingin mengaktifkan notifikasi jika harga mendekati support/resistance? (ya/tidak):")

	case StateWaitingAlertPrice:
		answer := strings.ToLower(text)
		if answer != "ya" && answer != "tidak" {
			return c.Send(`Jawaban tidak valid. Silakan masukkan "ya" atau "tidak".`)
		}
		data.AlertPrice = (answer == "ya")

		// All data collected, process it
		c.Send(fmt.Sprintf("✅ Posisi berhasil diatur dengan data berikut:\n`Beli di: %.2f`\n`Tanggal: %s`\n`Take Profit: %.2f`\n`Stop Loss: %.2f`\n`Max Hold: %d hari`\n`Alert: %t`\n\nData ini akan digunakan untuk pemantauan di masa mendatang.",
			data.BuyPrice, data.BuyDate, data.TakeProfit, data.StopLoss, data.MaxHolding, data.AlertPrice),
			&telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

		// Clean up state
		delete(t.userStates, userID)
		delete(t.userPositionData, userID)
	}
	return nil
}

func (t *TelegramBotService) handleAnalyzePositionConversation(c telebot.Context) error {
	userID := c.Sender().ID
	text := c.Text()
	state := t.userStates[userID]

	data, data_ok := t.userAnalysisPositionData[userID]
	if !data_ok {
		delete(t.userStates, userID)
		return c.Send("Terjadi kesalahan internal (data analisis posisi tidak ditemukan), silakan mulai lagi.")
	}

	switch state {
	case StateWaitingAnalysisPositionSymbol:
		data.Symbol = strings.ToUpper(text)
		t.userStates[userID] = StateWaitingAnalysisPositionBuyPrice
		return c.Send(fmt.Sprintf("Simbol %s diterima. Silakan masukkan harga beli (contoh: 150.5):", data.Symbol))

	case StateWaitingAnalysisPositionBuyPrice:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			c.Send("Format harga beli tidak valid. Silakan masukkan angka.")
			return nil
		}
		data.BuyPrice = price
		t.userStates[userID] = StateWaitingAnalysisPositionBuyDate
		return c.Send("Harga beli diterima. Silakan masukkan tanggal beli (format: YYYY-MM-DD):")

	case StateWaitingAnalysisPositionBuyDate:
		_, err := time.Parse("2006-01-02", text)
		if err != nil {
			c.Send("Format tanggal tidak valid. Silakan gunakan format YYYY-MM-DD.")
			return nil
		}
		data.BuyDate = text
		t.userStates[userID] = StateWaitingAnalysisPositionMaxDays
		return c.Send("Tanggal beli diterima. Silakan masukkan maksimal hari pemantauan (contoh: 5):")

	case StateWaitingAnalysisPositionMaxDays:
		intVal, err := strconv.Atoi(text)
		if err != nil || intVal <= 0 {
			c.Send("Format hari tidak valid. Silakan masukkan angka bulat positif.")
			return nil
		}
		data.MaxDays = intVal
		t.userStates[userID] = StateWaitingAnalysisPositionInterval
		return c.Send("Hari diterima. Silakan masukkan interval (contoh: 1d, 1h). Kosongkan untuk default (1d):")

	case StateWaitingAnalysisPositionInterval:
		data.Interval = text
		t.userStates[userID] = StateWaitingAnalysisPositionPeriod
		return c.Send("Interval diterima. Silakan masukkan periode (contoh: 1mo, 2m). Kosongkan untuk default (2m):")

	case StateWaitingAnalysisPositionPeriod:
		data.Period = text
		c.Send(fmt.Sprintf("✅ Pemantauan untuk %s berhasil diatur. Saya akan mulai memantau dan memberi tahu Anda hasilnya.", data.Symbol))

		// Clean up state first
		delete(t.userStates, userID)
		delete(t.userAnalysisPositionData, userID)

		// Execute the monitoring in the background
		t.executeAnalyzePosition(c, data)
		return nil // Explicitly return nil to satisfy linter
	}
	return nil
}

// handleNewAnalyzeConversation handles the first part of the new /analyze flow
func (t *TelegramBotService) handleNewAnalyzeConversation(c telebot.Context) error {
	userID := c.Sender().ID
	text := c.Text()
	state := t.userStates[userID]

	switch state {
	case StateWaitingAnalyzeSymbol:
		symbol := strings.ToUpper(text)
		data, ok := t.userAnalysisPositionData[userID]
		if !ok {
			c.Send("Terjadi kesalahan internal, silakan mulai lagi dengan /analyze.")
			delete(t.userStates, userID)
			return nil
		}
		data.Symbol = symbol
		t.userStates[userID] = StateWaitingAnalysisType

		menu := &telebot.ReplyMarkup{}
		btnGeneral := menu.Data("Analisis Umum", "btn_general_analysis", symbol)
		btnPosition := menu.Data("Analisis Posisi", "btn_position_analysis", symbol)
		menu.Inline(
			menu.Row(btnGeneral, btnPosition),
		)

		return c.Send(fmt.Sprintf("Simbol %s diterima. Silakan pilih jenis analisis:", symbol), menu)

	case StateWaitingAnalysisType:
		return c.Send("Silakan pilih salah satu opsi di atas dengan menekan tombol.")
	}
	return nil
}

// handlePositionAnalysis is the callback for the 'Position Analysis' button
func (t *TelegramBotService) handlePositionAnalysis(c telebot.Context) error {
	userID := c.Sender().ID
	symbol := c.Data() // The symbol is passed as data

	// Respond to the callback query
	c.Respond()
	// Delete the message with the buttons
	t.bot.Delete(c.Message())

	data, ok := t.userAnalysisPositionData[userID]
	if !ok {
		// This shouldn't happen, but as a safeguard:
		delete(t.userStates, userID)
		return c.Send("Terjadi kesalahan internal. Silakan mulai lagi dengan /analyze.")
	}
	data.Symbol = symbol // Ensure symbol is set correctly

	// Transition to the next state in the position analysis flow
	t.userStates[userID] = StateWaitingAnalysisPositionBuyPrice

	return c.Send(fmt.Sprintf("Analisis Posisi untuk %s. Silakan masukkan harga beli (contoh: 150.5):", symbol))
}

// executeAnalyzePosition performs the actual position analysis after collecting all data
func (t *TelegramBotService) executeAnalyzePosition(c telebot.Context, data *analysisPositionData) {
	// Default values for optional fields
	if data.Interval == "" {
		data.Interval = "1d"
	}
	if data.Period == "" {
		data.Period = "2m"
	}

	// Parse validated data
	symbol := data.Symbol
	buyPrice := data.BuyPrice
	buyDate, _ := time.Parse("2006-01-02", data.BuyDate)
	maxDays := data.MaxDays
	interval := data.Interval
	period := data.Period

	// Run analysis in background goroutine
	go func() {
		// Check if context is cancelled before starting analysis
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
		position, err := t.analyzer.MonitorPosition(t.ctx, request)
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
			err := c.Send(fmt.Sprintf("❌ Gagal memonitor posisi untuk %s: %s", symbol, err.Error()))
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		// Check if context is cancelled before sending analysis
		select {
		case <-t.ctx.Done():
			t.logger.Info("Telegram bot shutting down, skipping position monitoring message")
			return
		default:
		}

		// Format position monitoring message
		message := t.FormatPositionMonitoringMessage(position)

		// Send the position monitoring results
		err = c.Send(message, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML,
		})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send position monitoring message")
		}
	}()
}

// handleBuyList handles /buylist command - analyzes all stocks and shows buy list
func (t *TelegramBotService) handleBuyList(c telebot.Context) error {
	// Send initial message with estimation
	startTime := time.Now()
	estimatedTime := time.Duration(len(t.tradingConfig.StockList)) * 5 * time.Second // Estimate 5 seconds per stock
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

func (t *TelegramBotService) handleTextMessage(c telebot.Context) error {
	userID := c.Sender().ID

	// If user is in a conversation, handle it
	if state, ok := t.userStates[userID]; ok && state != StateIdle {
		t.handleConversation(c)
		return nil
	}

	// Existing logic for analyzing stock symbols
	symbol := strings.ToUpper(c.Text())

	// Default interval and period
	interval := "1d"
	period := "3mo"

	return t.analyzeStock(c, symbol, interval, period)
}
