package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"
)

var (
	btnGeneralAnalysis             telebot.Btn = telebot.Btn{Text: "Analisis Umum", Unique: "btn_general_analysis"}
	btnPositionAnalysis            telebot.Btn = telebot.Btn{Text: "Analisis Posisi", Unique: "btn_position_analysis"}
	btnSetPositionAlertPriceYes    telebot.Btn = telebot.Btn{Text: "✅ Ya", Unique: "btn_set_position_alert_price_yes", Data: "true"}
	btnSetPositionAlertPriceNo     telebot.Btn = telebot.Btn{Text: "❌ Tidak", Unique: "btn_set_position_alert_price_no", Data: "false"}
	btnSetPositionAlertMonitorYes  telebot.Btn = telebot.Btn{Text: "✅ Ya", Unique: "btn_set_position_alert_monitor_yes", Data: "true"}
	btnSetPositionAlertMonitorNo   telebot.Btn = telebot.Btn{Text: "❌ Tidak", Unique: "btn_set_position_alert_monitor_no", Data: "false"}
	btnStockMonitorAll             telebot.Btn = telebot.Btn{Text: "📊 Lihat Semua Saham", Unique: "btn_stock_monitor_all", Data: "all"}
	btnStockPositionMonitoring     telebot.Btn = telebot.Btn{Unique: "btn_stock_position_monitoring"}
	btnInputTimeFrameStockPosition telebot.Btn = telebot.Btn{Unique: "btn_input_time_frame_stock_position"}
	btnNotesTimeFrameStockPosition telebot.Btn = telebot.Btn{Text: "❓ Lihat Penjelasan Lengkap", Unique: "btn_input_time_frame_stock_position_notes"}
)

func (t *TelegramBotService) WithContext(handler func(ctx context.Context, c telebot.Context) error) func(c telebot.Context) error {
	return func(c telebot.Context) error {
		ctx, cancel := context.WithTimeout(t.ctx, 5*time.Minute)
		defer cancel()

		return handler(ctx, c)
	}
}

func (t *TelegramBotService) registerHandlers() {
	// Command handlers
	t.bot.Handle("/start", t.WithContext(t.handleStart))
	t.bot.Handle("/help", t.WithContext(t.handleHelp))
	t.bot.Handle("/analyze", t.WithContext(t.handleAnalyze))
	t.bot.Handle("/buylist", t.WithContext(t.handleBuyList))
	t.bot.Handle("/setposition", t.WithContext(t.handleSetPosition))
	t.bot.Handle("/cancel", t.WithContext(t.handleCancel))
	t.bot.Handle("/myposition", t.WithContext(t.handleMyPosition))

	// Callback handlers for inline buttons
	t.bot.Handle(&btnGeneralAnalysis, t.WithContext(t.handleGeneralAnalysis))
	t.bot.Handle(&btnPositionAnalysis, t.WithContext(t.handlePositionAnalysis))
	t.bot.Handle(&btnSetPositionAlertPriceYes, t.WithContext(t.handleSetPositionAlertPriceYes))
	t.bot.Handle(&btnSetPositionAlertPriceNo, t.WithContext(t.handleSetPositionAlertPriceNo))
	t.bot.Handle(&btnSetPositionAlertMonitorYes, t.WithContext(t.handleSetPositionAlertMonitorYes))
	t.bot.Handle(&btnSetPositionAlertMonitorNo, t.WithContext(t.handleSetPositionAlertMonitorNo))
	t.bot.Handle(&btnStockPositionMonitoring, t.WithContext(t.handleBtnTimeframeStockPositionMonitoring))
	t.bot.Handle(&btnNotesTimeFrameStockPosition, t.WithContext(t.handleBtnNotesTimeFrameStockPosition))
	t.bot.Handle(&btnInputTimeFrameStockPosition, t.WithContext(t.handleStockPositionMonitoring))

	// Handle incoming text messages for conversations
	t.bot.Handle(telebot.OnText, t.WithContext(t.handleConversation))

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

func (t *TelegramBotService) handleStart(ctx context.Context, c telebot.Context) error {
	message := `👋 *Halo, selamat datang di Bot Swing Trading!* 🤖  
Saya di sini untuk membantu kamu memantau saham dan mencari peluang terbaik dari pergerakan harga.

🔧 Berikut beberapa perintah yang bisa kamu gunakan:

📈 /analyze - Analisa saham pilihanmu berdasarkan strategi  
📋 /buylist - Lihat daftar saham potensial untuk dibeli  
📝 /setposition - Catat posisi saham yang sedang kamu pegang  
📊 /myposition - Lihat semua posisi yang sedang dipantau  

💡 Info & Bantuan:
🆘 /help - Lihat panduan penggunaan lengkap  
🔁 /start - Tampilkan pesan ini lagi  
❌ /cancel - Batalkan perintah yang sedang berjalan

🚀 *Siap mulai?* Coba ketik /analyze untuk memulai analisa pertamamu!`
	return c.Send(message, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (t *TelegramBotService) handleAnalyze(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID

	// If user is already in a conversation, cancel it first
	if _, inConversation := t.userStates[userID]; inConversation {
		t.handleCancel(ctx, c)
	}

	// Start a new conversation for analysis
	t.userStates[userID] = StateWaitingAnalyzeSymbol
	t.userAnalysisPositionData[userID] = &models.RequestAnalysisPositionData{} // Reuse this to store the symbol

	return c.Send("Silakan masukkan simbol saham yang ingin Anda analisis (contoh: BBCA, ANTM).")
}

func (t *TelegramBotService) handleHelp(ctx context.Context, c telebot.Context) error {
	message := `❓ *Panduan Penggunaan Bot Swing Trading* ❓

Bot ini membantu kamu memantau saham dan mencari peluang terbaik dengan analisa teknikal yang disesuaikan untuk swing trading.

Berikut daftar perintah yang bisa kamu gunakan:

🤖 *Perintah Utama:*
/start - Menampilkan pesan sambutan  
/help - Menampilkan panduan ini  
/analyze - Mulai analisa interaktif untuk saham tertentu  
/buylist - Lihat saham potensial yang sedang menarik untuk dibeli  
/setposition - Catat saham yang kamu beli agar bisa dipantau otomatis  
/myposition - Lihat semua posisi yang sedang kamu pantau  
/cancel - Batalkan perintah yang sedang berjalan

💡 *Tips Penggunaan:*
1. Gunakan /analyze untuk analisa cepat atau mendalam (bisa juga langsung kirim kode saham, misalnya: 'BBCA')  
2. Jalankan /buylist setiap pagi untuk melihat peluang baru  
3. Setelah beli saham, gunakan /setposition agar bot bisa bantu awasi harga  
4. Pantau semua posisi aktif kamu lewat /myposition


📌 Gunakan sinyal ini sebagai referensi tambahan saja, ya.  
Keputusan tetap di tangan kamu — jangan lupa *Do Your Own Research!* 🔍`
	return c.Send(message, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (t *TelegramBotService) handleSetPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	t.userStates[userID] = StateWaitingSetPositionSymbol
	reqData := &models.RequestSetPositionData{
		UserTelegram: models.ToRequestUserTelegram(c.Sender()),
	}
	t.userPositionData[userID] = reqData
	t.logger.Infof("Starting /setposition for user %d", userID)
	return c.Send("📈 Masukkan kode saham kamu (contoh: ANTM):", &telebot.SendOptions{})
}

func (t *TelegramBotService) handleCancel(ctx context.Context, c telebot.Context) error {
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

func (t *TelegramBotService) handleMyPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID

	positions, err := t.analyzer.GetStockPositionsTelegramUser(ctx, userID)
	if err != nil {
		return c.Send("❌ Terjadi kesalahan internal, silakan coba lagi.")
	}

	if len(positions) == 0 {
		return c.Send("❌ Tidak ada saham aktif yang kamu set position saat ini.")
	}

	header := "📈 *Monitor Saham Aktif Kamu*\nBerikut daftar posisi yang sedang kamu monitor.\nTekan tombol 🔍 untuk melihat analisa lengkap setiap saham.\n\n"
	c.Send(header, telebot.ModeMarkdown)

	for i, p := range positions {
		msg := t.FormatMyPositionMessage(p, i+1, len(positions))

		// Tombol analisa
		menu := &telebot.ReplyMarkup{}
		btn := menu.Data("🔍 Analisa "+p.StockCode, btnStockPositionMonitoring.Unique, p.StockCode)
		menu.Inline(menu.Row(btn))

		// Kirim pesan per saham
		if err := c.Send(msg, menu, telebot.ModeMarkdown); err != nil {
			t.logger.WithError(err).Error("send failed - list aktif posisi saham", logrus.Fields{
				"stock_code": p.StockCode,
			})
		}
	}

	// Tombol "Lihat Semua Saham"
	menu := &telebot.ReplyMarkup{}
	btnAll := menu.Data(btnStockMonitorAll.Text, btnStockMonitorAll.Unique, "all")
	menu.Inline(menu.Row(btnAll))
	return c.Send("📋 _Kamu juga bisa mengetuk tombol berikut untuk melihat semua rekap sekaligus._", menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleConversation(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID

	state, ok := t.userStates[userID]
	if !ok || state == StateIdle {
		// This should not be treated as a conversation.
		// Let the generic text handler deal with it.
		return t.handleTextMessage(ctx, c)
	}

	// Route to the correct conversation handler based on state range
	switch {
	case state >= StateWaitingGeneralAnalysisInterval && state <= StateWaitingGeneralAnalysisPeriod:
		return t.handleGeneralAnalysisConversation(ctx, c)
	case state >= StateWaitingAnalysisPositionSymbol && state <= StateWaitingAnalysisPositionPeriod:
		return t.handleAnalyzePositionConversation(ctx, c)
	case state >= StateWaitingSetPositionSymbol && state <= StateWaitingSetPositionMaxHolding:
		return t.handleSetPositionConversation(ctx, c)
	case state >= StateWaitingAnalyzeSymbol && state <= StateWaitingAnalysisType:
		return t.handleNewAnalyzeConversation(ctx, c)
	default:
		// If no specific conversation is matched, maybe it's a dangling state.
		delete(t.userStates, userID)
		return c.Send("Sepertinya Anda tidak sedang dalam percakapan aktif. Gunakan /help untuk melihat perintah yang tersedia.")
	}
}

func (t *TelegramBotService) handleGeneralAnalysis(ctx context.Context, c telebot.Context) error {
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

func (t *TelegramBotService) handleGeneralAnalysisConversation(ctx context.Context, c telebot.Context) error {
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
		return t.analyzeStock(ctx, c, data.Symbol, data.Interval, data.Period)
	}

	return nil
}

func (t *TelegramBotService) handleSetPositionConversation(ctx context.Context, c telebot.Context) error {
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
	case StateWaitingSetPositionSymbol:
		data.Symbol = strings.ToUpper(text)
		c.Send(fmt.Sprintf("👍 Oke, kode *%s* tercatat!", data.Symbol), &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		t.userStates[userID] = StateWaitingSetPositionBuyPrice
		return c.Send("💰 Berapa harga belinya ? (contoh: 150.5)", &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

	case StateWaitingSetPositionBuyPrice:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga beli tidak valid. Silakan masukkan angka (contoh: 150.5).")
		}
		data.BuyPrice = price
		t.userStates[userID] = StateWaitingSetPositionBuyDate
		return c.Send("📅 Kapan tanggal belinya? (format: YYYY-MM-DD)", &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

	case StateWaitingSetPositionBuyDate:
		_, err := time.Parse("2006-01-02", text)
		if err != nil {
			return c.Send("Format tanggal tidak valid. Silakan gunakan format YYYY-MM-DD.")
		}
		data.BuyDate = text
		t.userStates[userID] = StateWaitingSetPositionTakeProfit
		return c.Send("🎯 Target take profit-nya di harga berapa? (contoh: 180.0)", &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

	case StateWaitingSetPositionTakeProfit:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga take profit tidak valid. Silakan masukkan angka.")
		}
		data.TakeProfit = price
		t.userStates[userID] = StateWaitingSetPositionStopLoss
		return c.Send("📉 Stop loss-nya di harga berapa? (contoh: 140.0)", &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

	case StateWaitingSetPositionStopLoss:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga stop loss tidak valid. Silakan masukkan angka.")
		}
		data.StopLoss = price
		t.userStates[userID] = StateWaitingSetPositionMaxHolding
		return c.Send("⏳ Berapa maksimal hari mau di-hold? (contoh: 1) \n\n📌 *Note:* Isi angka dari *1* sampai *5* hari.", &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

	case StateWaitingSetPositionMaxHolding:
		intVal, err := strconv.Atoi(text)
		if err != nil || intVal <= 0 {
			return c.Send("Format maksimal hari hold tidak valid. Silakan masukkan angka bulat positif.")
		}
		data.MaxHolding = intVal
		t.userStates[userID] = StateWaitingSetPositionAlertPrice

		menu := &telebot.ReplyMarkup{}
		menu.Inline(
			menu.Row(btnSetPositionAlertPriceYes, btnSetPositionAlertPriceNo),
		)

		return c.Send("🚨 Aktifkan alert untuk data ini?\n\nNote: Sistem akan kirim pesan kalau harga mencapai take profit atau stop loss yang kamu tentukan.", menu)

	}
	return nil
}

func (t *TelegramBotService) handleSetPositionAlertPriceYes(ctx context.Context, c telebot.Context) error {
	if t.userStates[c.Sender().ID] != StateWaitingSetPositionAlertPrice {
		t.ResetUserState(c.Sender().ID)
		return c.Send("❌ Terjadi kesalahan internal, silakan mulai lagi dengan /setposition.")
	}
	userID := c.Sender().ID
	data := t.userPositionData[userID]
	data.AlertPrice = true
	t.bot.Edit(c.Message(), "✅ Alert harga saham diaktifkan.", &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	t.userStates[userID] = StateWaitingSetPositionAlertMonitor

	menu := &telebot.ReplyMarkup{}
	menu.Inline(
		menu.Row(btnSetPositionAlertMonitorYes, btnSetPositionAlertMonitorNo),
	)
	return c.Send("🔎 Aktifkan monitoring alert?\n\nNote: Sistem akan menganalisis posisi ini dan kirim laporan singkat: apakah masih aman, rawan, atau mendekati batas hold/SL.", menu)
}

func (t *TelegramBotService) handleBtnTimeframeStockPositionMonitoring(ctx context.Context, c telebot.Context) error {
	symbol := c.Data() // The symbol is passed as data
	menu := &telebot.ReplyMarkup{}

	msg := fmt.Sprintf("📊 Analisa Saham: *$%s*\n\nSilakan pilih strategi analisa yang paling sesuai dengan kondisimu saat ini 👇", symbol)
	btnMain := menu.Data("🔹 Main Signal", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf("%s|1d|3m", symbol))
	btnEntry := menu.Data("🔹 Entry Presisi", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf("%s|4h|1m", symbol))
	btnExit := menu.Data("🔹 Exit Presisi", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf("%s|1d|14d", symbol))
	btnNotes := menu.Data(btnNotesTimeFrameStockPosition.Text, btnNotesTimeFrameStockPosition.Unique, symbol)
	menu.Inline(
		menu.Row(btnMain, btnEntry),
		menu.Row(btnExit, btnNotes),
	)

	return c.Send(msg, menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnNotesTimeFrameStockPosition(ctx context.Context, c telebot.Context) error {
	symbol := c.Data() // The symbol is passed as data
	menu := &telebot.ReplyMarkup{}
	btnMain := menu.Data("🔹 Main Signal", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf("%s|1d|3m", symbol))
	btnEntry := menu.Data("🔹 Entry Presisi", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf("%s|4h|1m", symbol))
	btnExit := menu.Data("🔹 Exit Presisi", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf("%s|1d|14d", symbol))
	menu.Inline(
		menu.Row(btnMain, btnEntry, btnExit),
	)
	return c.Edit(t.FormatNotesTimeFrameStockPositionMessage(), menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleStockPositionMonitoring(ctx context.Context, c telebot.Context) error {
	data := c.Data() // The symbol is passed as data

	parts := strings.Split(data, "|")
	if len(parts) != 3 {
		return c.Edit("❌ Terjadi kesalahan internal, silakan coba lagi.", &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	symbol, interval, rng := parts[0], parts[1], parts[2]
	err := c.Edit(t.ShowAnalysisInProgress(symbol, interval, rng), &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	if err != nil {
		return c.Edit("❌ Terjadi kesalahan internal, silakan coba lagi.", &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}

	position, err := t.analyzer.MonitorPositionTelegramUser(ctx, &models.PositionMonitoringTelegramUserRequest{
		TelegramID: c.Sender().ID,
		Symbol:     symbol,
		Interval:   interval,
		Period:     rng,
	})
	if err != nil {
		// Send error message
		return c.Edit(fmt.Sprintf("❌ Gagal memonitor posisi untuk %s: %s", symbol, err.Error()))
	}

	// Format position monitoring message
	message := t.FormatPositionMonitoringMessage(position)

	// Send the position monitoring results
	return c.Edit(message, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	})
}

func (t *TelegramBotService) handleSetPositionAlertPriceNo(ctx context.Context, c telebot.Context) error {
	if t.userStates[c.Sender().ID] != StateWaitingSetPositionAlertPrice {
		t.ResetUserState(c.Sender().ID)
		return c.Send("❌ Terjadi kesalahan internal, silakan mulai lagi dengan /setposition.")
	}
	userID := c.Sender().ID
	data := t.userPositionData[userID]
	data.AlertPrice = false
	t.bot.Edit(c.Message(), "❌ Alert harga saham tidak diaktifkan.", &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	t.userStates[userID] = StateWaitingSetPositionAlertMonitor

	menu := &telebot.ReplyMarkup{}
	menu.Inline(
		menu.Row(btnSetPositionAlertMonitorYes, btnSetPositionAlertMonitorNo),
	)
	return c.Send("🔎 Aktifkan monitoring alert?\n\nNote: Sistem akan menganalisis posisi ini dan kirim laporan singkat: apakah masih aman, rawan, atau mendekati batas hold/SL.", menu)
}

func (t *TelegramBotService) handleSetPositionAlertMonitorYes(ctx context.Context, c telebot.Context) error {
	if t.userStates[c.Sender().ID] != StateWaitingSetPositionAlertMonitor {
		t.ResetUserState(c.Sender().ID)
		return c.Send("❌ Terjadi kesalahan internal, silakan mulai lagi dengan /setposition.")
	}
	userID := c.Sender().ID
	data := t.userPositionData[userID]
	data.AlertMonitor = true
	t.bot.Edit(c.Message(), "✅ Alert monitor diaktifkan.", &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	return t.handleSetPositionFinish(ctx, c)
}

func (t *TelegramBotService) handleSetPositionAlertMonitorNo(ctx context.Context, c telebot.Context) error {
	if t.userStates[c.Sender().ID] != StateWaitingSetPositionAlertMonitor {
		t.ResetUserState(c.Sender().ID)
		return c.Send("❌ Terjadi kesalahan internal, silakan mulai lagi dengan /setposition.")
	}
	userID := c.Sender().ID
	data := t.userPositionData[userID]
	data.AlertMonitor = false
	t.bot.Edit(c.Message(), "❌ Alert monitor tidak diaktifkan.", &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	return t.handleSetPositionFinish(ctx, c)
}

func (t *TelegramBotService) handleSetPositionFinish(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	data := t.userPositionData[userID]

	defer t.ResetUserState(userID)

	if err := t.analyzer.SetStockPosition(ctx, data); err != nil {
		return c.Send("❌ Terjadi kesalahan internal, silakan mulai lagi dengan /setposition.")
	}

	return c.Send(t.FormatResultSetPositionMessage(data), &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

}

func (t *TelegramBotService) handleAnalyzePositionConversation(ctx context.Context, c telebot.Context) error {
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
		t.executeAnalyzePosition(ctx, c, data)
		return nil // Explicitly return nil to satisfy linter
	}
	return nil
}

// handleNewAnalyzeConversation handles the first part of the new /analyze flow
func (t *TelegramBotService) handleNewAnalyzeConversation(ctx context.Context, c telebot.Context) error {
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
		btnGeneral := menu.Data(btnGeneralAnalysis.Text, btnGeneralAnalysis.Unique, symbol)
		btnPosition := menu.Data(btnPositionAnalysis.Text, btnPositionAnalysis.Unique, symbol)
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
func (t *TelegramBotService) handlePositionAnalysis(ctx context.Context, c telebot.Context) error {
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
func (t *TelegramBotService) executeAnalyzePosition(ctx context.Context, c telebot.Context, data *models.RequestAnalysisPositionData) {
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
	position, err := t.analyzer.MonitorPosition(ctx, request)
	if err != nil {
		t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to monitor position")

		// Send error message
		err := c.Send(fmt.Sprintf("❌ Gagal memonitor posisi untuk %s: %s", symbol, err.Error()))
		if err != nil {
			t.logger.WithError(err).Error("Failed to send error message")
		}
		return
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

}

// handleBuyList handles /buylist command - analyzes all stocks and shows buy list
func (t *TelegramBotService) handleBuyList(ctx context.Context, c telebot.Context) error {
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
		summary, err := t.analyzer.AnalyzeAllStocks(ctx, t.tradingConfig.StockList)
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

func (t *TelegramBotService) analyzeStock(ctx context.Context, c telebot.Context, symbol string, interval string, period string) error {
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
		analysis, err := t.analyzer.AnalyzeStock(ctx, symbol, interval, period)
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

func (t *TelegramBotService) handleTextMessage(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID

	// If user is in a conversation, handle it
	if state, ok := t.userStates[userID]; ok && state != StateIdle {
		t.handleConversation(ctx, c)
		return nil
	}

	// Existing logic for analyzing stock symbols
	symbol := strings.ToUpper(c.Text())

	// Default interval and period
	interval := "1d"
	period := "3mo"

	return t.analyzeStock(ctx, c, symbol, interval, period)
}
