package telegram_bot

import (
	"context"
	"golang-swing-trading-signal/internal/utils"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

// handleBuyList handles /buylist command - analyzes all stocks and shows buy list
func (t *TelegramBotService) handleBuyList(ctx context.Context, c telebot.Context) error {

	startTime := utils.TimeNowWIB()
	stopChan := make(chan struct{})
	parts := strings.Split(dataInputTimeFrameEntry, "|")
	if len(parts) != 3 {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	interval, rng := parts[1], parts[2]
	// Mulai loading animasi
	msg := t.showLoadingFlowAnalysis(c, stopChan)

	utils.SafeGo(func() {
		newCtx, cancel := context.WithTimeout(t.ctx, t.config.TimeoutBuyListDuration)
		defer cancel()

		summary, err := t.analyzer.AnalyzeAllStocks(newCtx, t.tradingConfig.StockList, interval, rng)
		if err != nil {
			close(stopChan)
			t.logger.WithError(err).Error("Failed to analyze stock")
			t.bot.Edit(msg, commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
			return
		}

		close(stopChan)

		// Calculate actual time taken
		actualTime := time.Since(startTime)

		// Format buy list summary message
		summaryMessage := t.FormatBuyListSummaryMessage(summary, actualTime)

		// Send the buy list summary results
		_, err = t.bot.Edit(msg, summaryMessage, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML,
		})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send buy list summary message")
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

	})

	return nil
}
