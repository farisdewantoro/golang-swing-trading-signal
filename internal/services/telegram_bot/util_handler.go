package telegram_bot

import (
	"context"
	"time"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) WithContext(handler func(ctx context.Context, c telebot.Context) error) func(c telebot.Context) error {
	return func(c telebot.Context) error {
		ctx, cancel := context.WithTimeout(t.ctx, 5*time.Minute)
		defer cancel()

		return handler(ctx, c)
	}
}

func (t *TelegramBotService) handleBtnDeleteMessage(ctx context.Context, c telebot.Context) error {
	c.Edit("✅ Pesan akan dihapus....")
	time.Sleep(1 * time.Second)
	return c.Delete()
}

func (t *TelegramBotService) handleCancel(c telebot.Context) error {
	userID := c.Sender().ID

	t.ResetUserState(userID)

	// Check if user is in any conversation state
	if state, ok := t.userStates[userID]; ok && state != StateIdle {
		return c.Send("✅ Percakapan dibatalkan.")
	}

	return nil

}

func (t *TelegramBotService) handleBtnCancel(ctx context.Context, c telebot.Context) error {
	return t.handleCancel(c)
}
