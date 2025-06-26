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
	unrealizedPnLPercentageStr := fmt.Sprintf("(+%.2f)", unrealizedPnLPercentage)

	if unrealizedPnLPercentage < 0 {
		unrealizedPnLPercentageStr = fmt.Sprintf("(%.2f)", unrealizedPnLPercentage)
	}

	daysRemaining := utils.RemainingDays(position.MaxHoldingPeriodDays, position.BuyDate)
	ageDays := int(time.Since(position.BuyDate).Hours() / 24)

	iconAction := "❔"
	if position.Action == "HOLD" {
		iconAction = "🟡"
	} else if position.Action == "SELL" {
		iconAction = "🔴"
	} else if position.Action == "BUY" {
		iconAction = "🟢"
	}

	sb.WriteString(fmt.Sprintf("📊 <b>Position Update: %s</b>\n", position.Symbol))
	sb.WriteString(fmt.Sprintf("💰 Buy: $%d\n", int(position.BuyPrice)))
	sb.WriteString(fmt.Sprintf("📌 Last Price: $%d %s\n", int(position.MarketPrice), unrealizedPnLPercentageStr))
	sb.WriteString(fmt.Sprintf("🎯 TP: $%d | SL: $%d\n", int(position.TargetPrice), int(position.CutLoss)))
	sb.WriteString(fmt.Sprintf("📈 Age: %d days | Remaining: %d days\n\n", ageDays, daysRemaining))

	// Recommendation
	sb.WriteString("💡 <b>Recommendation:</b>\n")
	sb.WriteString(fmt.Sprintf("• %s Action: %s\n", iconAction, position.Action))
	sb.WriteString(fmt.Sprintf("• 🎯 Target Price: $%d\n", int(position.ExitTargetPrice)))
	sb.WriteString(fmt.Sprintf("• 🛡 Stop Loss: $%d\n", int(position.ExitCutLoss)))
	sb.WriteString(fmt.Sprintf("• 🔁 Risk/Reward Ratio: %.2f\n", position.RiskRewardRatio))
	sb.WriteString(fmt.Sprintf("• 📊 Confidence: %d%%\n\n", position.ConfidenceLevel))
	// Reasoning
	sb.WriteString(fmt.Sprintf("🧠 <b>Reasoning:</b>\n %s\n\n", position.Reasoning))
	if len(position.ExitConditions) > 0 {
		sb.WriteString("💡 <b>Exit Conditions:</b>\n")
		for _, condition := range position.ExitConditions {
			sb.WriteString(fmt.Sprintf("• %s\n", condition))
		}
	}

	// Technical Analysis
	// Technical Analysis Summary
	sb.WriteString("\n<b>📉 Ringkasan Per Timeframe:</b>\n")
	sb.WriteString(fmt.Sprintf("• 1D: %s\n", position.TimeframeSummaries.TimeFrame1D))
	sb.WriteString(fmt.Sprintf("• 4H: %s\n", position.TimeframeSummaries.TimeFrame4H))
	sb.WriteString(fmt.Sprintf("• 1H: %s\n", position.TimeframeSummaries.TimeFrame1H))

	// News Summary
	sb.WriteString("\n📰 <b>News Analysis:</b>\n")
	if position.NewsConfidenceScore > 50 {
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

	sb.WriteString(fmt.Sprintf("📊 <b>Analysis for %s</b>\n", analysis.Symbol))
	sb.WriteString(fmt.Sprintf("🎯 Signal: <b>%s</b>\n", analysis.Action))
	sb.WriteString(fmt.Sprintf("📌 Last Price: %d (%s)\n\n", int(analysis.MarketPrice), analysis.AnalysisDate.Format("01-02 15:04")))

	// Recommendation
	if analysis.Action != "HOLD" {
		sb.WriteString("💡 <b>Recommendation:</b>\n")
		sb.WriteString(fmt.Sprintf("• 💵 Buy Price: $%d\n", int(analysis.BuyPrice)))
		sb.WriteString(fmt.Sprintf("• 🎯 Target Price: $%d\n", int(analysis.TargetPrice)))
		sb.WriteString(fmt.Sprintf("• 🛡 Stop Loss: $%d\n", int(analysis.CutLoss)))
		sb.WriteString(fmt.Sprintf("• 🔁 Risk/Reward Ratio: %.2f\n", analysis.RiskRewardRatio))
	}
	sb.WriteString(fmt.Sprintf("• 📊 Confidence: %d%%\n", analysis.ConfidenceLevel))
	// Reasoning
	sb.WriteString(fmt.Sprintf("\n🧠 <b>Reasoning:</b>\n %s\n\n", analysis.Reasoning))

	// Technical Analysis Summary
	sb.WriteString("<b>📉 Ringkasan Per Timeframe:</b>\n")
	sb.WriteString(fmt.Sprintf("• 1D: %s\n", analysis.TimeframeSummaries.TimeFrame1D))
	sb.WriteString(fmt.Sprintf("• 4H: %s\n", analysis.TimeframeSummaries.TimeFrame4H))
	sb.WriteString(fmt.Sprintf("• 1H: %s\n", analysis.TimeframeSummaries.TimeFrame1H))

	// News Summary
	sb.WriteString("\n📰 <b>News Analysis:</b>\n")
	if analysis.NewsConfidenceScore > 50 {
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

	sb := strings.Builder{}
	sb.WriteString("```\n")
	sb.WriteString("📊 Monitoring Saham\n\n")
	sb.WriteString(fmt.Sprintf("📦 %s\n", position.StockCode))
	sb.WriteString("────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("💰 Harga Beli   : %.2f\n", position.BuyPrice))
	sb.WriteString(fmt.Sprintf("🎯 Target Jual  : %.2f (+%.1f%%)\n", position.TakeProfitPrice, gain))
	sb.WriteString(fmt.Sprintf("🛑 Stop Loss    : %.2f (−%.1f%%)\n", position.StopLossPrice, loss))
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

func (t *TelegramBotService) FormatMyPositionListMessage(positions []models.StockPositionEntity) string {
	var sb strings.Builder

	for _, position := range positions {
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

		sb.WriteString(fmt.Sprintf(" 💰 Last Price: %d (%s)\n", int(dataStockMonitoring.MarketPrice), dataStockMonitoring.AnalysisDate.Format("01/02 15:04")))

		iconAction := "🔴"
		switch dataStockMonitoring.Action {
		case "SELL":
			iconAction = "🟢"
		case "HOLD":
			iconAction = "🟡"
		}

		pnl := (dataStockMonitoring.MarketPrice - dataStockMonitoring.BuyPrice) / dataStockMonitoring.BuyPrice * 100
		pnlText := fmt.Sprintf("%.2f%%", pnl)
		if pnl > 0 {
			pnlText = fmt.Sprintf("+%.2f%%", pnl)
		}
		sb.WriteString(fmt.Sprintf(" 📈 PnL: %s\n", pnlText))
		sb.WriteString(fmt.Sprintf(" %s %s | Confidence: %d/100\n", iconAction, dataStockMonitoring.Action, int(dataStockMonitoring.ConfidenceLevel)))

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
	sb.WriteString(fmt.Sprintf("\n• `$%s` - _(%s)_\n", analysis.Symbol, analysis.AnalysisDate.Format("02/15 15:04")))
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
