package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/telebot.v3"
)

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
	sb.WriteString(fmt.Sprintf("Trend: %s\n", position.Recommendation.TechnicalAnalysis.Trend))
	sb.WriteString(fmt.Sprintf("EMA: %s | RSI: %s\n", position.Recommendation.TechnicalAnalysis.EMASignal, position.Recommendation.TechnicalAnalysis.RSISignal))
	sb.WriteString(fmt.Sprintf("MACD: %s\n", position.Recommendation.TechnicalAnalysis.MACDSignal))
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
			sb.WriteString(fmt.Sprintf("   📈 Profit: %.2f%% | Risk Reward Ratio: %.2f | Max Days: %d\n", stock.ProfitPercentage, stock.RiskRewardRatio, stock.MaxHoldingDays))
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

func (t *TelegramBotService) FormatAnalysisMessage(analysis *models.IndividualAnalysisResponse) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("📊 **Analysis for %s**\n", analysis.Symbol))
	sb.WriteString(fmt.Sprintf("📅 Date: %s\n", analysis.AnalysisDate.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("🎯 Signal: **%s**\n\n", analysis.Recommendation.Action))

	// Technical Analysis Summary
	sb.WriteString("🔧 **Technical Analysis:**\n")
	sb.WriteString(fmt.Sprintf("• Trend: %s \n", analysis.TechnicalAnalysis.Trend))
	sb.WriteString(fmt.Sprintf("• EMA Signal: %s\n", analysis.TechnicalAnalysis.EMASignal))
	sb.WriteString(fmt.Sprintf("• RSI: %s\n", analysis.TechnicalAnalysis.RSISignal))
	sb.WriteString(fmt.Sprintf("• MACD: %s\n", analysis.TechnicalAnalysis.MACDSignal))
	sb.WriteString(fmt.Sprintf("• Momentum: %s\n", analysis.TechnicalAnalysis.Momentum))
	sb.WriteString(fmt.Sprintf("• Bollinger Bands Position: %s\n", analysis.TechnicalAnalysis.BollingerBandsPosition))
	sb.WriteString(fmt.Sprintf("• Support Level: $%.2f\n", analysis.TechnicalAnalysis.SupportLevel))
	sb.WriteString(fmt.Sprintf("• Resistance Level: $%.2f\n", analysis.TechnicalAnalysis.ResistanceLevel))
	sb.WriteString(fmt.Sprintf("• Technical Score: %d/100\n", analysis.TechnicalAnalysis.TechnicalScore))
	if len(analysis.TechnicalAnalysis.KeyInsights) > 0 {
		sb.WriteString("\n📌 **Key Insights:**\n")
		for _, insight := range analysis.TechnicalAnalysis.KeyInsights {
			sb.WriteString(fmt.Sprintf("• %s\n", utils.CapitalizeSentence(insight)))
		}
		sb.WriteString("\n")
	}

	// News Summary
	sb.WriteString("📰 **News Summary Analysis:**\n")
	sb.WriteString(fmt.Sprintf("Confidence Score: %.2f\n", analysis.NewsSummary.ConfidenceScore))
	sb.WriteString(fmt.Sprintf("Sentiment: %s\n", analysis.NewsSummary.Sentiment))
	sb.WriteString(fmt.Sprintf("Impact: %s\n\n", analysis.NewsSummary.Impact))

	sb.WriteString("🗞 **Key Issues:**\n")
	if len(analysis.NewsSummary.KeyIssues) > 0 {
		for _, issue := range analysis.NewsSummary.KeyIssues {
			sb.WriteString(fmt.Sprintf("• %s\n", utils.CapitalizeSentence(issue)))
		}
	}
	sb.WriteString("\n")

	// Recommendation
	sb.WriteString("💡 **Recommendation:**\n")
	sb.WriteString(fmt.Sprintf("• 💵 Buy Price: $%.2f\n", analysis.Recommendation.BuyPrice))
	sb.WriteString(fmt.Sprintf("• 🎯 Target Price: $%.2f\n", analysis.Recommendation.TargetPrice))
	sb.WriteString(fmt.Sprintf("• 🛡 Stop Loss: $%.2f\n", analysis.Recommendation.CutLoss))
	sb.WriteString(fmt.Sprintf("• 🔁 Risk/Reward Ratio: %.2f\n", analysis.Recommendation.RiskRewardRatio))
	sb.WriteString(fmt.Sprintf("• 📊 Confidence: %d%%\n\n", analysis.Recommendation.ConfidenceLevel))
	// Reasoning
	sb.WriteString(fmt.Sprintf("🧠 **Reasoning:**\n %s\n\n", analysis.Recommendation.Reasoning))

	return sb.String()
}

func (t *TelegramBotService) FormatResultSetPositionMessage(data *models.RequestSetPositionData) string {
	var sb strings.Builder

	sb.WriteString("💾 Posisi saham berhasil disimpan!\n\n")
	sb.WriteString("📊 Detail:\n")
	sb.WriteString("— Saham: " + data.Symbol + "\n")
	sb.WriteString("— Harga Beli: " + strconv.FormatFloat(data.BuyPrice, 'f', 0, 64) + "\n")
	sb.WriteString("— Tanggal Beli: " + data.BuyDate + "\n")
	sb.WriteString("— Take Profit: " + strconv.FormatFloat(data.TakeProfit, 'f', 0, 64) + "\n")
	sb.WriteString("— Stop Loss: " + strconv.FormatFloat(data.StopLoss, 'f', 0, 64) + "\n")
	sb.WriteString("— Max Hold: " + strconv.Itoa(data.MaxHolding) + " hari\n\n")

	if data.AlertPrice {
		sb.WriteString("🔔 Alert harga *ON* — sistem akan kirim notifikasi jika harga menyentuh TP atau SL.\n")
	} else {
		sb.WriteString("🔕 Alert harga *OFF*.\n")
	}

	if data.AlertMonitor {
		sb.WriteString("🧠 Monitoring *ON* — kamu akan dapat laporan harian selama posisi masih berjalan.")
	} else {
		sb.WriteString("🧠 Monitoring *OFF*.\n")
	}

	return sb.String()
}

func (t *TelegramBotService) FormatMyPositionMessage(position models.StockPositionEntity, index, total int) string {
	now := time.Now()
	age := int(now.Sub(position.BuyDate).Hours() / 24)
	if age < 0 {
		age = 0
	}
	remaining := position.MaxHoldingPeriodDays - age
	if remaining < 0 {
		remaining = 0
	}

	gain := float64(position.TakeProfitPrice-position.BuyPrice) / float64(position.BuyPrice) * 100
	loss := float64(position.BuyPrice-position.StopLossPrice) / float64(position.BuyPrice) * 100

	alertStatus := "✅ Aktif"
	if position.PriceAlert == nil || !*position.PriceAlert {
		alertStatus = "❌ Tidak Aktif"
	}

	monitorStatus := "✅ Aktif"
	if position.MonitorPosition == nil || !*position.MonitorPosition {
		monitorStatus = "❌ Tidak Aktif"
	}

	return fmt.Sprintf(
		"```\n📊 (%d/%d) Monitoring Saham\n\n"+
			"📦 %s\n"+
			"──────────────────────\n"+
			"💰 Harga Beli   : %.2f\n"+
			"🎯 Target Jual  : %.2f (+%.1f%%)\n"+
			"🛑 Stop Loss    : %.2f (−%.1f%%)\n"+
			"📅 Tgl Beli     : %s\n"+
			"⏳ Umur Posisi  : %d hari\n"+
			"⌛ Sisa Waktu   : %d hari\n"+
			"──────────────────────\n"+
			"🔔 Alert        : %s\n"+
			"📡 Monitoring   : %s\n"+
			"──────────────────────\n```",
		index, total,
		position.StockCode,
		position.BuyPrice,
		position.TakeProfitPrice, gain,
		position.StopLossPrice, loss,
		position.BuyDate.Format("02 Jan 2006"),
		age,
		remaining,
		alertStatus,
		monitorStatus,
	)
}

func (t *TelegramBotService) FormatMyStockPositionMessage(position models.StockPositionEntity) string {
	now := time.Now()
	age := int(now.Sub(position.BuyDate).Hours() / 24)
	if age < 0 {
		age = 0
	}
	remaining := position.MaxHoldingPeriodDays - age
	if remaining < 0 {
		remaining = 0
	}

	gain := float64(position.TakeProfitPrice-position.BuyPrice) / float64(position.BuyPrice) * 100
	loss := float64(position.BuyPrice-position.StopLossPrice) / float64(position.BuyPrice) * 100

	alertStatus := "✅ Aktif"
	if position.PriceAlert == nil || !*position.PriceAlert {
		alertStatus = "❌ Tidak Aktif"
	}

	monitorStatus := "✅ Aktif"
	if position.MonitorPosition == nil || !*position.MonitorPosition {
		monitorStatus = "❌ Tidak Aktif"
	}

	return fmt.Sprintf(
		"```\n📊 Monitoring Saham\n\n"+
			"📦 %s\n"+
			"──────────────────────\n"+
			"💰 Harga Beli   : %.2f\n"+
			"🎯 Target Jual  : %.2f (+%.1f%%)\n"+
			"🛑 Stop Loss    : %.2f (−%.1f%%)\n"+
			"📅 Tgl Beli     : %s\n"+
			"⏳ Umur Posisi  : %d hari\n"+
			"⌛ Sisa Waktu   : %d hari\n"+
			"──────────────────────\n"+
			"🔔 Alert        : %s\n"+
			"📡 Monitoring   : %s\n"+
			"──────────────────────\n```",
		position.StockCode,
		position.BuyPrice,
		position.TakeProfitPrice, gain,
		position.StopLossPrice, loss,
		position.BuyDate.Format("02 Jan 2006"),
		age,
		remaining,
		alertStatus,
		monitorStatus,
	)
}

func (t *TelegramBotService) FormatNotesTimeFrameStockMessage() string {
	return `
Berikut adalah penjelasan singkat tentang setiap time frame:
━━━━━━━━━━━━━

🔹 *Main Signal*  
⏱️ Time Frame: 1 hari  |  📅 Range: 3 bulan  
📌 Untuk melihat arah tren besar saham.  
👉 Cocok kalau kamu ingin tahu apakah saham ini sedang bagus untuk dibeli.

🔹 *Entry Presisi*  
⏱️ Time Frame: 4 jam  |  📅 Range: 1 bulan  
📌 Untuk cari waktu terbaik masuk setelah sinyal beli muncul.  
👉 Cocok kalau kamu sudah yakin mau beli, tapi ingin harga yang lebih pas.

🔹 *Exit Presisi*  
⏱️ Time Frame: 1 jam  |  📅 Range: 14 hari  
📌 Untuk bantu kamu ambil untung atau cut loss.  
👉 Cocok kalau kamu sudah punya saham dan ingin tahu kapan jual.
`
}

func (t *TelegramBotService) ShowAnalysisInProgress(stockCode string, interval string, period string) string {
	return fmt.Sprintf(`
🔍 Sedang menganalisis *$%s*...

🕐 Interval: %s  
📆 Range: %s

⏳ Mohon tunggu sebentar, bot sedang memproses data:
- Mengambil data harga 📈
- Menghitung sinyal teknikal 📊
- Menyusun rekomendasi 💡

📬 Hasil analisa akan muncul dalam beberapa detik...
`, stockCode, interval, period)

}

func (t *TelegramBotService) showLoadingFlowAnalysis(c telebot.Context, stop <-chan struct{}) *telebot.Message {
	msgRoot := c.Message()
	initial := "Sedang menganalisis saham kamu, mohon tunggu"

	var msg *telebot.Message
	var err error

	// Cek apakah pesan terakhir berasal dari bot
	if msgRoot == nil || msgRoot.Sender == nil || !msgRoot.Sender.IsBot {
		msg, err = t.bot.Send(c.Chat(), initial, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send loading message")
			return nil
		}
	} else {
		msg, err = t.bot.Edit(msgRoot, initial)
		if err != nil {
			t.logger.WithError(err).Error("Failed to edit loading message")
			return nil
		}
	}

	go func() {
		dots := []string{"⏳", "⏳⏳", "⏳⏳⏳"}
		i := 0
		for {
			if utils.ShouldStopChan(stop, t.logger) {
				return
			}
			_, err := t.bot.Edit(msg, fmt.Sprintf("%s%s", initial, dots[i%len(dots)]))
			if err != nil {
				t.logger.WithError(err).Error("Failed to update loading animation")
				return
			}
			i++
			time.Sleep(200 * time.Millisecond)
		}
	}()

	return msg
}

func (t *TelegramBotService) showLoadingGeneral(c telebot.Context, stop <-chan struct{}) *telebot.Message {
	msgRoot := c.Message()

	initial := "Mohon tunggu sebentar, bot sedang memproses data"
	msg, _ := t.bot.Edit(msgRoot, initial)

	go func() {
		dots := []string{"⏳", "⏳⏳", "⏳⏳⏳"}
		i := 0
		for {
			if utils.ShouldStopChan(stop, t.logger) {
				return
			}
			_, err := t.bot.Edit(msg, fmt.Sprintf("%s%s", initial, dots[i%len(dots)]))
			if err != nil {
				t.logger.WithError(err).Error("Failed to update loading animation")
				return
			}
			i++
			time.Sleep(200 * time.Millisecond)
		}
	}()

	return msg
}

func (t *TelegramBotService) showLoadingBuyList(c telebot.Context, stockCode string, msgRoot *telebot.Message, stop <-chan struct{}, result *strings.Builder) *telebot.Message {
	steps := []string{
		fmt.Sprintf("📊 *Sedang menganalisis saham %s...*\n", stockCode),
		"🔍 Langkah 1: Mengecek pergerakan harga (OHLC)...",
		"🗞️ Langkah 2: Memindai berita dan sentimen pasar...",
		"🧠 Langkah 3: AI sedang melakukan analisa teknikal & fundamental...",
		"\nMohon tunggu sebentar, hasil segera keluar...",
	}

	stepsCount := 0

	sb := &strings.Builder{}
	sb.WriteString(result.String())
	sb.WriteString("\n")
	sb.WriteString(steps[stepsCount])

	utils.SafeGo(func() {
		dots := []string{"⏳", "⏳⏳", "⏳⏳⏳"}
		i := 0
		stepsCount++
		for {
			if utils.ShouldStopChan(stop, t.logger) {
				return
			}

			if stepsCount < len(steps) {
				sb.WriteString("\n" + steps[stepsCount])
				stepsCount++
				_, err := t.bot.Edit(msgRoot, sb.String(), telebot.ModeMarkdown)
				if err != nil {
					t.logger.WithError(err).Error("Failed to update loading animation")
					return
				}

				if stepsCount < len(steps) {
					time.Sleep(200 * time.Millisecond)
					sb.WriteString("✅")
					_, err = t.bot.Edit(msgRoot, sb.String(), telebot.ModeMarkdown)
					if err != nil {
						t.logger.WithError(err).Error("Failed to update loading animation")
						return
					}
				}

				continue
			}

			_, err := t.bot.Edit(msgRoot, fmt.Sprintf("%s%s", sb.String(), dots[i%len(dots)]), telebot.ModeMarkdown)
			if err != nil {
				t.logger.WithError(err).Error("Failed to update loading animation")
				return
			}
			i++
			time.Sleep(400 * time.Millisecond)
		}
	})

	return msgRoot
}

type Progress struct {
	Index     int
	StockCode string
	Content   string
	Header    string
}

func (t *TelegramBotService) showProgressBarWithChannel(
	ctx context.Context,
	c telebot.Context,
	msgRoot *telebot.Message,
	progressCh <-chan Progress,
	totalSteps int,
	wg *sync.WaitGroup,
) {
	utils.SafeGo(func() {
		const barLength = 15 // total panjang bar, bisa diubah sesuai estetika

		current := Progress{Index: 0}

		defer func() {
			result := fmt.Sprintf("%s\n%s", current.Header, current.Content)
			_, errInner := t.telegramRateLimiter.EditWithoutLimit(ctx, c, msgRoot, result, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
			if errInner != nil {
				t.logger.WithError(errInner).Error("Gagal edit pesan")
			}
			wg.Done()
		}()

		for {
			select {
			case <-ctx.Done():
				t.logger.WithError(ctx.Err()).Error("Done signal received")

				return

			case newProgress, ok := <-progressCh:
				if !ok {
					t.logger.Warn("showProgressBarWithChannel - Progress channel closed")
					return
				}

				current = newProgress

				// Hitung persen dan jumlah "blok" progress
				percent := int(float64(current.Index) / float64(totalSteps) * 100)
				progressBlocks := int(float64(barLength) * float64(current.Index) / float64(totalSteps))
				if progressBlocks > barLength {
					progressBlocks = barLength
				}

				// Buat bar: ▓ untuk progress, ░ untuk sisanya
				currentAnalysis := fmt.Sprintf(messageLoadingAnalysis, current.StockCode)
				filled := strings.Repeat("▓", progressBlocks)
				empty := strings.Repeat("░", barLength-progressBlocks)
				progressBar := fmt.Sprintf("⏳ Progress: [%s%s] %d%%", filled, empty, percent)

				menu := &telebot.ReplyMarkup{}
				btnCancel := menu.Data(btnCancelBuyListAnalysis.Text, btnCancelBuyListAnalysis.Unique)
				menu.Inline(menu.Row(btnCancel))

				body := &strings.Builder{}
				body.WriteString(current.Header)
				body.WriteString("\n")

				if current.Content != "" {
					body.WriteString(current.Content)
					body.WriteString("\n\n")
				}

				body.WriteString(currentAnalysis)
				body.WriteString("\n")
				body.WriteString(progressBar)

				time.Sleep(100 * time.Millisecond)

				if msgRoot == nil {
					msgNew, err := t.telegramRateLimiter.Send(ctx, c, body.String(), menu, telebot.ModeMarkdown)
					if err != nil {
						t.logger.WithError(err).Error("Gagal create progress bar")
					}
					msgRoot = msgNew
				} else {
					_, err := t.telegramRateLimiter.Edit(ctx, c, msgRoot, body.String(), menu, telebot.ModeMarkdown)
					if err != nil {
						t.logger.WithError(err).Error("Gagal update progress bar")
					}
				}

			}
		}
	})
}

func (t *TelegramBotService) formatMessageBuyList(index int, analysis *models.IndividualAnalysisResponse) *strings.Builder {
	profitPercentage := ((analysis.Recommendation.TargetPrice - analysis.Recommendation.BuyPrice) / analysis.Recommendation.BuyPrice) * 100
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("\n• `$%s`\n", analysis.Symbol))
	sb.WriteString(fmt.Sprintf("   💵 Buy: %d\n", int(analysis.Recommendation.BuyPrice)))
	sb.WriteString(fmt.Sprintf("   🎯 TP: %d  🛡 SL: %d\n", int(analysis.Recommendation.TargetPrice), int(analysis.Recommendation.CutLoss)))
	sb.WriteString(fmt.Sprintf("   🔁 RR: %.1f   💰 Profit: +%.1f%%", analysis.Recommendation.RiskRewardRatio, profitPercentage))

	return sb

}

func (t *TelegramBotService) formatMessageMenuNews() string {
	result := `📋 Menu Berita Saham

Tetap update dengan pergerakan pasar.
Cari berita saham, aktifkan rangkuman harian,
atau nyalakan alert otomatis saat muncul berita penting
yang bisa mempengaruhi harga saham.

Pilih fitur yang ingin kamu akses:`

	return result

}

func (t *TelegramBotService) formatMessageNewsList(newsList []models.StockNewsEntity, age int) string {
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("📢 Berikut adalah rangkuman berita penting terbaru yang berkaitan dengan saham $%s dalam %d hari terakhir", newsList[0].StockCode, age))

	for idx, news := range newsList {
		sb.WriteString(fmt.Sprintf("\n\n %d. %s\n", idx+1, utils.TruncateTitle(news.Title, 80)))
		sb.WriteString(fmt.Sprintf(" 📅 %s | 🌐 %s\n", news.PublishedAt.Format("2006-01-02"), utils.ExtractDomain(news.Link)))
		sb.WriteString(fmt.Sprintf(" 📊 Sentimen: %s | 💯 Score: %.2f\n", news.Sentiment, news.FinalScore))
		sb.WriteString(fmt.Sprintf(" 🧠 %s\n", utils.TruncateTitle(news.Reason, 200)))
		link := fmt.Sprintf("[Selengkapnya](%s)", news.Link)
		sb.WriteString(fmt.Sprintf(" 🔗 %s\n", link))
	}
	sb.WriteString("\n")

	return sb.String()
}

func (t *TelegramBotService) formatMessageNewsSummary(summary *models.StockNewsSummaryEntity) string {

	action := strings.ToUpper(summary.SuggestedAction)
	iconAction := "❔"
	if action == "HOLD" {
		iconAction = "🟡"
	} else if action == "SELL" {
		iconAction = "🔴"
	} else if action == "BUY" {
		iconAction = "🟢"
	}

	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("📚 *Ringkasan Analisis Saham $%s*\n\n", summary.StockCode))
	sb.WriteString(fmt.Sprintf("📝 *TL;DR:* %s\n\n", summary.ShortSummary))
	sb.WriteString(fmt.Sprintf("🧠 *Sentimen:* %s\n", summary.SummarySentiment))
	sb.WriteString(fmt.Sprintf("📈 *Dampak:* %s\n", summary.SummaryImpact))
	sb.WriteString(fmt.Sprintf("📉 *Confidence Score:* %.2f\n", summary.SummaryConfidenceScore))
	sb.WriteString(fmt.Sprintf("🎯 *Saran:* %s %s\n", iconAction, action))
	sb.WriteString("\n🔑 *Isu Kunci:*\n")
	for _, issue := range summary.KeyIssues {
		sb.WriteString(fmt.Sprintf("• %s\n", issue))
	}
	sb.WriteString(fmt.Sprintf("\n🧩 *Alasan:* %s\n\n", summary.Reasoning))
	sb.WriteString(fmt.Sprintf("📆 *Periode:* %s - %s\n", summary.SummaryStart.Format("2006-01-02"), summary.SummaryEnd.Format("2006-01-02")))

	return sb.String()
}
