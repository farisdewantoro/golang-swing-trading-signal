package telegram_bot

import (
	"context"
	"fmt"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleBtnNewsStockPosition(ctx context.Context, c telebot.Context) error {
	stockCode := c.Data()
	summary, err := t.stockService.GetLastStockNewsSummary(ctx, t.config.FeatureNewsMaxAgeInDays, stockCode)
	if err != nil {
		return err
	}

	if summary == nil {
		t.telegramRateLimiter.Edit(ctx, c, c.Message(), fmt.Sprintf("Maaf, saat ini belum tersedia berita untuk saham %s coba lagi nanti.", stockCode), telebot.ModeMarkdown)
		return nil
	}
	msg := t.formatMessageNewsSummary(summary)
	_, err = t.telegramRateLimiter.Edit(ctx, c, c.Message(), msg, telebot.ModeMarkdown)
	if err != nil {
		return err
	}
	return nil
}
