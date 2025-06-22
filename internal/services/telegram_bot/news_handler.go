package telegram_bot

import (
	"context"
	"golang-swing-trading-signal/internal/models"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleNews(ctx context.Context, c telebot.Context) error {
	menu := &telebot.ReplyMarkup{}

	btnFind := menu.Data(btnActionNewsFind.Text, btnActionNewsFind.Unique)
	btnAlert := menu.Data(btnActionNewsAlert.Text, btnActionNewsAlert.Unique)
	btnDailySummary := menu.Data(btnActionNewsDailySummary.Text, btnActionNewsDailySummary.Unique)
	btnDeleteForCancel := menu.Data(btnCancelGeneral.Text, btnDeleteMessage.Unique)

	menu.Inline(
		menu.Row(btnFind),
		menu.Row(btnAlert),
		menu.Row(btnDailySummary),
		menu.Row(btnDeleteForCancel),
	)

	t.telegramRateLimiter.Send(ctx, c, t.formatMessageMenuNews(), menu, telebot.ModeMarkdown)
	return nil
}

func (t *TelegramBotService) handleBtnActionNewsFind(ctx context.Context, c telebot.Context) error {
	t.ResetUserState(c.Sender().ID)
	t.userStates[c.Sender().ID] = StateWaitingNewsFindSymbol
	msg := `🔍 Silakan masukkan kode saham yang ingin kamu cari berita
(contoh: BBRI, TLKM, ANTM)
`
	_, err := t.telegramRateLimiter.Send(ctx, c, msg, telebot.ModeMarkdown)
	return err
}

func (t *TelegramBotService) handleNewsFind(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	text := c.Text()
	state := t.userStates[userID]

	if state != StateWaitingNewsFindSymbol {
		return t.handleCancel(c)
	}

	age := 3

	news, err := t.stockService.GetTopNews(ctx, models.StockNewsQueryParam{
		StockCodes:       []string{text},
		Limit:            5,
		MaxNewsAgeInDays: age,
	})
	if err != nil {
		return err
	}

	if len(news) == 0 {
		_, err = t.telegramRateLimiter.Send(ctx, c, "Maaf, tidak ada data berita ditemukan untuk kode saham tersebut.", telebot.ModeMarkdown)
		return err
	}

	defer t.ResetUserState(userID)

	msg := t.formatMessageNewsList(news, age)
	_, err = t.telegramRateLimiter.Send(ctx, c, msg, telebot.ModeMarkdown)
	return err
}
