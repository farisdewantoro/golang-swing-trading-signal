package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
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

	defer t.ResetUserState(userID)

	// Check if user is in any conversation state
	if state, ok := t.userStates[userID]; ok && state != StateIdle {
		return c.Send("✅ Percakapan dibatalkan.")
	}

	return nil

}

func (t *TelegramBotService) handleBtnCancel(ctx context.Context, c telebot.Context) error {
	return t.handleCancel(c)
}

func (t *TelegramBotService) getLastMarketPrice(ctx context.Context, stockCodes []string) (map[string]models.RedisLastPrice, error) {
	lastPrices := make(map[string]models.RedisLastPrice)
	cmds := make(map[string]*redis.MapStringStringCmd)

	pipe := t.redisClient.Pipeline()
	for _, stockCode := range stockCodes {
		key := fmt.Sprintf("last_price:%s", stockCode)
		cmds[stockCode] = pipe.HGetAll(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		t.logger.WithError(err).Error("Failed to execute pipeline for last market prices")
		return nil, fmt.Errorf("failed to get last market prices: %w", err)
	}

	// Baca hasil
	for symbol, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			fmt.Printf("Error get %s: %v\n", symbol, err)
			continue
		}

		if len(data) == 0 {
			t.logger.Warnf("No last price data found for stock code: %s", symbol)
			continue
		}

		price, err := strconv.Atoi(data["price"])
		if err != nil {
			t.logger.WithError(err).Errorf("Failed to parse price for stock code: %s", symbol)
			continue
		}
		timestamp, err := strconv.ParseInt(data["timestamp"], 10, 64)
		if err != nil {
			t.logger.WithError(err).Errorf("Failed to parse timestamp for stock code: %s", symbol)
			continue
		}

		lastPrices[symbol] = models.RedisLastPrice{
			StockCode: symbol,
			Price:     float64(price),
			Timestamp: timestamp,
		}

	}

	return lastPrices, nil
}
