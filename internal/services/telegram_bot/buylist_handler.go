package telegram_bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleBuyList(ctx context.Context, c telebot.Context) error {

	parts := strings.Split(dataInputTimeFrameEntry, "|")

	if len(parts) != 3 {
		return t.telegramRateLimiter.EditWithoutMsg(ctx, c, commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	interval, rng := parts[1], parts[2]

	stocks, err := t.stockService.GetStocks(ctx)
	if err != nil {
		return t.telegramRateLimiter.EditWithoutMsg(ctx, c, commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}

	if len(stocks) == 0 {
		return t.telegramRateLimiter.EditWithoutMsg(ctx, c, "Tidak ada data saham", &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}

	utils.SafeGo(func() {
		var cancel context.CancelFunc
		t.mu.Lock()
		if prevCancel, exists := t.userCancelFuncs[c.Sender().ID]; exists {
			prevCancel()
		}
		newCtx, cancel := context.WithTimeout(t.ctx, t.config.TimeoutBuyListDuration)
		t.userCancelFuncs[c.Sender().ID] = cancel
		t.mu.Unlock()

		var wg sync.WaitGroup
		wg.Add(1)
		defer func() {
			wg.Wait()
			t.mu.Lock()
			delete(t.userCancelFuncs, c.Sender().ID)
			t.mu.Unlock()
			cancel()
		}()

		msgRoot, err := t.telegramRateLimiter.Send(newCtx, c, `🧠 Sedang menganalisis saham terbaik untuk dibeli...`, telebot.ModeMarkdown)
		if err != nil {
			t.logger.WithError(err).Error("Failed to send loading message")
			t.telegramRateLimiter.EditWithoutMsg(newCtx, c, commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
			return
		}

		buyListResultMsg := &strings.Builder{}

		msgHeader := &strings.Builder{}
		msgHeader.WriteString(fmt.Sprintf(`
📊 Analisis Saham Sedang Berlangsung...
📌 Interval: %s
📅 Time Range: %s
`, strings.ToUpper(interval), strings.ToUpper(rng)))

		progressCh := make(chan Progress, len(stocks)+1)
		t.showProgressBarWithChannel(newCtx, c, msgRoot, progressCh, len(stocks), &wg)

		progressCh <- Progress{Index: 0, StockCode: stocks[0].Code, Header: msgHeader.String()}

		buyCount := 0
		for idx, stock := range stocks {

			if stop, err := utils.ShouldStopCtx(newCtx, t.logger); stop {
				switch {
				case errors.Is(err, context.Canceled):
					t.telegramRateLimiter.SendWithoutLimit(newCtx, c, "✅ Proses analisa berhasil dihentikan.")
				case errors.Is(err, context.DeadlineExceeded):
					t.telegramRateLimiter.SendWithoutLimit(newCtx, c, "⏰ Proses analisa dihentikan karena timeout.")
				}
				return
			}

			fields := logrus.Fields{
				"symbol": stock.Code,
				"index":  idx + 1,
				"total":  len(stocks),
			}

			t.logger.Info("Buy list - Analisa saham", fields)
			intervalTime, err := utils.GetTimeBefore(interval)
			if err != nil {
				t.logger.WithError(err).WithField("symbol", stock.Code).Error("Buy list - Gagal parse data interval duration")
				buyListResultMsg.WriteString(commonMessageInternalError)
				progressCh <- Progress{Index: idx + 1, StockCode: stock.Code, Content: buyListResultMsg.String(), Header: msgHeader.String()}
				continue
			}
			stockSignals, err := t.stockService.GetLatestStockSignal(newCtx, models.GetStockBuySignalParam{
				Interval:  interval,
				Range:     rng,
				After:     intervalTime,
				StockCode: stock.Code,
				ReqAnalyzer: &models.RequestStockAnalyzer{
					TelegramID: c.Sender().ID,
					NotifyUser: false,
				},
			})

			if err != nil {
				t.logger.WithError(err).WithField("symbol", stock.Code).Error("Buy list - Gagal mengambil data")
				buyListResultMsg.WriteString(fmt.Sprintf("*%d. %s* - ❌ Gagal mengambil data\n", idx+1, stock.Code))
				progressCh <- Progress{Index: idx + 1, StockCode: stock.Code, Content: buyListResultMsg.String(), Header: msgHeader.String()}
				continue
			}

			if len(stockSignals) == 0 {
				t.logger.Warn("Buy list - Tidak ditemukan sinyal BUY", fields)
				buyListResultMsg.WriteString(fmt.Sprintf("*%d. %s* - ❌ Saat ini data tidak tersedia\n", idx+1, stock.Code))
				progressCh <- Progress{Index: idx + 1, StockCode: stock.Code, Content: buyListResultMsg.String(), Header: msgHeader.String()}
				continue
			}

			stockSignal := stockSignals[0]

			if stockSignal.Signal != "BUY" {
				t.logger.Debug("Buy list - Tidak direkomendasikan untuk BUY", fields)
				continue
			}

			buyCount++
			if buyCount == 1 {
				msgHeader.WriteString("\n📈 Ditemukan sinyal BUY:")
			}
			var analysis models.IndividualAnalysisResponse
			if err := json.Unmarshal([]byte(stockSignal.Data), &analysis); err != nil {
				t.logger.WithError(err).WithField("symbol", stock.Code).Error("Failed to unmarshal analysis")
				buyListResultMsg.WriteString(fmt.Sprintf("*%d. %s* - ❌ Gagal parse data\n", idx+1, stock.Code))
				progressCh <- Progress{Index: idx + 1, StockCode: stock.Code, Content: buyListResultMsg.String(), Header: msgHeader.String()}
				continue
			}
			newBuyListMsg := t.formatMessageBuyList(buyCount, &analysis)

			buyListResultMsg.WriteString(newBuyListMsg.String())
			progressCh <- Progress{Index: idx + 1, StockCode: stock.Code, Content: buyListResultMsg.String(), Header: msgHeader.String()}

		}

		if buyCount > 0 {
			msgHeader.Reset()
			msgHeader.WriteString(fmt.Sprintf("📈 Berikut saham %d yang direkomendasikan untuk BUY:", buyCount))
			msgFooter := fmt.Sprintf(`

📌 Interval: %s  
📅 Time Range: %s 

🧠 Rekomendasi berdasarkan analisis teknikal dan sentimen pasar

`, strings.ToUpper(interval), strings.ToUpper(rng))
			buyListResultMsg.WriteString(msgFooter)
			progressCh <- Progress{Index: len(stocks), StockCode: stocks[len(stocks)-1].Code, Content: buyListResultMsg.String(), Header: msgHeader.String()}
		} else {
			msgHeader.Reset()
			msgHeader.WriteString("❌ Tidak ditemukan sinyal BUY hari ini.")
			msgFooter := fmt.Sprintf(`

📌 Interval: %s  
📅 Time Range: %s 

Coba lagi besok atau gunakan filter /analyze untuk menemukan peluang baru.
			
			`, strings.ToUpper(interval), strings.ToUpper(rng))
			buyListResultMsg.WriteString(msgFooter)
			progressCh <- Progress{Index: len(stocks), StockCode: stocks[len(stocks)-1].Code, Content: buyListResultMsg.String(), Header: msgHeader.String()}
		}

		close(progressCh)
	})
	return nil
}

func (t *TelegramBotService) handleBtnCancelBuyListAnalysis(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID

	t.ResetUserState(userID)

	return t.telegramRateLimiter.Respond(ctx, c, &telebot.CallbackResponse{
		Text: "❌ Analisis dibatalkan.",
	})
}
