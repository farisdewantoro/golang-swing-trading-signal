package telegram_bot

import (
	"context"
	"errors"
	"fmt"
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

		msgHeaderInProgress := &strings.Builder{}
		msgHeaderInProgress.WriteString(fmt.Sprintf(`
📊 Analisis Saham Sedang Berlangsung...
📌 Interval: %s
📅 Time Range: %s
`, strings.ToUpper(interval), strings.ToUpper(rng)))

		progressCh := make(chan Progress, len(stocks)+1)
		t.showProgressBarWithChannel(newCtx, c, msgRoot, progressCh, len(stocks), &wg)

		progressCh <- Progress{Index: 0, StockCode: stocks[0].Code, Header: msgHeaderInProgress.String()}

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

			t.logger.Info("Buy list - Analisa saham", logrus.Fields{
				"symbol": stock.Code,
				"index":  idx + 1,
				"total":  len(stocks),
			})

			analysis, err := t.analyzer.AnalyzeStock(newCtx, stock.Code, interval, rng)
			if err != nil {
				t.logger.WithError(err).WithField("symbol", stock.Code).Error("Failed to analyze stock")
				buyListResultMsg.WriteString(fmt.Sprintf("*%d. %s* - ❌ Gagal analisa\n", idx+1, stock.Code))
				progressCh <- Progress{Index: idx + 1, StockCode: stock.Code, Content: buyListResultMsg.String(), Header: msgHeaderInProgress.String()}
				continue
			}

			if strings.ToUpper(analysis.Signal) != "BUY" {
				t.logger.Debug("Buy list - Tidak direkomendasikan untuk BUY", logrus.Fields{
					"symbol": stock.Code,
					"index":  idx + 1,
					"total":  len(stocks),
				})
				progressCh <- Progress{Index: idx + 1, StockCode: stock.Code, Content: buyListResultMsg.String(), Header: msgHeaderInProgress.String()}
				continue
			}
			buyCount++
			if buyCount == 1 {
				msgHeaderInProgress.WriteString("\n📈 Ditemukan sinyal BUY:")
			}
			newBuyListMsg := t.formatMessageBuyList(buyCount, analysis)

			buyListResultMsg.WriteString(newBuyListMsg.String())
			progressCh <- Progress{Index: idx + 1, StockCode: stock.Code, Content: buyListResultMsg.String(), Header: msgHeaderInProgress.String()}

		}

		if buyCount > 0 {
			msgHeaderInProgress.Reset()
			msgHeaderInProgress.WriteString(fmt.Sprintf("📈 Berikut saham %d yang direkomendasikan untuk BUY:", buyCount))
			msgFooter := fmt.Sprintf(`

📌 Interval: %s  
📅 Time Range: %s 

🧠 Rekomendasi berdasarkan analisis teknikal dan sentimen pasar

`, strings.ToUpper(interval), strings.ToUpper(rng))
			buyListResultMsg.WriteString(msgFooter)
			progressCh <- Progress{Index: len(stocks), StockCode: stocks[len(stocks)-1].Code, Content: buyListResultMsg.String(), Header: msgHeaderInProgress.String()}
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
