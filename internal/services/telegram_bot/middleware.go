package telegram_bot

import (
	"golang-swing-trading-signal/internal/utils"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) RegisterMiddleware() {
	t.bot.Use(t.LoggingMiddleware)
	t.bot.Use(t.RecoverMiddleware())
	t.bot.Use(t.DeleteUserStateOnErrorMiddleware())
}

func (t *TelegramBotService) LoggingMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		now := utils.TimeNowWIB()
		userID := c.Sender().ID
		err := next(c)
		t.logger.Debug("Processed message from user", logrus.Fields{
			"timestamp": now,
			"user_id":   userID,
			"error":     err,
			"duration":  time.Since(now),
			"message":   c.Message().Text,
		})

		return err
	}
}

func (t *TelegramBotService) RecoverMiddleware() telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					t.logger.Error("Recovered from panic: ", logrus.Fields{
						"user_id": c.Sender().ID,
						"error":   r,
						"message": c.Message().Text,
					})
					_ = c.Send("⚠️ Terjadi kesalahan internal. Mohon coba lagi nanti.")
				}
			}()
			return next(c)
		}
	}
}

func (t *TelegramBotService) IsOnConversationMiddleware() telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) (err error) {
			if _, inConversation := t.userStates[c.Sender().ID]; inConversation {
				t.handleCancel(c)
			}
			return next(c)
		}
	}
}

func (t *TelegramBotService) DeleteUserStateOnErrorMiddleware() telebot.MiddlewareFunc {
	return func(next telebot.HandlerFunc) telebot.HandlerFunc {
		return func(c telebot.Context) (err error) {
			defer func() {
				if err != nil {
					t.ResetUserState(c.Sender().ID)
				}
			}()
			return next(c)
		}
	}
}
