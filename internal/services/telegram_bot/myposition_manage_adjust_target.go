package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"strconv"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleBtnAdjustTargetPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	data := c.Data()

	t.ResetUserState(userID)

	stockPositionIDInt, err := strconv.Atoi(data)
	if err != nil {
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}

	stockPosition, err := t.stockService.GetStockPosition(ctx, models.StockPositionQueryParam{
		IDs: []uint{uint(stockPositionIDInt)},
	})
	if err != nil {
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}
	if len(stockPosition) == 0 {
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}
	t.userStates[userID] = StateWaitingAdjustTargetPositionInputTargetPrice
	reqData := &models.RequestAdjustTargetPositionData{
		StockPositionID: stockPosition[0].ID,
		StockCode:       stockPosition[0].StockCode,
	}
	t.userAdjustTargetPositionData[userID] = reqData

	msg := fmt.Sprintf(`🎯 (1/4) Masukan Target Price Baru:
(Target Price Saat ini : %d)	

<i>Ketika "0" jika tidak ingin mengubah</i>
`, int(stockPosition[0].TakeProfitPrice))

	_, err = t.telegramRateLimiter.Edit(ctx, c, c.Message(), msg, telebot.ModeHTML)
	return err
}

func (t *TelegramBotService) handleAdjustTargetPositionConversation(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	text := c.Text()
	state := t.userStates[userID] // We already know the state exists

	data, data_ok := t.userAdjustTargetPositionData[userID]
	if !data_ok {
		// Should not happen, but as a safeguard
		t.ResetUserState(userID)
		_, err := t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}

	stockPosition, err := t.stockService.GetStockPosition(ctx, models.StockPositionQueryParam{
		IDs: []uint{data.StockPositionID},
	})
	if err != nil {
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}
	if len(stockPosition) == 0 {
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}

	switch state {
	case StateWaitingAdjustTargetPositionInputTargetPrice:
		targetPrice, err := strconv.ParseFloat(text, 64)
		if err != nil {
			_, err = t.telegramRateLimiter.Send(ctx, c, "❌ Harap masukan angka yang valid contoh: 2002")
			return err
		}
		if targetPrice == 0 {
			targetPrice = stockPosition[0].TakeProfitPrice
		}
		data.TargetPrice = targetPrice
		t.userStates[userID] = StateWaitingAdjustTargetPositionInputStopLossPrice

		msg := fmt.Sprintf(`💰 (2/4) Masukan Stop Loss Baru:
(Stop Loss Saat ini : %d)	

<i>Ketika "0" jika tidak ingin mengubah</i>
`, int(stockPosition[0].StopLossPrice))
		_, err = t.telegramRateLimiter.Send(ctx, c, msg, telebot.ModeHTML)
		return err
	case StateWaitingAdjustTargetPositionInputStopLossPrice:
		stopLossPrice, err := strconv.ParseFloat(text, 64)
		if err != nil {
			_, err = t.telegramRateLimiter.Send(ctx, c, "❌ Harap masukan angka yang valid contoh: 100")
			return err
		}
		if stopLossPrice == 0 {
			stopLossPrice = stockPosition[0].StopLossPrice
		}
		data.StopLossPrice = stopLossPrice
		t.userStates[userID] = StateWaitingAdjustTargetPositionMaxHoldingDays

		msg := fmt.Sprintf(`⏳ (3/4) Masukan Max Holding Days Baru:
(Max Holding Days Saat ini : %d)	

<i>Ketika "0" jika tidak ingin mengubah</i>
`, int(stockPosition[0].MaxHoldingPeriodDays))
		_, err = t.telegramRateLimiter.Send(ctx, c, msg, telebot.ModeHTML)
		return err
	case StateWaitingAdjustTargetPositionMaxHoldingDays:
		maxHoldingDays, err := strconv.Atoi(text)
		if err != nil {
			_, err = t.telegramRateLimiter.Send(ctx, c, "❌ Harap masukan angka yang valid contoh: 1-5 hari")
			return err
		}
		if maxHoldingDays == 0 {
			maxHoldingDays = stockPosition[0].MaxHoldingPeriodDays
		}
		data.MaxHoldingDays = maxHoldingDays
		t.userStates[userID] = StateWaitingAdjustTargetPositionConfirm

		msg := fmt.Sprintf(`📝 (4/4) Konfirmasi Perubahan Target Posisi - %s

Berikut adalah rincian perubahan yang akan diterapkan:

🎯 Target Price       : %d  
🔻 Stop Loss          : %d  
⏳ Max Holding Days   : %d hari  

Mohon periksa kembali angka-angka di atas sebelum disimpan.

<i>Jika sudah sesuai, tekan tombol ✅ Konfirmasi.  
Jika ingin membatalkan atau mengubah, tekan ❌ Batal.</i>

`, stockPosition[0].StockCode, int(data.TargetPrice), int(data.StopLossPrice), data.MaxHoldingDays)

		menu := &telebot.ReplyMarkup{}
		menu.Inline(
			menu.Row(btnAdjustTargetPositionConfirm), menu.Row(btnCancelGeneral),
		)

		_, err = t.telegramRateLimiter.Send(ctx, c, msg, menu, telebot.ModeHTML)
		return err
	case StateWaitingAdjustTargetPositionConfirm:
		return c.Send("👆 Silakan pilih salah satu opsi di atas, atau kirim /cancel untuk membatalkan.")
	default:
		return c.Send("Terjadi kesalahan internal (state tidak ditemukan), silakan coba lagi dengan /setposition.")
	}
}

func (t *TelegramBotService) handleBtnAdjustTargetPositionConfirm(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	data := t.userAdjustTargetPositionData[userID]

	defer t.ResetUserState(userID)

	if err := t.stockService.UpdateStockPositionTelegramUser(ctx, userID, uint(data.StockPositionID), &models.StockPositionUpdateRequest{
		TargetPrice:          &data.TargetPrice,
		StopLossPrice:        &data.StopLossPrice,
		MaxHoldingPeriodDays: &data.MaxHoldingDays,
	}); err != nil {
		_, err = t.telegramRateLimiter.Send(ctx, c, commonMessageInternalError)
		return err
	}

	msg := fmt.Sprintf(`✅<b> Perubahan berhasil disimpan!</b>

Target posisi %s telah diperbarui dengan detail berikut:

🎯 Target Price     : %d  
🔻 Stop Loss        : %d  
⏳ Max Holding Days : %d hari

<i>📊 Sistem akan mulai memantau posisi Anda berdasarkan parameter baru ini.</i>

Terima kasih telah memperbarui strategi Anda.  
Tetap disiplin dan semoga cuan! 🚀

`, data.StockCode, int(data.TargetPrice), int(data.StopLossPrice), data.MaxHoldingDays)
	_, err := t.telegramRateLimiter.Edit(ctx, c, c.Message(), msg, telebot.ModeHTML)
	return err
}
