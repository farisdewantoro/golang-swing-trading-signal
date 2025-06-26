package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleNews(ctx context.Context, c telebot.Context) error {
	menu := &telebot.ReplyMarkup{}

	btnFind := menu.Data(btnActionNewsFind.Text, btnActionNewsFind.Unique)
	btnTopNews := menu.Data(btnActionTopNews.Text, btnActionTopNews.Unique)
	btnDeleteForCancel := menu.Data(btnCancelGeneral.Text, btnDeleteMessage.Unique)

	menu.Inline(
		menu.Row(btnFind),
		menu.Row(btnTopNews),
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
	_, err := t.telegramRateLimiter.Edit(ctx, c, c.Message(), msg, telebot.ModeMarkdown)
	return err
}

func (t *TelegramBotService) handleNewsFindConversation(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	state := t.userStates[userID]

	switch state {
	case StateWaitingNewsFindSymbol:
		return t.handleNewsFind(ctx, c)
	case StateWaitingNewsFindSendSummaryConfirmation:
		return c.Send("👆 Silakan pilih salah satu opsi di atas, atau kirim /cancel untuk membatalkan.")
	default:
		return t.handleCancel(c)
	}
}

func (t *TelegramBotService) handleNewsFind(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	text := strings.ToUpper(c.Text())
	state := t.userStates[userID]
	var err error

	if state != StateWaitingNewsFindSymbol {
		return t.handleCancel(c)
	}

	age := t.config.FeatureNewsMaxAgeInDays

	defer func() {
		if err != nil {
			t.ResetUserState(userID)
		}
	}()

	news, err := t.stockService.GetTopNews(ctx, models.StockNewsQueryParam{
		StockCodes:       []string{text},
		Limit:            t.config.FeatureNewsLimitStockNews,
		MaxNewsAgeInDays: age,
	})
	if err != nil {
		return err
	}

	if len(news) == 0 {
		_, err = t.telegramRateLimiter.Send(ctx, c, "Maaf, tidak ada data berita ditemukan untuk kode saham tersebut.", telebot.ModeMarkdown)
		return err
	}

	msg := t.formatMessageNewsList(news, age)
	_, err = t.telegramRateLimiter.Send(ctx, c, msg, telebot.ModeMarkdown)
	if err != nil {
		return err
	}

	summary, err := t.stockService.GetLastStockNewsSummary(ctx, age, text)
	if err != nil {
		return err
	}

	if summary == nil {
		t.ResetUserState(userID)
		return nil
	}

	confirmSendSummaryMsg := fmt.Sprintf("📚 Saya telah menampilkan beberapa berita penting untuk saham %s. \n\nIngin melihat ringkasan analisis, sentimen, dan saran terkait saham ini?", text)

	menu := &telebot.ReplyMarkup{}
	btnConfirm := menu.Data("✅ Iya", btnNewsConfirmSendSummary.Unique, fmt.Sprintf("%s|%t", text, true))
	btnReject := menu.Data("❌ Tidak", btnNewsConfirmSendSummary.Unique, fmt.Sprintf("%s|%t", text, false))
	menu.Inline(menu.Row(btnConfirm, btnReject))

	time.Sleep(1 * time.Second)
	_, err = t.telegramRateLimiter.Send(ctx, c, confirmSendSummaryMsg, menu, telebot.ModeMarkdown)
	if err != nil {
		return err
	}

	t.userStates[userID] = StateWaitingNewsFindSendSummaryConfirmation
	return nil
}

func (t *TelegramBotService) handleBtnNewsConfirmSendSummary(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	state := t.userStates[userID]
	if state != StateWaitingNewsFindSendSummaryConfirmation {
		return t.handleCancel(c)
	}

	defer t.ResetUserState(userID)

	data := strings.Split(c.Data(), "|")
	if len(data) != 2 {
		return t.handleBtnDeleteMessage(ctx, c)
	}

	// parse bool
	confirm := data[1] == "true"
	if !confirm {
		return t.handleBtnDeleteMessage(ctx, c)
	}

	summary, err := t.stockService.GetLastStockNewsSummary(ctx, t.config.FeatureNewsMaxAgeInDays, data[0])
	if err != nil {
		return err
	}

	if summary == nil {
		t.telegramRateLimiter.Edit(ctx, c, c.Message(), "Maaf, tidak ada data berita ditemukan untuk kode saham tersebut.", telebot.ModeMarkdown)
		return nil
	}
	msg := t.formatMessageNewsSummary(summary)
	_, err = t.telegramRateLimiter.Edit(ctx, c, c.Message(), msg, telebot.ModeMarkdown)
	if err != nil {
		return err
	}

	return nil
}
