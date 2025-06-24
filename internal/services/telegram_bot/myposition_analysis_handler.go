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

func (t *TelegramBotService) handleBtnBackDetailStockPosition(ctx context.Context, c telebot.Context) error {
	symbol := c.Data()
	return t.handleBtnBackDetailStockPositionWithParam(ctx, c, &symbol, nil)
}

func (t *TelegramBotService) handleBtnTimeframeStockPositionMonitoring(ctx context.Context, c telebot.Context) error {
	symbol := c.Data() // The symbol is passed as data
	menu := &telebot.ReplyMarkup{}

	msg := fmt.Sprintf("📊 Analisa Posisi Saham: *$%s*\n\nSilahkan pilih strategi analisa yang paling relevan dengan kondisi posisi kamu saat ini 👇", symbol)
	btnMain := menu.Data("🔹 Main Signal", btnInputTimeFrameStockPositionAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameMain, symbol))
	btnEntry := menu.Data("🔹 Entry Presisi", btnInputTimeFrameStockPositionAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameEntry, symbol))
	btnExit := menu.Data("🔹 Exit Presisi", btnInputTimeFrameStockPositionAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameExit, symbol))
	btnNotes := menu.Data(btnNotesTimeFrameStockPosition.Text, btnNotesTimeFrameStockPosition.Unique, symbol)
	btnBack := menu.Data(btnBackDetailStockPosition.Text, btnBackDetailStockPosition.Unique, symbol)
	menu.Inline(
		menu.Row(btnMain, btnEntry),
		menu.Row(btnExit, btnNotes),
		menu.Row(btnBack),
	)

	return c.Edit(msg, menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnNotesTimeFrameStockPosition(ctx context.Context, c telebot.Context) error {
	symbol := c.Data() // The symbol is passed as data
	menu := &telebot.ReplyMarkup{}
	btnMain := menu.Data("🔹 Main Signal", btnInputTimeFrameStockPositionAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameMain, symbol))
	btnEntry := menu.Data("🔹 Entry Presisi", btnInputTimeFrameStockPositionAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameEntry, symbol))
	btnExit := menu.Data("🔹 Exit Presisi", btnInputTimeFrameStockPositionAnalysis.Unique, fmt.Sprintf(dataInputTimeFrameExit, symbol))
	btnBack := menu.Data("🔙 Kembali", btnStockPositionMonitoring.Unique, symbol)
	menu.Inline(
		menu.Row(btnMain, btnEntry, btnExit),
		menu.Row(btnBack),
	)
	return c.Edit(t.FormatNotesTimeFrameStockMessage(), menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnStockPositionMonitoringAnalysis(ctx context.Context, c telebot.Context) error {
	data := c.Data() // The symbol is passed as data

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

		positions, err := t.stockService.GetLatestStockPositionMonitoring(newCtx, models.GetStockPositionMonitoringParam{
			StockCode:  symbol,
			Interval:   interval,
			Range:      rng,
			TelegramID: c.Sender().ID,
			Limit:      1,
		})
		if err != nil {
			close(stopChan)
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to get stock position monitoring")
			_, err := t.telegramRateLimiter.Edit(newCtx, c, msg, commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		if len(positions) == 0 {
			defer close(stopChan)
			t.logger.WithField("symbol", symbol).Warn("No stock position monitoring found")
			t.stockService.RequestStockPositionMonitoring(newCtx, &models.RequestStockPositionMonitoring{
				TelegramID:     c.Sender().ID,
				StockCode:      symbol,
				Interval:       interval,
				Range:          rng,
				SendToTelegram: true,
			})
			return
		}

		defer close(stopChan)
		position := positions[0]

		var stockMonitoring models.PositionMonitoringResponse
		if err := json.Unmarshal([]byte(position.Data), &stockMonitoring); err != nil {
			t.logger.WithError(err).WithField("symbol", symbol).Error("Failed to unmarshal stock monitoring")
			// Send error message
			_, err := t.telegramRateLimiter.Send(newCtx, c, fmt.Sprintf("❌ Gagal parse data %s", symbol))
			if err != nil {
				t.logger.WithError(err).Error("Failed to send error message")
			}
			return
		}

		// Format position monitoring message
		message := t.FormatPositionMonitoringMessage(&stockMonitoring)

		// Send the position monitoring results
		_, err = t.telegramRateLimiter.Edit(newCtx, c, msg, message, &telebot.SendOptions{
			ParseMode: telebot.ModeMarkdown,
		})
		if err != nil {
			t.logger.WithError(err).Error("Failed to send error message")
		}

	})

	return nil
}
