package telegram_bot

import (
	"context"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleReport(ctx context.Context, c telebot.Context) error {
	telegramID := c.Sender().ID

	param := models.StockPositionQueryParam{
		TelegramIDs: []int64{telegramID},
		IsExit:      utils.ToPointer(true),
		IsActive:    false,
	}
	stockPositions, err := t.stockService.GetStockPosition(ctx, param)
	if err != nil {
		t.logger.WithError(err).Error("Failed to get stock positions for report")
		_, errSend := t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		if errSend != nil {
			t.logger.WithError(errSend).Error("Failed to send internal error message")
		}
		return err
	}

	if len(stockPositions) == 0 {
		_, errSend := t.telegramRateLimiter.Send(ctx, c, t.formatMessageReportNotExits(), telebot.ModeMarkdown)
		if errSend != nil {
			t.logger.WithError(errSend).Error("Failed to send no exit positions message")
		}
		return errSend
	}

	reportMessage := t.formatMessageReport(stockPositions)
	_, errSend := t.telegramRateLimiter.Send(ctx, c, reportMessage, telebot.ModeHTML)
	if errSend != nil {
		t.logger.WithError(errSend).Error("Failed to send report message")
		return errSend
	}

	return nil
}
