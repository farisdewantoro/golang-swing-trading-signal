package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"strconv"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleSetPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	t.userStates[userID] = StateWaitingSetPositionSymbol
	reqData := &models.RequestSetPositionData{
		UserTelegram: models.ToRequestUserTelegram(c.Sender()),
	}
	t.userPositionData[userID] = reqData
	t.logger.Infof("Starting /setposition for user %d", userID)
	return c.Send("📈 Masukkan kode saham kamu (contoh: ANTM):", &telebot.SendOptions{})
}

func (t *TelegramBotService) handleSetPositionConversation(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	text := c.Text()
	state := t.userStates[userID] // We already know the state exists

	data, data_ok := t.userPositionData[userID]
	if !data_ok {
		// Should not happen, but as a safeguard
		delete(t.userStates, userID)
		return c.Send("Terjadi kesalahan internal (data posisi tidak ditemukan), silakan coba lagi dengan /setposition.")
	}

	switch state {
	case StateWaitingSetPositionSymbol:
		data.Symbol = strings.ToUpper(text)
		c.Send(fmt.Sprintf("👍 Oke, kode *%s* tercatat!", data.Symbol), &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		t.userStates[userID] = StateWaitingSetPositionBuyPrice
		return c.Send("💰 Berapa harga belinya ? (contoh: 150.5)", &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

	case StateWaitingSetPositionBuyPrice:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga beli tidak valid. Silakan masukkan angka (contoh: 150.5).")
		}
		data.BuyPrice = price
		t.userStates[userID] = StateWaitingSetPositionBuyDate
		return c.Send("📅 Kapan tanggal belinya? (format: YYYY-MM-DD)", &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

	case StateWaitingSetPositionBuyDate:
		_, err := time.Parse("2006-01-02", text)
		if err != nil {
			return c.Send("Format tanggal tidak valid. Silakan gunakan format YYYY-MM-DD.")
		}
		data.BuyDate = text
		t.userStates[userID] = StateWaitingSetPositionTakeProfit
		return c.Send("🎯 Target take profit-nya di harga berapa? (contoh: 180.0)", &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

	case StateWaitingSetPositionTakeProfit:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga take profit tidak valid. Silakan masukkan angka.")
		}
		data.TakeProfit = price
		t.userStates[userID] = StateWaitingSetPositionStopLoss
		return c.Send("📉 Stop loss-nya di harga berapa? (contoh: 140.0)", &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

	case StateWaitingSetPositionStopLoss:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga stop loss tidak valid. Silakan masukkan angka.")
		}
		data.StopLoss = price
		t.userStates[userID] = StateWaitingSetPositionMaxHolding
		return c.Send("⏳ Berapa maksimal hari mau di-hold? (contoh: 1) \n\n📌 *Note:* Isi angka dari *1* sampai *5* hari.", &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

	case StateWaitingSetPositionMaxHolding:
		intVal, err := strconv.Atoi(text)
		if err != nil || intVal <= 0 {
			return c.Send("Format maksimal hari hold tidak valid. Silakan masukkan angka bulat positif.")
		}
		data.MaxHolding = intVal
		t.userStates[userID] = StateWaitingSetPositionAlertPrice

		menu := &telebot.ReplyMarkup{}
		menu.Inline(
			menu.Row(btnSetPositionAlertPriceYes, btnSetPositionAlertPriceNo),
		)

		return c.Send("🚨 Aktifkan alert untuk data ini?\n\nNote: Sistem akan kirim pesan kalau harga mencapai take profit atau stop loss yang kamu tentukan.", menu)

	case StateWaitingSetPositionAlertPrice, StateWaitingSetPositionAlertMonitor:
		return c.Send("👆 Silakan pilih salah satu opsi di atas, atau kirim /cancel untuk membatalkan.")
	}
	return nil
}

func (t *TelegramBotService) handleBtnSetPositionAlertPriceYes(ctx context.Context, c telebot.Context) error {
	if t.userStates[c.Sender().ID] != StateWaitingSetPositionAlertPrice {
		t.ResetUserState(c.Sender().ID)
		return c.Send("❌ Terjadi kesalahan internal, silakan mulai lagi dengan /setposition.")
	}
	userID := c.Sender().ID
	data := t.userPositionData[userID]
	data.AlertPrice = true
	t.bot.Edit(c.Message(), "✅ Alert harga saham diaktifkan.", &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	t.userStates[userID] = StateWaitingSetPositionAlertMonitor

	menu := &telebot.ReplyMarkup{}
	menu.Inline(
		menu.Row(btnSetPositionAlertMonitorYes, btnSetPositionAlertMonitorNo),
	)
	return c.Send("🔎 Aktifkan monitoring alert?\n\nNote: Sistem akan menganalisis posisi ini dan kirim laporan singkat: apakah masih aman, rawan, atau mendekati batas hold/SL.", menu)
}

func (t *TelegramBotService) handleBtnSetPositionAlertPriceNo(ctx context.Context, c telebot.Context) error {
	if t.userStates[c.Sender().ID] != StateWaitingSetPositionAlertPrice {
		t.ResetUserState(c.Sender().ID)
		return c.Send("❌ Terjadi kesalahan internal, silakan mulai lagi dengan /setposition.")
	}
	userID := c.Sender().ID
	data := t.userPositionData[userID]
	data.AlertPrice = false
	t.bot.Edit(c.Message(), "❌ Alert harga saham tidak diaktifkan.", &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	t.userStates[userID] = StateWaitingSetPositionAlertMonitor

	menu := &telebot.ReplyMarkup{}
	menu.Inline(
		menu.Row(btnSetPositionAlertMonitorYes, btnSetPositionAlertMonitorNo),
	)
	return c.Send("🔎 Aktifkan monitoring alert?\n\nNote: Sistem akan menganalisis posisi ini dan kirim laporan singkat: apakah masih aman, rawan, atau mendekati batas hold/SL.", menu)
}

func (t *TelegramBotService) handleSetPositionAlertMonitorYes(ctx context.Context, c telebot.Context) error {
	if t.userStates[c.Sender().ID] != StateWaitingSetPositionAlertMonitor {
		t.ResetUserState(c.Sender().ID)
		return c.Send("❌ Terjadi kesalahan internal, silakan mulai lagi dengan /setposition.")
	}
	userID := c.Sender().ID
	data := t.userPositionData[userID]
	data.AlertMonitor = true
	t.bot.Edit(c.Message(), "✅ Alert monitor diaktifkan.", &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	return t.handleSetPositionFinish(ctx, c)
}

func (t *TelegramBotService) handleSetPositionAlertMonitorNo(ctx context.Context, c telebot.Context) error {
	if t.userStates[c.Sender().ID] != StateWaitingSetPositionAlertMonitor {
		t.ResetUserState(c.Sender().ID)
		return c.Send("❌ Terjadi kesalahan internal, silakan mulai lagi dengan /setposition.")
	}
	userID := c.Sender().ID
	data := t.userPositionData[userID]
	data.AlertMonitor = false
	t.bot.Edit(c.Message(), "❌ Alert monitor tidak diaktifkan.", &telebot.SendOptions{
		ParseMode: telebot.ModeMarkdown,
	})
	return t.handleSetPositionFinish(ctx, c)
}

func (t *TelegramBotService) handleSetPositionFinish(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	data := t.userPositionData[userID]

	defer t.ResetUserState(userID)

	if err := t.stockService.SetStockPosition(ctx, data); err != nil {
		return c.Send("❌ Terjadi kesalahan internal, silakan mulai lagi dengan /setposition.")
	}

	return c.Send(t.FormatResultSetPositionMessage(data), &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})

}
