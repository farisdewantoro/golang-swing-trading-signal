package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/telebot.v3"
)

var (
	btnGeneralAnalysis             telebot.Btn = telebot.Btn{Text: "Analisis Umum", Unique: "btn_general_analysis"}
	btnSetPositionAlertPriceYes    telebot.Btn = telebot.Btn{Text: "✅ Ya", Unique: "btn_set_position_alert_price_yes", Data: "true"}
	btnSetPositionAlertPriceNo     telebot.Btn = telebot.Btn{Text: "❌ Tidak", Unique: "btn_set_position_alert_price_no", Data: "false"}
	btnSetPositionAlertMonitorYes  telebot.Btn = telebot.Btn{Text: "✅ Ya", Unique: "btn_set_position_alert_monitor_yes", Data: "true"}
	btnSetPositionAlertMonitorNo   telebot.Btn = telebot.Btn{Text: "❌ Tidak", Unique: "btn_set_position_alert_monitor_no", Data: "false"}
	btnStockMonitorAll             telebot.Btn = telebot.Btn{Text: "📊 Lihat Semua Saham", Unique: "btn_stock_monitor_all", Data: "all"}
	btnStockPositionMonitoring     telebot.Btn = telebot.Btn{Unique: "btn_stock_position_monitoring"}
	btnInputTimeFrameStockPosition telebot.Btn = telebot.Btn{Unique: "btn_input_time_frame_stock_position"}
	btnInputTimeFrameStockAnalysis telebot.Btn = telebot.Btn{Unique: "btn_input_time_frame_stock_analysis"}
	btnNotesTimeFrameStockPosition telebot.Btn = telebot.Btn{Text: "❓ Detail", Unique: "btn_input_time_frame_stock_position_notes"}
	btnNotesTimeFrameStockAnalysis telebot.Btn = telebot.Btn{Text: "❓ Detail", Unique: "btn_input_time_frame_stock_analysis_notes"}
	btnManageStockPosition         telebot.Btn = telebot.Btn{Text: "⚙️ Kelola", Unique: "btn_manage_stock_position"}
	btnListStockPosition           telebot.Btn = telebot.Btn{Unique: "btn_list_stock_position"}
	btnBackStockPosition           telebot.Btn = telebot.Btn{Text: "🔙 Kembali", Unique: "btn_back_stock_position"}
	btnNewsStockPosition           telebot.Btn = telebot.Btn{Text: "📰 Berita", Unique: "btn_news_stock_position"}
	btnBackActionStockPosition     telebot.Btn = telebot.Btn{Text: "🔙 Kembali", Unique: "btn_back_action_stock_position"}
	btnBackDetailStockPosition     telebot.Btn = telebot.Btn{Text: "🔙 Kembali", Unique: "btn_back_detail_stock_position"}
	btnDeleteMessage               telebot.Btn = telebot.Btn{Text: "🗑️ Hapus Pesan", Unique: "btn_delete_message"}
	btnBackStockAnalysis           telebot.Btn = telebot.Btn{Text: "🔙 Kembali", Unique: "btn_back_stock_analysis"}
	btnDeleteStockPosition         telebot.Btn = telebot.Btn{Text: "🗑️ Hapus Posisi", Unique: "btn_delete_stock_position"}
	btnUpdateAlertPrice            telebot.Btn = telebot.Btn{Unique: "btn_update_alert_price"}
	btnUpdateAlertMonitor          telebot.Btn = telebot.Btn{Unique: "btn_update_alert_monitor"}
	btnExitStockPosition           telebot.Btn = telebot.Btn{Unique: "btn_exit_stock_position"}
	btnSaveExitPosition            telebot.Btn = telebot.Btn{Text: "💾 Simpan", Unique: "btn_save_exit_position"}
	btnCancelGeneral               telebot.Btn = telebot.Btn{Text: "❌ Batal", Unique: "btn_cancel_general"}

	dataInputTimeFrameMain  string = "%s|1d|3m"
	dataInputTimeFrameEntry string = "%s|4h|1m"
	dataInputTimeFrameExit  string = "%s|1d|14d"

	commonMessageInternalError string = "❌ Terjadi kesalahan internal, silakan coba lagi."
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
	t.bot.Handle("/analyze", t.WithContext(t.handleAnalyze), t.IsOnConversationMiddleware())
	t.bot.Handle("/buylist", t.WithContext(t.handleBuyList), t.IsOnConversationMiddleware())
	t.bot.Handle("/setposition", t.WithContext(t.handleSetPosition), t.IsOnConversationMiddleware())
	t.bot.Handle("/cancel", t.handleCancel)
	t.bot.Handle("/myposition", t.WithContext(t.handleMyPosition), t.IsOnConversationMiddleware())

	// Callback handlers for inline buttons
	t.bot.Handle(&btnGeneralAnalysis, t.WithContext(t.handleGeneralAnalysis))
	t.bot.Handle(&btnSetPositionAlertPriceYes, t.WithContext(t.handleSetPositionAlertPriceYes))
	t.bot.Handle(&btnSetPositionAlertPriceNo, t.WithContext(t.handleSetPositionAlertPriceNo))
	t.bot.Handle(&btnSetPositionAlertMonitorYes, t.WithContext(t.handleSetPositionAlertMonitorYes))
	t.bot.Handle(&btnSetPositionAlertMonitorNo, t.WithContext(t.handleSetPositionAlertMonitorNo))
	t.bot.Handle(&btnStockPositionMonitoring, t.WithContext(t.handleBtnTimeframeStockPositionMonitoring))
	t.bot.Handle(&btnNotesTimeFrameStockPosition, t.WithContext(t.handleBtnNotesTimeFrameStockPosition))
	t.bot.Handle(&btnInputTimeFrameStockPosition, t.WithContext(t.handleStockPositionMonitoring))
	t.bot.Handle(&btnInputTimeFrameStockAnalysis, t.WithContext(t.handleBtnGeneralAnalysis))
	t.bot.Handle(&btnNotesTimeFrameStockAnalysis, t.WithContext(t.handleBtnNotesTimeFrameStockAnalysis))
	t.bot.Handle(&btnManageStockPosition, t.WithContext(t.handleBtnManageStockPosition))
	t.bot.Handle(&btnListStockPosition, t.WithContext(t.handleBtnListStockPosition))
	t.bot.Handle(&btnBackStockPosition, t.WithContext(t.handleBtnBackStockPosition))
	t.bot.Handle(&btnBackActionStockPosition, t.WithContext(t.handleBtnBackActionStockPosition))
	t.bot.Handle(&btnBackDetailStockPosition, t.WithContext(t.handleBtnBackDetailStockPosition))
	t.bot.Handle(&btnDeleteMessage, t.WithContext(t.handleBtnDeleteMessage))
	t.bot.Handle(&btnBackStockAnalysis, t.WithContext(t.handleBtnBackStockAnalysis))
	t.bot.Handle(&btnDeleteStockPosition, t.WithContext(t.handleBtnDeleteStockPosition))
	t.bot.Handle(&btnUpdateAlertPrice, t.WithContext(t.handleBtnUpdateAlertPrice))
	t.bot.Handle(&btnUpdateAlertMonitor, t.WithContext(t.handleBtnUpdateAlertMonitor))
	t.bot.Handle(&btnExitStockPosition, t.WithContext(t.handleBtnExitStockPosition))
	t.bot.Handle(&btnCancelGeneral, t.WithContext(t.handleBtnCancel))
	t.bot.Handle(&btnSaveExitPosition, t.WithContext(t.handleBtnSaveExitPosition))

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
	go t.analyzer.CreateTelegramUserIfNotExist(ctx, models.ToRequestUserTelegram(c.Sender()))
	return c.Send(message, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (t *TelegramBotService) handleAnalyze(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID

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

func (t *TelegramBotService) handleCancel(c telebot.Context) error {
	userID := c.Sender().ID

	// Check if user is in any conversation state
	if state, ok := t.userStates[userID]; ok && state != StateIdle {
		// Reset user state
		t.ResetUserState(userID)
		return c.Send("✅ Percakapan dibatalkan.")
	}

	return c.Send("🤷‍♀️ Tidak ada percakapan aktif yang bisa dibatalkan.")
}

func (t *TelegramBotService) handleBtnCancel(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID

	// Check if user is in any conversation state
	if state, ok := t.userStates[userID]; ok && state != StateIdle {
		// Reset user state
		t.ResetUserState(userID)
		return c.Edit("✅ Percakapan dibatalkan.")
	}

	return c.Send("🤷‍♀️ Tidak ada percakapan aktif yang bisa dibatalkan.")
}

func (t *TelegramBotService) handleBtnDeleteMessage(ctx context.Context, c telebot.Context) error {
	c.Edit("✅ Pesan akan dihapus....")
	time.Sleep(1 * time.Second)
	return c.Delete()
}

func (t *TelegramBotService) handleMyPosition(ctx context.Context, c telebot.Context) error {
	return t.handleMyPositionWithEditMessage(ctx, c, false)
}

func (t *TelegramBotService) handleMyPositionWithEditMessage(ctx context.Context, c telebot.Context, isEditMessage bool) error {
	userID := c.Sender().ID

	positions, err := t.analyzer.GetStockPositionsTelegramUser(ctx, userID)
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	if len(positions) == 0 {
		return c.Send("❌ Tidak ada saham aktif yang kamu set position saat ini.")
	}

	header := `📊 Posisi Saham yang Kamu Pantau

Berikut adalah daftar saham yang sedang kamu monitor.

Tekan salah satu saham di bawah ini untuk melihat detail lengkap posisinya — termasuk harga beli, target jual, stop loss, umur posisi, status alert, dan monitoring.

Kamu juga bisa mengatur ulang atau menghapus posisi setelah membukanya.`

	menu := &telebot.ReplyMarkup{}
	rows := []telebot.Row{}

	for _, p := range positions {
		btn := menu.Data(fmt.Sprintf("➤ %s ($%.2f)", p.StockCode, p.BuyPrice), btnListStockPosition.Unique, fmt.Sprintf("%d", p.ID))
		rows = append(rows, menu.Row(btn))
	}
	btnDeleteMessage := menu.Data(btnDeleteMessage.Text, btnDeleteMessage.Unique)
	rows = append(rows, menu.Row(btnDeleteMessage))

	menu.Inline(rows...)

	if isEditMessage {
		return c.Edit(header, menu, telebot.ModeMarkdown)
	}

	return c.Send(header, menu, telebot.ModeMarkdown)

}

func (t *TelegramBotService) handleBtnListStockPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	id, err := strconv.Atoi(c.Data())
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	positions, err := t.analyzer.GetStockPosition(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{userID},
		IsActive:    true,
		IDs:         []uint{uint(id)},
	})
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	if len(positions) == 0 {
		return c.Send("❌ Tidak ada posisi yang ditemukan.")
	}

	// Tombol analisa
	menu := &telebot.ReplyMarkup{}
	btn := menu.Data("🔍 Analisa", btnStockPositionMonitoring.Unique, positions[0].StockCode)
	btnManage := menu.Data(btnManageStockPosition.Text, btnManageStockPosition.Unique, strconv.FormatUint(uint64(positions[0].ID), 10))
	btnBack := menu.Data(btnBackStockPosition.Text, btnBackStockPosition.Unique)
	btnNews := menu.Data(btnNewsStockPosition.Text, btnNewsStockPosition.Unique, positions[0].StockCode)

	menu.Inline(
		menu.Row(btn, btnManage),
		menu.Row(btnNews, btnBack),
	)

	return c.Edit(t.FormatMyStockPositionMessage(positions[0]), menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnBackDetailStockPosition(ctx context.Context, c telebot.Context) error {
	symbol := c.Data()
	return t.handleBtnBackDetailStockPositionWithParam(ctx, c, &symbol, nil)
}

func (t *TelegramBotService) handleBtnBackDetailStockPositionWithParam(ctx context.Context, c telebot.Context, symbol *string, stockPosisitionID *uint) error {
	senderID := c.Sender().ID

	param := models.StockPositionQueryParam{
		TelegramIDs: []int64{senderID},
		IsActive:    true,
	}
	if symbol != nil {
		param.StockCodes = []string{*symbol}
	}
	if stockPosisitionID != nil {
		param.IDs = []uint{*stockPosisitionID}
	}
	positions, err := t.analyzer.GetStockPosition(ctx, param)
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	if len(positions) == 0 {
		return c.Send("❌ Tidak ada posisi yang ditemukan.")
	}

	// Tombol analisa
	menu := &telebot.ReplyMarkup{}
	btn := menu.Data("🔍 Analisa", btnStockPositionMonitoring.Unique, positions[0].StockCode)
	btnManage := menu.Data(btnManageStockPosition.Text, btnManageStockPosition.Unique, strconv.FormatUint(uint64(positions[0].ID), 10))
	btnBack := menu.Data(btnBackStockPosition.Text, btnBackStockPosition.Unique)
	btnNews := menu.Data(btnNewsStockPosition.Text, btnNewsStockPosition.Unique, positions[0].StockCode)

	menu.Inline(
		menu.Row(btn, btnManage),
		menu.Row(btnNews, btnBack),
	)

	return c.Edit(t.FormatMyStockPositionMessage(positions[0]), menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnBackStockPosition(ctx context.Context, c telebot.Context) error {
	return t.handleMyPositionWithEditMessage(ctx, c, true)
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
	case state >= StateWaitingSetPositionSymbol && state <= StateWaitingSetPositionAlertMonitor:
		return t.handleSetPositionConversation(ctx, c)
	case state >= StateWaitingAnalyzeSymbol && state <= StateWaitingAnalysisType:
		return t.handleGeneralAnalysis(ctx, c)
	case state >= StateWaitingExitPositionInputExitPrice && state <= StateWaitingExitPositionConfirm:
		return t.handleExitPositionConversation(ctx, c)
	default:
		// If no specific conversation is matched, maybe it's a dangling state.
		t.ResetUserState(userID)
		return c.Send("Sepertinya Anda tidak sedang dalam percakapan aktif. Gunakan /help untuk melihat perintah yang tersedia.")
	}
}

func (t *TelegramBotService) handleGeneralAnalysis(ctx context.Context, c telebot.Context) error {
	symbol := c.Text()
	return t.handleGeneralAnalysisWithParam(ctx, c, symbol, false)
}
func (t *TelegramBotService) handleGeneralAnalysisWithParam(ctx context.Context, c telebot.Context, symbol string, isEdit bool) error {
	userID := c.Sender().ID

	t.ResetUserState(userID)

	menu := &telebot.ReplyMarkup{}

	msg := fmt.Sprintf("📊 Analisa Saham: *$%s*\n\nSilakan pilih strategi analisa yang paling sesuai dengan kondisimu saat ini 👇", symbol)
	btnMain := menu.Data("🔹 Main Signal", btnInputTimeFrameStockAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameMain, symbol))
	btnEntry := menu.Data("🔹 Entry Presisi", btnInputTimeFrameStockAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameEntry, symbol))
	btnExit := menu.Data("🔹 Exit Presisi", btnInputTimeFrameStockAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameExit, symbol))
	btnNotes := menu.Data(btnNotesTimeFrameStockAnalysis.Text, btnNotesTimeFrameStockAnalysis.Unique, symbol)
	btnDelete := menu.Data(btnDeleteMessage.Text, btnDeleteMessage.Unique, symbol)
	menu.Inline(
		menu.Row(btnMain, btnEntry),
		menu.Row(btnExit, btnNotes),
		menu.Row(btnDelete),
	)

	if isEdit {
		return c.Edit(msg, menu, telebot.ModeMarkdown)
	}
	return c.Send(msg, menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnGeneralAnalysis(ctx context.Context, c telebot.Context) error {

	data := c.Data()

	parts := strings.Split(data, "|")
	if len(parts) != 3 {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	symbol, interval, rng := parts[0], parts[1], parts[2]

	stopChan := make(chan struct{})

	// Mulai loading animasi
	msg := t.showLoadingFlowAnalysis(c, stopChan)

	go func() {
		newCtx, cancel := context.WithTimeout(t.ctx, t.config.TimeoutDuration)
		defer cancel()
		analysis, err := t.analyzer.AnalyzeStock(newCtx, symbol, interval, rng)

		if err != nil {
			close(stopChan)
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to analyze stock")

			// Send error message
			err := c.Send(fmt.Sprintf("❌ Failed to analyze %s: %s", symbol, err.Error()))
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		// Format analysis message
		analysisMessage := t.FormatAnalysisMessage(analysis)

		// Stop animasi loading
		close(stopChan)

		// Ganti pesan loading dengan hasil analisa
		_, err = t.bot.Edit(msg, analysisMessage, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML,
		})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send analysis message")
		}

	}()

	return nil
}

func (t *TelegramBotService) handleBtnNotesTimeFrameStockAnalysis(ctx context.Context, c telebot.Context) error {
	symbol := c.Data() // The symbol is passed as data
	menu := &telebot.ReplyMarkup{}
	btnMain := menu.Data("🔹 Main Signal", btnInputTimeFrameStockAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameMain, symbol))
	btnEntry := menu.Data("🔹 Entry Presisi", btnInputTimeFrameStockAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameEntry, symbol))
	btnExit := menu.Data("🔹 Exit Presisi", btnInputTimeFrameStockAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameExit, symbol))
	btnBack := menu.Data(btnBackStockAnalysis.Text, btnBackStockAnalysis.Unique, symbol)
	menu.Inline(
		menu.Row(btnMain, btnEntry, btnExit),
		menu.Row(btnBack),
	)
	return c.Edit(t.FormatNotesTimeFrameStockMessage(), menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnBackStockAnalysis(ctx context.Context, c telebot.Context) error {
	return t.handleGeneralAnalysisWithParam(ctx, c, c.Data(), true)
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

	case StateWaitingSetPositionAlertPrice, StateWaitingSetPositionAlertMonitor:
		return c.Send("👆 Silakan pilih salah satu opsi di atas, atau kirim /cancel untuk membatalkan.")
	}
	return nil
}

func (t *TelegramBotService) handleExitPositionConversation(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	text := c.Text()
	state := t.userStates[userID] // We already know the state exists

	data, data_ok := t.userExitPositionData[userID]
	if !data_ok {
		// Should not happen, but as a safeguard
		delete(t.userStates, userID)
		return c.Send("Terjadi kesalahan internal (data posisi tidak ditemukan).")
	}

	switch state {
	case StateWaitingExitPositionInputExitPrice:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga jual tidak valid. Silakan masukkan angka (contoh: 150.5).")
		}
		data.ExitPrice = price
		err = c.Send(fmt.Sprintf(`
🚀 Exit posisi saham *%s (2/2)*

📅 Kapan tanggal jualnya? (contoh: 2025-05-18)`, data.Symbol), &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		if err != nil {
			return c.Send(commonMessageInternalError)
		}
		t.userStates[userID] = StateWaitingExitPositionInputExitDate
		return nil
	case StateWaitingExitPositionInputExitDate:
		date, err := time.Parse("2006-01-02", text)
		if err != nil {
			return c.Send("Format tanggal tidak valid. Silakan gunakan format YYYY-MM-DD.")
		}
		data.ExitDate = date
		t.userStates[userID] = StateWaitingExitPositionConfirm
		msg := fmt.Sprintf(`
📌 Mohon cek kembali data yang kamu masukkan:

• Kode Saham   : %s 
• Harga Exit   : %.2f  
• Tanggal Exit : %s  
		`, data.Symbol, data.ExitPrice, data.ExitDate.Format("2006-01-02"))
		menu := &telebot.ReplyMarkup{}
		btnSave := menu.Data(btnSaveExitPosition.Text, btnSaveExitPosition.Unique)
		btnCancel := menu.Data(btnCancelGeneral.Text, btnCancelGeneral.Unique)
		menu.Inline(
			menu.Row(btnSave, btnCancel),
		)
		return c.Send(msg, menu, telebot.ModeMarkdown)
	case StateWaitingExitPositionConfirm:
		return c.Send("👆 Silakan pilih salah satu opsi di atas, atau kirim /cancel untuk membatalkan.")
	}
	return nil
}

func (t *TelegramBotService) handleBtnSaveExitPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	data := t.userExitPositionData[userID]
	if data == nil {
		return c.Send(commonMessageInternalError)
	}
	if data.ExitPrice == 0 || data.ExitDate.IsZero() {
		return c.Send("❌ Data tidak lengkap, silakan masukkan harga exit dan tanggal exit.")
	}

	stopChan := make(chan struct{})
	msg := t.showLoadingGeneral(c, stopChan)

	go func() {
		newCtx, cancel := context.WithTimeout(t.ctx, t.config.TimeoutDuration)
		defer cancel()

		if err := t.analyzer.UpdateStockPositionTelegramUser(newCtx, userID, data.StockPositionID, &models.StockPositionUpdateRequest{
			ExitPrice: utils.ToPointer(data.ExitPrice),
			ExitDate:  utils.ToPointer(data.ExitDate),
			IsActive:  utils.ToPointer(false),
		}); err != nil {
			close(stopChan)
			if err := c.Send(commonMessageInternalError); err != nil {
				t.logger.WithError(err).Error("Failed to update stock position")
			}
		}
		time.Sleep(1 * time.Second)
		close(stopChan)
		t.bot.Edit(msg, "✅ Exit posisi berhasil disimpan.")
		time.Sleep(1 * time.Second)
		t.handleMyPositionWithEditMessage(newCtx, c, true)

	}()
	t.ResetUserState(userID)

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

	msg := fmt.Sprintf("📊 Analisa Posisi Saham: *$%s*\n\nSilahkan pilih strategi analisa yang paling relevan dengan kondisi posisi kamu saat ini 👇", symbol)
	btnMain := menu.Data("🔹 Main Signal", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf(dataInputTimeFrameMain, symbol))
	btnEntry := menu.Data("🔹 Entry Presisi", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf(dataInputTimeFrameEntry, symbol))
	btnExit := menu.Data("🔹 Exit Presisi", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf(dataInputTimeFrameExit, symbol))
	btnNotes := menu.Data(btnNotesTimeFrameStockPosition.Text, btnNotesTimeFrameStockPosition.Unique, symbol)
	btnBack := menu.Data(btnBackDetailStockPosition.Text, btnBackDetailStockPosition.Unique, symbol)
	menu.Inline(
		menu.Row(btnMain, btnEntry),
		menu.Row(btnExit, btnNotes),
		menu.Row(btnBack),
	)

	return c.Edit(msg, menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnNotesTimeFrameStockPosition(ctx context.Context, c telebot.Context) error {
	symbol := c.Data() // The symbol is passed as data
	menu := &telebot.ReplyMarkup{}
	btnMain := menu.Data("🔹 Main Signal", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf(dataInputTimeFrameMain, symbol))
	btnEntry := menu.Data("🔹 Entry Presisi", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf(dataInputTimeFrameEntry, symbol))
	btnExit := menu.Data("🔹 Exit Presisi", btnInputTimeFrameStockPosition.Unique, fmt.Sprintf(dataInputTimeFrameExit, symbol))
	btnBack := menu.Data("🔙 Kembali", btnStockPositionMonitoring.Unique, symbol)
	menu.Inline(
		menu.Row(btnMain, btnEntry, btnExit),
		menu.Row(btnBack),
	)
	return c.Edit(t.FormatNotesTimeFrameStockMessage(), menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnManageStockPosition(ctx context.Context, c telebot.Context) error {
	stockPositionID := c.Data()

	stockPositionIDInt, err := strconv.Atoi(stockPositionID)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal mengambil posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	stockPosition, err := t.analyzer.GetStockPosition(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{c.Sender().ID},
		IDs:         []uint{uint(stockPositionIDInt)},
	})
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal mengambil posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	if len(stockPosition) == 0 {
		return c.Edit(fmt.Sprintf("❌ Tidak ditemukan posisi untuk %s", stockPositionID))
	}

	stockCode := stockPosition[0].StockCode
	msgText := fmt.Sprintf(
		"⚙️ Kelola Posisi Saham *%s*\n\nPilih aksi yang ingin kamu lakukan terhadap saham ini 👇\n\n",
		stockCode,
	)

	// Buat keyboard
	keyboard := &telebot.ReplyMarkup{}

	// Tombol aksi utama
	btnExit := keyboard.Data("🚪 Exit dari Posisi", btnExitStockPosition.Unique, fmt.Sprintf("%s|%d", stockCode, stockPositionIDInt))
	btnDelete := keyboard.Data(btnDeleteStockPosition.Text, btnDeleteStockPosition.Unique, stockPositionID)

	// Toggle Alert
	var btnAlert telebot.Btn
	isAlertOn := stockPosition[0].PriceAlert != nil && *stockPosition[0].PriceAlert
	if isAlertOn {
		btnAlert = keyboard.Data("🔕 Nonaktifkan Alert", btnUpdateAlertPrice.Unique, fmt.Sprintf("%s|false", stockPositionID))
	} else {
		btnAlert = keyboard.Data("🔔 Aktifkan Alert", btnUpdateAlertPrice.Unique, fmt.Sprintf("%s|true", stockPositionID))
	}

	// Toggle Monitoring
	var btnMonitor telebot.Btn
	isMonitoringOn := stockPosition[0].MonitorPosition != nil && *stockPosition[0].MonitorPosition
	if isMonitoringOn {
		btnMonitor = keyboard.Data("❌ Nonaktifkan Monitoring", btnUpdateAlertMonitor.Unique, fmt.Sprintf("%s|false", stockPositionID))
	} else {
		btnMonitor = keyboard.Data("📡 Aktifkan Monitoring", btnUpdateAlertMonitor.Unique, fmt.Sprintf("%s|true", stockPositionID))
	}

	// Tombol kembali
	btnBack := keyboard.Data(btnBackActionStockPosition.Text, btnBackActionStockPosition.Unique, stockPositionID)

	// Susun tombol: satu per baris
	keyboard.Inline(
		keyboard.Row(btnExit),
		keyboard.Row(btnDelete),
		keyboard.Row(btnAlert),
		keyboard.Row(btnMonitor),
		keyboard.Row(btnBack),
	)

	return c.Edit(msgText, keyboard, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnDeleteStockPosition(ctx context.Context, c telebot.Context) error {
	c.Respond(&telebot.CallbackResponse{
		Text:      "🔄 Menghapus....",
		ShowAlert: false,
	})

	stockPositionID := c.Data()

	stockPositionIDInt, err := strconv.Atoi(stockPositionID)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal mengambil posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	if err = t.analyzer.DeleteStockPositionTelegramUser(ctx, c.Sender().ID, uint(stockPositionIDInt)); err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal menghapus posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	return c.Edit("✅ Posisi saham berhasil dihapus.")
}

func (t *TelegramBotService) handleBtnUpdateAlertPrice(ctx context.Context, c telebot.Context) error {
	data := c.Data()
	parts := strings.Split(data, "|")
	if len(parts) != 2 {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	stockPositionID := parts[0]
	isAlertOn := parts[1]

	stockPositionIDInt, err := strconv.Atoi(stockPositionID)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal mengambil posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	isAlertOnBool, err := strconv.ParseBool(isAlertOn)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal parse data untuk %s: %s", stockPositionID, err.Error()))
	}

	if err = t.analyzer.UpdateStockPositionTelegramUser(ctx, c.Sender().ID, uint(stockPositionIDInt), &models.StockPositionUpdateRequest{
		PriceAlert: &isAlertOnBool,
	}); err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal update status alert untuk %s: %s", stockPositionID, err.Error()))
	}

	isActive := "✅ Alert Price berhasil diaktifkan."
	if !isAlertOnBool {
		isActive = "❌ Alert Price berhasil dinonaktifkan."
	}

	c.Edit(isActive)
	time.Sleep(200 * time.Millisecond)
	return t.handleBtnBackDetailStockPositionWithParam(ctx, c, nil, utils.ToPointer(uint(stockPositionIDInt)))
}

func (t *TelegramBotService) handleBtnUpdateAlertMonitor(ctx context.Context, c telebot.Context) error {
	data := c.Data()
	parts := strings.Split(data, "|")
	if len(parts) != 2 {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	stockPositionID := parts[0]
	isMonitorOn := parts[1]

	stockPositionIDInt, err := strconv.Atoi(stockPositionID)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal mengambil posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	isMonitorOnBool, err := strconv.ParseBool(isMonitorOn)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal parse data untuk %s: %s", stockPositionID, err.Error()))
	}

	if err = t.analyzer.UpdateStockPositionTelegramUser(ctx, c.Sender().ID, uint(stockPositionIDInt), &models.StockPositionUpdateRequest{
		MonitorPosition: &isMonitorOnBool,
	}); err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal update status monitoring untuk %s: %s", stockPositionID, err.Error()))
	}

	isActive := "✅ Stock Monitoring berhasil diaktifkan."
	if !isMonitorOnBool {
		isActive = "❌ Stock Monitoring berhasil dinonaktifkan."
	}

	c.Edit(isActive)
	time.Sleep(200 * time.Millisecond)
	return t.handleBtnBackDetailStockPositionWithParam(ctx, c, nil, utils.ToPointer(uint(stockPositionIDInt)))

}

func (t *TelegramBotService) handleBtnBackActionStockPosition(ctx context.Context, c telebot.Context) error {
	return t.handleBtnListStockPosition(ctx, c)
}

func (t *TelegramBotService) handleBtnExitStockPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	data := c.Data()

	userState := t.userStates[userID]
	if userState != StateIdle {
		t.ResetUserState(userID)
		return c.Send(commonMessageInternalError)
	}

	parts := strings.Split(data, "|")
	if len(parts) != 2 {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}

	stockPositionIDInt, err := strconv.Atoi(parts[1])
	if err != nil {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}

	msg := fmt.Sprintf(`🚀 Exit posisi saham *%s (1/2)*

Masukkan *harga jual* kamu di bawah ini (dalam angka).  
Contoh: 175.00

`, parts[0])

	err = c.Edit(msg, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
	if err != nil {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	t.userStates[userID] = StateWaitingExitPositionInputExitPrice
	t.userExitPositionData[userID] = &models.RequestExitPositionData{
		Symbol:          parts[0],
		StockPositionID: uint(stockPositionIDInt),
	}

	return nil
}

func (t *TelegramBotService) handleStockPositionMonitoring(ctx context.Context, c telebot.Context) error {
	data := c.Data() // The symbol is passed as data

	parts := strings.Split(data, "|")
	if len(parts) != 3 {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	symbol, interval, rng := parts[0], parts[1], parts[2]

	stopChan := make(chan struct{})

	// Mulai loading animasi
	msg := t.showLoadingFlowAnalysis(c, stopChan)

	go func() {
		newCtx, cancel := context.WithTimeout(t.ctx, t.config.TimeoutDuration)
		defer cancel()

		position, err := t.analyzer.MonitorPositionTelegramUser(newCtx, &models.PositionMonitoringTelegramUserRequest{
			TelegramID: c.Sender().ID,
			Symbol:     symbol,
			Interval:   interval,
			Period:     rng,
		})
		if err != nil {
			close(stopChan)
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to analyze stock")
			t.bot.Edit(msg, commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
			return
		}

		close(stopChan)

		// Format position monitoring message
		message := t.FormatPositionMonitoringMessage(position)

		// Send the position monitoring results
		t.bot.Edit(msg, message, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML,
		})

	}()

	return nil
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
	// Perform analysis
	analysis, err := t.analyzer.AnalyzeStock(ctx, symbol, interval, period)
	if err != nil {
		t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to analyze stock")

		// Send error message
		err := c.Send(fmt.Sprintf("❌ Failed to analyze %s: %s", symbol, err.Error()))
		if err != nil {
			t.logger.WithError(err).Error("Failed to send error message")
		}
		return err
	}

	// Format analysis message
	analysisMessage := t.FormatAnalysisMessage(analysis)

	// Send the analysis results
	err = c.Send(analysisMessage, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	})
	if err != nil {
		t.logger.WithError(err).Error("Failed to send analysis message")
		return err
	}

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
