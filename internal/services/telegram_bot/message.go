package telegram_bot

import (
	"context"
	"encoding/json"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) FormatPositionMonitoringMessage(position *models.PositionMonitoringResponseMultiTimeframe) string {
	var sb strings.Builder

	unrealizedPnLPercentage := ((position.MarketPrice - position.BuyPrice) / position.BuyPrice) * 100

	daysRemaining := utils.RemainingDays(position.MaxHoldingPeriodDays, position.BuyDate)
	ageDays := int(time.Since(position.BuyDate).Hours() / 24)

	iconAction := "❔"
	if position.Action == "HOLD" {
		iconAction = "🟡"
	} else if position.Action == "CUT_LOSS" {
		iconAction = "🔴"
	} else if position.Action == "TAKE_PROFIT" {
		iconAction = "🟢"
	} else if position.Action == "TRAIL_STOP" {
		iconAction = "🟠"
	}

	sb.WriteString(fmt.Sprintf("\n📊 <b>Position Update: %s</b>\n", position.Symbol))
	sb.WriteString(fmt.Sprintf("💰 Buy: $%d\n", int(position.BuyPrice)))
	sb.WriteString(fmt.Sprintf("📌 Last Price: $%d %s\n", int(position.MarketPrice), utils.FormatPercentage(unrealizedPnLPercentage)))
	sb.WriteString(fmt.Sprintf("🎯 TP: $%d | SL: $%d | RR: %.2f\n", int(position.TargetPrice), int(position.CutLoss), position.RiskRewardRatio))
	sb.WriteString(fmt.Sprintf("📈 Age: %d days | Remaining: %d days\n\n", ageDays, daysRemaining))

	// Recommendation
	gain := float64(position.ExitTargetPrice-position.BuyPrice) / float64(position.BuyPrice) * 100
	loss := float64(position.ExitCutLossPrice-position.BuyPrice) / float64(position.BuyPrice) * 100
	sb.WriteString("💡 <b>Recommendation:</b>\n")
	sb.WriteString(fmt.Sprintf(" • Action: %s %s\n", iconAction, position.Action))
	sb.WriteString(fmt.Sprintf(" • Target Price: $%d %s\n", int(position.ExitTargetPrice), utils.FormatPercentage(gain)))
	sb.WriteString(fmt.Sprintf(" • Stop Loss: $%d %s\n", int(position.ExitCutLossPrice), utils.FormatPercentage(loss)))
	sb.WriteString(fmt.Sprintf(" • Risk/Reward Ratio: %.2f\n", position.ExitRiskRewardRatio))
	sb.WriteString(fmt.Sprintf(" • Confidence: %d%%\n", position.ConfidenceLevel))
	sb.WriteString(fmt.Sprintf(" • Technical Score: %d\n\n", position.TechnicalScore))
	// Reasoning
	sb.WriteString(fmt.Sprintf("🧠 <b>Reasoning:</b>\n %s\n\n", position.Reasoning))

	// Technical Analysis
	sb.WriteString("🔍 <b>Analisa Multi-Timeframe</b>")
	sb.WriteString(fmt.Sprintf("\n<b>Daily (1D)</b>: %s | RSI: %d\n", position.TimeframeAnalysis.Timeframe4H.Trend, position.TimeframeAnalysis.Timeframe1D.RSI))
	sb.WriteString(fmt.Sprintf("> Sinyal Kunci: %s\n", position.TimeframeAnalysis.Timeframe1D.KeySignal))
	sb.WriteString(fmt.Sprintf("> Support/Resistance: %d/%d\n", int(position.TimeframeAnalysis.Timeframe1D.Support), int(position.TimeframeAnalysis.Timeframe1D.Resistance)))

	sb.WriteString(fmt.Sprintf("\n<b>4 Hours (4H)</b>: %s | RSI: %d\n", position.TimeframeAnalysis.Timeframe4H.Trend, position.TimeframeAnalysis.Timeframe4H.RSI))
	sb.WriteString(fmt.Sprintf("> Sinyal Kunci: %s\n", position.TimeframeAnalysis.Timeframe4H.KeySignal))
	sb.WriteString(fmt.Sprintf("> Support/Resistance: %d/%d\n", int(position.TimeframeAnalysis.Timeframe4H.Support), int(position.TimeframeAnalysis.Timeframe4H.Resistance)))

	sb.WriteString(fmt.Sprintf("\n<b>1 Hour (1H)</b>: %s | RSI: %d\n", position.TimeframeAnalysis.Timeframe1H.Trend, position.TimeframeAnalysis.Timeframe1H.RSI))
	sb.WriteString(fmt.Sprintf("> Sinyal Kunci: %s\n", position.TimeframeAnalysis.Timeframe1H.KeySignal))
	sb.WriteString(fmt.Sprintf("> Support/Resistance: %d/%d\n", int(position.TimeframeAnalysis.Timeframe1H.Support), int(position.TimeframeAnalysis.Timeframe1H.Resistance)))

	// News Summary
	sb.WriteString("\n📰 <b>News Analysis:</b>\n")
	if position.NewsSummary.ConfidenceScore > 0 {
		sb.WriteString(fmt.Sprintf("Confidence Score: %.2f\n", position.NewsSummary.ConfidenceScore))
		sb.WriteString(fmt.Sprintf("Sentiment: %s\n", position.NewsSummary.Sentiment))
		sb.WriteString(fmt.Sprintf("Impact: %s\n\n", position.NewsSummary.Impact))
		sb.WriteString(fmt.Sprintf("🧠 News Insight: \n%s\n\n", position.NewsSummary.Reasoning))
	} else {
		sb.WriteString("<i>Belum ada data berita terbaru yang tersedia untuk saham ini.</i>\n\n")
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("📅 <i>Terakhir dianalisis: %s</i>\n", position.AnalysisDate.Format("2006-01-02 15:04:05")))

	return sb.String()
}

func (t *TelegramBotService) FormatAnalysisMessage(analysis *models.IndividualAnalysisResponseMultiTimeframe) string {
	var sb strings.Builder
	signalIcon := "🟡"
	if analysis.Action == "BUY" {
		signalIcon = "🟢"
	}
	sb.WriteString(fmt.Sprintf("\n%s <b>SIGNAL %s: $%s</b> %s\n\n", signalIcon, analysis.Action, analysis.Symbol, signalIcon))
	// Recommendation
	if analysis.Action != "HOLD" {
		gain := float64(analysis.TargetPrice-analysis.BuyPrice) / float64(analysis.BuyPrice) * 100
		loss := float64(analysis.CutLoss-analysis.BuyPrice) / float64(analysis.BuyPrice) * 100
		sb.WriteString("<b>Trade Plan</b>\n")
		sb.WriteString(fmt.Sprintf("📌 Last Price: %d (%s)\n", int(analysis.MarketPrice), analysis.AnalysisDate.Format("01-02 15:04")))
		sb.WriteString(fmt.Sprintf("💵 Buy Area: $%d\n", int(analysis.BuyPrice)))
		sb.WriteString(fmt.Sprintf("🎯 Target Price: $%d %s\n", int(analysis.TargetPrice), utils.FormatPercentage(gain)))
		sb.WriteString(fmt.Sprintf("🛡 Cut Loss: $%d %s\n", int(analysis.CutLoss), utils.FormatPercentage(loss)))
		sb.WriteString(fmt.Sprintf("⚖️ Risk/Reward Ratio: %.2f\n", analysis.RiskRewardRatio))
		sb.WriteString(fmt.Sprintf("<i>⏳ Estimasi Waktu Profit: %d hari kerja</i>\n", analysis.EstimatedHoldingDays))
	} else if analysis.Action == "HOLD" {
		sb.WriteString("<b>Status saat ini</b>\n")
		sb.WriteString(fmt.Sprintf("📌 Last Price: %d (%s)\n", int(analysis.MarketPrice), analysis.AnalysisDate.Format("01-02 15:04")))
		if analysis.EstimatedHoldingDays > 0 {
			sb.WriteString(fmt.Sprintf("<i>🔍 Perkiraan Waktu Tunggu: %d hari kerja</i>\n", analysis.EstimatedHoldingDays))
		}
	}

	sb.WriteString("\n<b>Key Metrics</b>\n")
	sb.WriteString(fmt.Sprintf("📶 Confidence: %d%%\n", analysis.ConfidenceLevel))
	sb.WriteString(fmt.Sprintf("🔢 Technical Score: %d\n", analysis.TechnicalScore))

	// Reasoning
	sb.WriteString(fmt.Sprintf("\n🧠 <b>Reasoning:</b>\n%s\n\n", analysis.Reasoning))

	sb.WriteString("🔍 <b>Analisa Multi-Timeframe</b>")
	sb.WriteString(fmt.Sprintf("\n<b>Daily (1D)</b>: %s | RSI: %d\n", analysis.TimeframeAnalysis.Timeframe4H.Trend, analysis.TimeframeAnalysis.Timeframe1D.RSI))
	sb.WriteString(fmt.Sprintf("> Sinyal Kunci: %s\n", analysis.TimeframeAnalysis.Timeframe1D.KeySignal))
	sb.WriteString(fmt.Sprintf("> Support/Resistance: %d/%d\n", int(analysis.TimeframeAnalysis.Timeframe1D.Support), int(analysis.TimeframeAnalysis.Timeframe1D.Resistance)))

	sb.WriteString(fmt.Sprintf("\n<b>4 Hours (4H)</b>: %s | RSI: %d\n", analysis.TimeframeAnalysis.Timeframe4H.Trend, analysis.TimeframeAnalysis.Timeframe4H.RSI))
	sb.WriteString(fmt.Sprintf("> Sinyal Kunci: %s\n", analysis.TimeframeAnalysis.Timeframe4H.KeySignal))
	sb.WriteString(fmt.Sprintf("> Support/Resistance: %d/%d\n", int(analysis.TimeframeAnalysis.Timeframe4H.Support), int(analysis.TimeframeAnalysis.Timeframe4H.Resistance)))

	sb.WriteString(fmt.Sprintf("\n<b>1 Hour (1H)</b>: %s | RSI: %d\n", analysis.TimeframeAnalysis.Timeframe1H.Trend, analysis.TimeframeAnalysis.Timeframe1H.RSI))
	sb.WriteString(fmt.Sprintf("> Sinyal Kunci: %s\n", analysis.TimeframeAnalysis.Timeframe1H.KeySignal))
	sb.WriteString(fmt.Sprintf("> Support/Resistance: %d/%d\n", int(analysis.TimeframeAnalysis.Timeframe1H.Support), int(analysis.TimeframeAnalysis.Timeframe1H.Resistance)))

	// News Summary
	sb.WriteString("\n📰 <b>News Analysis:</b>\n")
	if analysis.NewsSummary.ConfidenceScore > 0 {
		sb.WriteString(fmt.Sprintf("Confidence Score: %.2f\n", analysis.NewsSummary.ConfidenceScore))
		sb.WriteString(fmt.Sprintf("Sentiment: %s\n", analysis.NewsSummary.Sentiment))
		sb.WriteString(fmt.Sprintf("Impact: %s\n\n", analysis.NewsSummary.Impact))
		sb.WriteString(fmt.Sprintf("🧠 News Insight: \n%s\n\n", analysis.NewsSummary.Reasoning))
	} else {
		sb.WriteString("<i>Belum ada data berita terbaru yang tersedia untuk saham ini.</i>\n\n")
	}

	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("📅 <i>Terakhir dianalisis: %s</i>\n", analysis.AnalysisDate.Format("2006-01-02 15:04:05")))

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

func (t *TelegramBotService) FormatMyStockPositionMessage(position *models.StockPositionEntity, marketPrice *models.RedisLastPrice) string {
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

	sb := strings.Builder{}
	sb.WriteString("```\n")
	sb.WriteString("📊 Monitoring Saham\n\n")
	sb.WriteString(fmt.Sprintf("📦 %s\n", position.StockCode))
	sb.WriteString("────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("💰 Harga Beli   : %d\n", int(position.BuyPrice)))
	if marketPrice != nil && marketPrice.Price > 0 {
		pnl := (marketPrice.Price - position.BuyPrice) / position.BuyPrice * 100
		sb.WriteString(fmt.Sprintf("💵 Harga Pasar  : %d (%s)\n", int(marketPrice.Price), utils.FormatPercentage(pnl)))
	}
	sb.WriteString(fmt.Sprintf("🎯 Target Jual  : %d %s\n", int(position.TakeProfitPrice), utils.FormatPercentage(gain)))
	sb.WriteString(fmt.Sprintf("🛑 Stop Loss    : %d %s\n", int(position.StopLossPrice), utils.FormatPercentage(loss)))
	sb.WriteString(fmt.Sprintf("📅 Tgl Beli     : %s\n", position.BuyDate.Format("02 Jan 2006")))
	sb.WriteString(fmt.Sprintf("⏳ Umur Posisi  : %d hari\n", age))
	sb.WriteString(fmt.Sprintf("⌛ Sisa Waktu   : %d hari\n", remaining))
	sb.WriteString("────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("🔔 Alert        : %s\n", alertStatus))
	sb.WriteString(fmt.Sprintf("📡 Monitoring   : %s\n", monitorStatus))
	sb.WriteString("────────────────────────────────\n")

	if len(position.StockPositionMonitorings) > 0 {
		sb.WriteString("📖 Riwayat Analisa\n")
		for _, monitoring := range position.StockPositionMonitorings {
			var data models.PositionMonitoringResponseMultiTimeframe
			err := json.Unmarshal([]byte(monitoring.Data), &data)
			if err != nil {
				continue
			}

			iconAction := "🔴"
			switch data.Action {
			case "SELL":
				iconAction = "🟢"
			case "HOLD":
				iconAction = "🟡"
			}
			sb.WriteString("\n")
			sb.WriteString(fmt.Sprintf("• 🕒 %s | Conf: %d/100\n", data.AnalysisDate.Format("02 Jan 15:04"), int(data.ConfidenceLevel)))
			sb.WriteString(fmt.Sprintf("  %s %s @%d (%.2f%%)\n",
				iconAction, data.Action, int(data.MarketPrice), ((data.MarketPrice - data.BuyPrice) / data.BuyPrice * 100)))
		}
	}
	sb.WriteString("```\n")

	return sb.String()
}

func (t *TelegramBotService) FormatMyPositionListMessage(positions []models.StockPositionEntity, lastMarketPriceMap map[string]models.RedisLastPrice) string {
	var sb strings.Builder

	for _, position := range positions {
		var (
			lastMarketPrice   float64
			lastMarketPriceAt time.Time
		)

		sb.WriteString(fmt.Sprintf("\n• %s", position.StockCode))
		sb.WriteString(fmt.Sprintf("\n🎯 Buy: %d | TP: %d | SL: %d\n", int(position.BuyPrice), int(position.TakeProfitPrice), int(position.StopLossPrice)))
		if len(position.StockPositionMonitorings) == 0 {
			sb.WriteString(" ℹ️ <i>Saat ini data belum tersedia. Silakan coba lagi nanti.</i>\n")
			continue
		}
		var dataStockMonitoring *models.PositionMonitoringResponseMultiTimeframe
		err := json.Unmarshal([]byte(position.StockPositionMonitorings[0].Data), &dataStockMonitoring)
		if err != nil {
			sb.WriteString(" ℹ️ <i>Data tidak valid. Silakan coba lagi nanti.</i>\n")
			continue
		}

		if lastMarketPriceData, ok := lastMarketPriceMap[position.StockCode]; ok && lastMarketPriceData.Price > 0 {
			lastMarketPrice = lastMarketPriceData.Price
			lastMarketPriceAt = lastMarketPriceData.Time
		} else {
			lastMarketPrice = dataStockMonitoring.MarketPrice
			lastMarketPriceAt = dataStockMonitoring.AnalysisDate
		}

		sb.WriteString(fmt.Sprintf(" 💰 Last Price: %d (%s)\n", int(lastMarketPrice), lastMarketPriceAt.Format("01/02 15:04")))

		iconAction := "🔴"
		switch dataStockMonitoring.Action {
		case "SELL":
			iconAction = "🟢"
		case "HOLD":
			iconAction = "🟡"
		}

		pnl := (lastMarketPrice - dataStockMonitoring.BuyPrice) / dataStockMonitoring.BuyPrice * 100
		sb.WriteString(fmt.Sprintf(" 📈 PnL: %s\n", utils.FormatPercentage(pnl)))
		sb.WriteString(fmt.Sprintf(" %s %s | Confidence: %d/100\n", iconAction, dataStockMonitoring.Action, int(dataStockMonitoring.ConfidenceLevel)))
		sb.WriteString(fmt.Sprintf(" <i>🗓️ Last Analysis: %s</i>\n", dataStockMonitoring.AnalysisDate.Format("02 Jan 15:04")))
	}
	return sb.String()
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

func (t *TelegramBotService) formatMessageBuyList(index int, analysis *models.IndividualAnalysisResponseMultiTimeframe) *strings.Builder {
	profitPercentage := ((analysis.TargetPrice - analysis.BuyPrice) / analysis.BuyPrice) * 100
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("\n• `$%s` - _(%s)_\n", analysis.Symbol, analysis.AnalysisDate.Format("01/02 15:04")))
	sb.WriteString(fmt.Sprintf("   🔖 Last Price: %d\n", int(analysis.MarketPrice)))
	sb.WriteString(fmt.Sprintf("   💵 Buy: %d 📊 Score: %d\n", int(analysis.BuyPrice), ((analysis.ConfidenceLevel + analysis.TechnicalScore) / 2)))
	sb.WriteString(fmt.Sprintf("   🎯 TP: %d  🛡 SL: %d\n", int(analysis.TargetPrice), int(analysis.CutLoss)))
	sb.WriteString(fmt.Sprintf("   🔁 RR: %.1f   💰 Profit: +%.1f%%\n", analysis.RiskRewardRatio, profitPercentage))
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

func (t *TelegramBotService) formatMessageReportNotExits() string {
	return `📭 *Belum Ada Riwayat Trading*

Kamu belum memiliki data trading yang bisa ditampilkan.

📌 Berikut alur untuk mulai mencatat performa trading kamu:

1️⃣ Gunakan perintah */setposition* untuk mencatat saat kamu masuk posisi (BUY/SELL).

2️⃣ Setelah keluar dari posisi, klik tombol *Exit Posisi* dan isi form exit (harga keluar, tanggal, dll).

3️⃣ Setelah posisi ditutup, kamu bisa menggunakan perintah */report* untuk melihat performa trading kamu.

💡 Data baru akan muncul di report setelah kamu menyelesaikan langkah di atas minimal 1 kali.`
}

func (t *TelegramBotService) formatMessageReport(positions []models.StockPositionEntity) string {
	sb := &strings.Builder{}
	// header
	sb.WriteString("📊 <b>Trading Report</b>\n")
	sb.WriteString("Laporan ini menampilkan ringkasan performa dari posisi trading yang sudah selesai. Gunakan sebagai bahan evaluasi untuk strategi swing trading kamu.\n")

	sbBody := &strings.Builder{}
	sbBody.WriteString("\n\n🔎 Detail Saham:")

	countWin := 0
	countLose := 0
	countPnL := 0.0
	for _, position := range positions {
		pnl := ((*position.ExitPrice - position.BuyPrice) / position.BuyPrice) * 100
		icon := "🔴"
		countPnL += pnl
		if pnl > 0 {
			icon = "🟢"
			countWin++
		} else {
			countLose++
		}
		sbBody.WriteString(fmt.Sprintf("\n- $%s <i>(%s-%s)</i>", position.StockCode, position.BuyDate.Format("01/02"), position.ExitDate.Format("01/02")))
		sbBody.WriteString(fmt.Sprintf("\n		%s PnL: %+.2f%%", icon, pnl))
		sbBody.WriteString(fmt.Sprintf("\n		💰 Buy: %d | Exit: %d", int(position.BuyPrice), int(*position.ExitPrice)))
	}

	sbSummary := &strings.Builder{}
	sbSummary.WriteString(fmt.Sprintf("\n🟢 <b>Win</b>: %d | 🔴 Lose: %d", countWin, countLose))
	sbSummary.WriteString(fmt.Sprintf("\n📈 <b>Total PnL</b>: %+.2f%%", countPnL))
	sbSummary.WriteString(fmt.Sprintf("\n🏆 <b>Win Rate</b>: %.2f%%", float64(countWin)/float64(len(positions))*100))

	result := fmt.Sprintf("%s%s%s", sb.String(), sbSummary.String(), sbBody.String())
	return result
}

func (t *TelegramBotService) formatMessageTopNewsList(newsList []models.TopNewsCustomResult) string {
	sb := &strings.Builder{}
	sb.WriteString(fmt.Sprintf("📈 <b>Top News Saham Hari Ini (%s)</b>\n", utils.TimeNowWIB().Format("02/01 15:04")))

	for _, news := range newsList {
		sb.WriteString(fmt.Sprintf("\n<i>📅 %s | 🌐  %s</i>\n", news.PublishedAt.Format("2006-01-02"), utils.ExtractDomain(news.Link)))
		sb.WriteString(fmt.Sprintf("<b>%s</b>\n", utils.TruncateTitle(news.Title, 80)))
		sb.WriteString(utils.TruncateTitle(news.Summary, 300))
		sb.WriteString("\n")
		if len(news.StockCodes) > 0 {
			sb.WriteString(fmt.Sprintf(" 📊 Saham: %s\n", strings.Join(news.StockCodes, ", ")))
		}
		link := fmt.Sprintf("<a href='%s'>Selengkapnya</a>", news.Link)
		sb.WriteString(fmt.Sprintf(" 🔗 %s\n", link))

	}
	sb.WriteString("\n")

	return sb.String()
}
