package telegram_bot

import (
	"context"
	"encoding/json"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strings"

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
	return t.handleGeneralAnalysisWithParam(ctx, c, symbol, false)
}
func (t *TelegramBotService) handleGeneralAnalysisWithParam(ctx context.Context, c telebot.Context, symbol string, isEdit bool) error {
	userID := c.Sender().ID

	t.ResetUserState(userID)

	menu := &telebot.ReplyMarkup{}

	msg := fmt.Sprintf("📊 Analisa Saham: *$%s*\n\nSilakan pilih strategi analisa yang paling sesuai dengan kondisimu saat ini 👇", symbol)
	btnMain := menu.Data("🔍 Analisa", btnInputTimeFrameStockAnalysis.Unique, symbol)
	btnDelete := menu.Data(btnDeleteMessage.Text, btnDeleteMessage.Unique, symbol)
	menu.Inline(
		menu.Row(btnMain),
		menu.Row(btnDelete),
	)

	if isEdit {
		return c.Edit(msg, menu, telebot.ModeMarkdown)
	}
	return c.Send(msg, menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnGeneralAnalysis(ctx context.Context, c telebot.Context) error {

	data := c.Data()

	parts := strings.Split(data, "|")
	if len(parts) != 3 {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	symbol, interval, rng := parts[0], parts[1], parts[2]

	stopChan := make(chan struct{})

	// Mulai loading animasi
	msg := t.showLoadingFlowAnalysis(c, stopChan)

	utils.SafeGo(func() {
		newCtx, cancel := context.WithTimeout(t.ctx, t.config.TimeoutDuration)
		defer cancel()

		intervalTime, err := utils.GetTimeBefore(interval)
		if err != nil {
			close(stopChan)
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to parse interval")
			if err := c.Send(commonMessageInternalError); err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		stockSignal, err := t.stockService.GetLatestStockSignal(newCtx, models.GetStockBuySignalParam{
			Interval:  interval,
			Range:     rng,
			After:     intervalTime,
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
				Interval:   interval,
				Range:      rng,
				NotifyUser: true,
			})

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

func (t *TelegramBotService) handleBtnNotesTimeFrameStockAnalysis(ctx context.Context, c telebot.Context) error {
	symbol := c.Data() // The symbol is passed as data
	menu := &telebot.ReplyMarkup{}
	btnMain := menu.Data("🔍 Analisa", btnInputTimeFrameStockAnalysis.Unique, symbol)
	btnBack := menu.Data(btnBackStockAnalysis.Text, btnBackStockAnalysis.Unique, symbol)
	menu.Inline(
		menu.Row(btnMain),
		menu.Row(btnBack),
	)
	return c.Edit(t.FormatNotesTimeFrameStockMessage(), menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnBackStockAnalysis(ctx context.Context, c telebot.Context) error {
	return t.handleGeneralAnalysisWithParam(ctx, c, c.Data(), true)
}
