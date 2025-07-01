package telegram_bot

import "gopkg.in/telebot.v3"

// UI elements for telegram bot
var (
	btnSetPositionAlertPriceYes    telebot.Btn = telebot.Btn{Text: "✅ Ya", Unique: "btn_set_position_alert_price_yes", Data: "true"}
	btnSetPositionAlertPriceNo     telebot.Btn = telebot.Btn{Text: "❌ Tidak", Unique: "btn_set_position_alert_price_no", Data: "false"}
	btnSetPositionAlertMonitorYes  telebot.Btn = telebot.Btn{Text: "✅ Ya", Unique: "btn_set_position_alert_monitor_yes", Data: "true"}
	btnSetPositionAlertMonitorNo   telebot.Btn = telebot.Btn{Text: "❌ Tidak", Unique: "btn_set_position_alert_monitor_no", Data: "false"}
	btnStockPositionMonitoring     telebot.Btn = telebot.Btn{Unique: "btn_stock_position_monitoring"}
	btnManageStockPosition         telebot.Btn = telebot.Btn{Text: "⚙️ Kelola", Unique: "btn_manage_stock_position"}
	btnToDetailStockPosition       telebot.Btn = telebot.Btn{Unique: "btn_list_stock_position"}
	btnBackStockPosition           telebot.Btn = telebot.Btn{Text: "🔙 Kembali", Unique: "btn_back_stock_position"}
	btnNewsStockPosition           telebot.Btn = telebot.Btn{Text: "📰 Berita", Unique: "btn_news_stock_position"}
	btnBackActionStockPosition     telebot.Btn = telebot.Btn{Text: "🔙 Kembali", Unique: "btn_back_action_stock_position"}
	btnBackDetailStockPosition     telebot.Btn = telebot.Btn{Text: "🔙 Kembali", Unique: "btn_back_detail_stock_position"}
	btnDeleteMessage               telebot.Btn = telebot.Btn{Text: "🗑️ Hapus Pesan", Unique: "btn_delete_message"}
	btnDeleteStockPosition         telebot.Btn = telebot.Btn{Text: "🗑️ Hapus Posisi", Unique: "btn_delete_stock_position"}
	btnUpdateAlertPrice            telebot.Btn = telebot.Btn{Unique: "btn_update_alert_price"}
	btnUpdateAlertMonitor          telebot.Btn = telebot.Btn{Unique: "btn_update_alert_monitor"}
	btnExitStockPosition           telebot.Btn = telebot.Btn{Unique: "btn_exit_stock_position"}
	btnSaveExitPosition            telebot.Btn = telebot.Btn{Text: "💾 Simpan", Unique: "btn_save_exit_position"}
	btnCancelGeneral               telebot.Btn = telebot.Btn{Text: "❌ Batal", Unique: "btn_cancel_general"}
	btnCancelBuyListAnalysis       telebot.Btn = telebot.Btn{Text: "⛔ Hentikan Analisis", Unique: "btn_cancel_buy_list_analysis"}
	btnActionNewsFind              telebot.Btn = telebot.Btn{Text: "• Cari Berita", Unique: "btn_action_news_find"}
	btnActionTopNews               telebot.Btn = telebot.Btn{Text: "• Top Berita Saham", Unique: "btn_action_top_news"}
	btnNewsConfirmSendSummary      telebot.Btn = telebot.Btn{Unique: "btn_news_confirm_send_summary"}
	btnAdjustTargetPosition        telebot.Btn = telebot.Btn{Text: "🎯 Atur Target", Unique: "btn_adjust_target_position"}
	btnAdjustTargetPositionConfirm telebot.Btn = telebot.Btn{Text: "✅ Konfirmasi", Unique: "btn_adjust_target_position_confirm"}
)

var (
	commonMessageInternalError  string = "❌ Terjadi kesalahan internal, silakan coba lagi."
	messageLoadingAnalysis      string = "🔍 Menganalisis: $%s"
	messageAnalysisNotAvailable string = "🔍Saat ini, data analisa untuk saham $%s belum tersedia.\n\nNamun jangan khawatir — proses analisa sedang kami mulai untuk mendapatkan insight terbaru. Kami akan segera memberitahumu begitu hasil analisa siap.\n\nMohon ditunggu sebentar, ya!"
)
