package telegram_bot

import (
	"context"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleBtnActionTopNews(ctx context.Context, c telebot.Context) error {
	news, err := t.stockService.GetTopNewsGlobal(ctx, t.config.FeatureNewsLimitStockNews, 5)
	if err != nil {
		return err
	}

	msg := t.formatMessageTopNewsList(news)
	_, err = t.telegramRateLimiter.Send(ctx, c, msg, telebot.ModeHTML)
	if err != nil {
		return err
	}
	return nil
}
