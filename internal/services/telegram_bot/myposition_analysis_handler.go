package telegram_bot

import (
	"context"
	"encoding/json"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleBtnBackDetailStockPosition(ctx context.Context, c telebot.Context) error {
	symbol := c.Data()
	return t.handleBtnBackDetailStockPositionWithParam(ctx, c, &symbol, nil)
}

func (t *TelegramBotService) handleBtnTimeframeStockPositionMonitoring(ctx context.Context, c telebot.Context) error {
	symbol := c.Data() // The symbol is passed as data

	stopChan := make(chan struct{})

	// Mulai loading animasi
	msg := t.showLoadingFlowAnalysis(c, stopChan)

	utils.SafeGo(func() {
		newCtx, cancel := context.WithTimeout(t.ctx, t.config.TimeoutDuration)
		defer cancel()

		positions, err := t.stockService.GetLatestStockPositionMonitoring(newCtx, models.GetStockPositionMonitoringParam{
			StockCode:  symbol,
			TelegramID: c.Sender().ID,
			Limit:      1,
			AfterTime:  utils.TimeNowWIB().Add(-t.tradingConfig.GetLatestSignalBefore),
		})
		if err != nil {
			close(stopChan)
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to get stock position monitoring")
			_, err := t.telegramRateLimiter.Edit(newCtx, c, msg, commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		if len(positions) == 0 {
			defer close(stopChan)
			t.logger.WithField("symbol", symbol).Warn("No stock position monitoring found")
			t.stockService.RequestStockPositionMonitoring(newCtx, &models.RequestStockPositionMonitoring{
				TelegramID:     c.Sender().ID,
				StockCode:      symbol,
				SendToTelegram: true,
			})

			if _, err := t.telegramRateLimiter.Edit(newCtx, c, msg, fmt.Sprintf(messageAnalysisNotAvailable, symbol), &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			}); err != nil {
				t.logger.WithError(err).Error("Failed to edit message")
			}
			return
		}

		close(stopChan)
		position := positions[0]

		var stockMonitoring models.PositionMonitoringResponseMultiTimeframe
		if err := json.Unmarshal([]byte(position.Data), &stockMonitoring); err != nil {
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to unmarshal stock monitoring")
			// Send error message
			_, err := t.telegramRateLimiter.Send(newCtx, c, fmt.Sprintf("❌ Gagal parse data %s", symbol))
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		// Format position monitoring message
		message := t.FormatPositionMonitoringMessage(&stockMonitoring)

		// Send the position monitoring results
		_, err = t.telegramRateLimiter.Edit(newCtx, c, msg, message, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML,
		})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send error message")
		}

	})

	return nil
}
