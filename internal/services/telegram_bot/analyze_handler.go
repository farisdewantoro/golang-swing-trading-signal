package telegram_bot

import (
	"context"
	"encoding/json"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleAnalyze(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID

	// Start a new conversation for analysis
	t.userStates[userID] = StateWaitingAnalyzeSymbol
	t.userAnalysisPositionData[userID] = &models.RequestAnalysisPositionData{} // Reuse this to store the symbol

	return c.Send("Silakan masukkan simbol saham yang ingin Anda analisis (contoh: BBCA, ANTM).")
}

func (t *TelegramBotService) handleGeneralAnalysis(ctx context.Context, c telebot.Context) error {
	symbol := c.Text()

	stopChan := make(chan struct{})

	t.ResetUserState(c.Sender().ID)

	// Mulai loading animasi
	msg := t.showLoadingFlowAnalysis(c, stopChan)

	utils.SafeGo(func() {
		newCtx, cancel := context.WithTimeout(t.ctx, t.config.TimeoutDuration)
		defer cancel()

		stockSignal, err := t.stockService.GetLatestStockSignal(newCtx, models.GetStockBuySignalParam{
			After:     utils.TimeNowWIB().Add(-t.tradingConfig.GetLatestSignalBefore),
			StockCode: symbol,
		})

		if err != nil {
			close(stopChan)
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to get stock signal")

			// Send error message
			_, err = t.telegramRateLimiter.Send(newCtx, c, fmt.Sprintf("❌ Failed to get stock signal %s: %s", symbol, err.Error()))
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		if len(stockSignal) == 0 {
			defer close(stopChan)
			t.logger.WithField("symbol", symbol).Warn("No stock signal found")
			t.stockService.RequestStockAnalyzer(newCtx, &models.RequestStockAnalyzer{
				TelegramID: c.Sender().ID,
				StockCode:  symbol,
				NotifyUser: true,
			})

			if _, err := t.telegramRateLimiter.Edit(newCtx, c, msg, fmt.Sprintf(messageAnalysisNotAvailable, symbol), &telebot.SendOptions{
				ParseMode: telebot.ModeMarkdown,
			}); err != nil {
				t.logger.WithError(err).Error("Failed to edit message")
			}

			return
		}

		var analysis models.IndividualAnalysisResponseMultiTimeframe
		if err := json.Unmarshal([]byte(stockSignal[0].Data), &analysis); err != nil {
			close(stopChan)
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to unmarshal analysis")
			// Send error message
			_, err = t.telegramRateLimiter.Send(newCtx, c, fmt.Sprintf("❌ Gagal parse data %s", symbol))
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		// Format analysis message
		analysisMessage := t.FormatAnalysisMessage(&analysis)

		// Stop animasi loading
		close(stopChan)

		// Ganti pesan loading dengan hasil analisa
		_, err = t.telegramRateLimiter.Edit(newCtx, c, msg, analysisMessage, &telebot.SendOptions{
			ParseMode: telebot.ModeHTML,
		})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send analysis message")
		}

	})

	return nil
}
