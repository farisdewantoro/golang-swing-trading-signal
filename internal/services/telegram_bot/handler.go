package telegram_bot

import (
	"context"
	"golang-swing-trading-signal/internal/models"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) registerHandlers() {
	// Command handlers
	t.bot.Handle("/start", t.WithContext(t.handleStart))
	t.bot.Handle("/help", t.WithContext(t.handleHelp))
	t.bot.Handle("/analyze", t.WithContext(t.handleAnalyze), t.IsOnConversationMiddleware())
	t.bot.Handle("/buylist", t.WithContext(t.handleBuyList), t.IsOnConversationMiddleware())
	t.bot.Handle("/setposition", t.WithContext(t.handleSetPosition), t.IsOnConversationMiddleware())
	t.bot.Handle("/cancel", t.handleCancel)
	t.bot.Handle("/myposition", t.WithContext(t.handleMyPosition), t.IsOnConversationMiddleware())
	t.bot.Handle("/news", t.WithContext(t.handleNews), t.IsOnConversationMiddleware())
	t.bot.Handle("/report", t.WithContext(t.handleReport), t.IsOnConversationMiddleware())
	t.bot.Handle("/scheduler", t.WithContext(t.handleScheduler))

	// Inline button handlers

	// Set position handlers
	t.bot.Handle(&btnSetPositionAlertPriceYes, t.WithContext(t.handleBtnSetPositionAlertPriceYes))
	t.bot.Handle(&btnSetPositionAlertPriceNo, t.WithContext(t.handleBtnSetPositionAlertPriceNo))
	t.bot.Handle(&btnSetPositionAlertMonitorYes, t.WithContext(t.handleSetPositionAlertMonitorYes))
	t.bot.Handle(&btnSetPositionAlertMonitorNo, t.WithContext(t.handleSetPositionAlertMonitorNo))

	t.bot.Handle(&btnStockPositionMonitoring, t.WithContext(t.handleBtnTimeframeStockPositionMonitoring))
	t.bot.Handle(&btnManageStockPosition, t.WithContext(t.handleBtnManageStockPosition))
	t.bot.Handle(&btnToDetailStockPosition, t.WithContext(t.handleBtnToDetailStockPosition))
	t.bot.Handle(&btnBackStockPosition, t.WithContext(t.handleBtnBackStockPosition))
	t.bot.Handle(&btnBackActionStockPosition, t.WithContext(t.handleBtnBackActionStockPosition))
	t.bot.Handle(&btnBackDetailStockPosition, t.WithContext(t.handleBtnBackDetailStockPosition))
	t.bot.Handle(&btnDeleteMessage, t.WithContext(t.handleBtnDeleteMessage))

	t.bot.Handle(&btnDeleteStockPosition, t.WithContext(t.handleBtnDeleteStockPosition))
	t.bot.Handle(&btnUpdateAlertPrice, t.WithContext(t.handleBtnUpdateAlertPrice))
	t.bot.Handle(&btnUpdateAlertMonitor, t.WithContext(t.handleBtnUpdateAlertMonitor))
	t.bot.Handle(&btnExitStockPosition, t.WithContext(t.handleBtnExitStockPosition))
	t.bot.Handle(&btnCancelGeneral, t.WithContext(t.handleBtnCancel))
	t.bot.Handle(&btnSaveExitPosition, t.WithContext(t.handleBtnSaveExitPosition))
	t.bot.Handle(&btnCancelBuyListAnalysis, t.WithContext(t.handleBtnCancelBuyListAnalysis))
	t.bot.Handle(&btnActionNewsFind, t.WithContext(t.handleBtnActionNewsFind))
	t.bot.Handle(&btnNewsConfirmSendSummary, t.WithContext(t.handleBtnNewsConfirmSendSummary))
	t.bot.Handle(&btnNewsStockPosition, t.WithContext(t.handleBtnNewsStockPosition))
	t.bot.Handle(&btnActionTopNews, t.WithContext(t.handleBtnActionTopNews))
	t.bot.Handle(&btnAdjustTargetPosition, t.WithContext(t.handleBtnAdjustTargetPosition))
	t.bot.Handle(&btnAdjustTargetPositionConfirm, t.WithContext(t.handleBtnAdjustTargetPositionConfirm))
	t.bot.Handle(&btnDetailJob, t.WithContext(t.handleBtnDetailJob))
	t.bot.Handle(&btnActionBackToJobList, t.WithContext(t.handleBtnActionBackToJobList))
	t.bot.Handle(&btnActionRunJob, t.WithContext(t.handleBtnActionRunJob))
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
📰 /news - Lihat berita terkini, alert berita penting saham, ringkasan berita
💰 /report Melihat ringkasan hasil trading kamu berdasarkan posisi yang sudah kamu entry dan exit.

💡 Info & Bantuan:
🆘 /help - Lihat panduan penggunaan lengkap  
🔁 /start - Tampilkan pesan ini lagi  
❌ /cancel - Batalkan perintah yang sedang berjalan

🚀 *Siap mulai?* Coba ketik /analyze untuk memulai analisa pertamamu!`
	go t.analyzer.CreateTelegramUserIfNotExist(ctx, models.ToRequestUserTelegram(c.Sender()))
	return c.Send(message, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
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
/news - Lihat berita terkini, alert berita penting saham, ringkasan berita
/cancel - Batalkan perintah yang sedang berjalan
/report - Melihat ringkasan hasil trading kamu berdasarkan posisi yang sudah kamu entry dan exit.

💡 *Tips Penggunaan:*
1. Gunakan /analyze untuk analisa cepat atau mendalam (bisa juga langsung kirim kode saham, misalnya: 'BBCA')  
2. Jalankan /buylist setiap pagi untuk melihat peluang baru  
3. Setelah beli saham, gunakan /setposition agar bot bisa bantu awasi harga  
4. Pantau semua posisi aktif kamu lewat /myposition


📌 Gunakan sinyal ini sebagai referensi tambahan saja, ya.  
Keputusan tetap di tangan kamu — jangan lupa *Do Your Own Research!* 🔍`
	return c.Send(message, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
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
	case state >= StateWaitingNewsFindSymbol && state <= StateWaitingNewsFindSendSummaryConfirmation:
		return t.handleNewsFindConversation(ctx, c)
	case state >= StateWaitingAdjustTargetPositionInputTargetPrice && state <= StateWaitingAdjustTargetPositionConfirm:
		return t.handleAdjustTargetPositionConversation(ctx, c)
	default:
		// If no specific conversation is matched, maybe it's a dangling state.
		t.ResetUserState(userID)
		return c.Send("Sepertinya Anda tidak sedang dalam percakapan aktif. Gunakan /help untuk melihat perintah yang tersedia.")
	}
}

func (t *TelegramBotService) handleTextMessage(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID

	// If user is in a conversation, handle it
	if state, ok := t.userStates[userID]; ok && state != StateIdle {
		t.handleConversation(ctx, c)
		return nil
	}

	// Cek apakah bukan command
	if !strings.HasPrefix(c.Text(), "/") {
		return c.Send("Saya tidak mengenali perintahmu. Gunakan /help untuk melihat daftar perintah.")
	}

	return nil
}
