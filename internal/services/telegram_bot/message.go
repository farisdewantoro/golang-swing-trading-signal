package telegram_bot

import (
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strconv"
	"strings"
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
	sb.WriteString(fmt.Sprintf("Trend: %s (Strength: %s)\n", position.Recommendation.TechnicalAnalysis.Trend, position.Recommendation.TechnicalAnalysis.TrendStrength))
	sb.WriteString(fmt.Sprintf("EMA: %s | RSI: %s\n", position.Recommendation.TechnicalAnalysis.EMASignal, position.Recommendation.TechnicalAnalysis.RSISignal))
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

	sb.WriteString(fmt.Sprintf("📊 <b>Analysis for %s</b>\n", analysis.Symbol))
	sb.WriteString(fmt.Sprintf("📅 Date: %s\n", analysis.AnalysisDate.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("🎯 Signal: <b>%s</b>\n\n", analysis.Signal))

	// Technical Analysis Summary
	sb.WriteString("🔧 <b>Technical Analysis:</b>\n")
	sb.WriteString(fmt.Sprintf("Trend: %s (Strength: %s)\n", analysis.TechnicalAnalysis.Trend, analysis.TechnicalAnalysis.TrendStrength))
	sb.WriteString(fmt.Sprintf("EMA Signal: %s\n", analysis.TechnicalAnalysis.EMASignal))
	sb.WriteString(fmt.Sprintf("RSI: %s\n", analysis.TechnicalAnalysis.RSISignal))
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
	msg, _ := t.bot.Edit(msgRoot, initial)

	go func() {
		dots := []string{"⏳", "⏳⏳", "⏳⏳⏳"}
		i := 0
		for {
			select {
			case <-stop:
				// Stop signal diterima, keluar dari loop
				return
			default:
				t.bot.Edit(msg, fmt.Sprintf("%s%s", initial, dots[i%len(dots)]))
				i++
				time.Sleep(200 * time.Millisecond)
			}
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
			select {
			case <-stop:
				// Stop signal diterima, keluar dari loop
				return
			default:
				t.bot.Edit(msg, fmt.Sprintf("%s%s", initial, dots[i%len(dots)]))
				i++
				time.Sleep(200 * time.Millisecond)
			}
		}
	}()

	return msg
}
