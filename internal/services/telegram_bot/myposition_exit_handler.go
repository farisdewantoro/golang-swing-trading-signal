package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strconv"
	"time"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleExitPositionConversation(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	text := c.Text()
	state := t.userStates[userID] // We already know the state exists

	data, data_ok := t.userExitPositionData[userID]
	if !data_ok {
		// Should not happen, but as a safeguard
		delete(t.userStates, userID)
		return c.Send("Terjadi kesalahan internal (data posisi tidak ditemukan).")
	}

	switch state {
	case StateWaitingExitPositionInputExitPrice:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return c.Send("Format harga jual tidak valid. Silakan masukkan angka (contoh: 150.5).")
		}
		data.ExitPrice = price
		err = c.Send(fmt.Sprintf(`
🚀 Exit posisi saham *%s (2/2)*

📅 Kapan tanggal jualnya? (contoh: 2025-05-18)`, data.Symbol), &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
		if err != nil {
			return c.Send(commonMessageInternalError)
		}
		t.userStates[userID] = StateWaitingExitPositionInputExitDate
		return nil
	case StateWaitingExitPositionInputExitDate:
		date, err := time.Parse("2006-01-02", text)
		if err != nil {
			return c.Send("Format tanggal tidak valid. Silakan gunakan format YYYY-MM-DD.")
		}
		data.ExitDate = date
		t.userStates[userID] = StateWaitingExitPositionConfirm
		msg := fmt.Sprintf(`
📌 Mohon cek kembali data yang kamu masukkan:

• Kode Saham   : %s 
• Harga Exit   : %.2f  
• Tanggal Exit : %s  
		`, data.Symbol, data.ExitPrice, data.ExitDate.Format("2006-01-02"))
		menu := &telebot.ReplyMarkup{}
		btnSave := menu.Data(btnSaveExitPosition.Text, btnSaveExitPosition.Unique)
		btnCancel := menu.Data(btnCancelGeneral.Text, btnCancelGeneral.Unique)
		menu.Inline(
			menu.Row(btnSave, btnCancel),
		)
		return c.Send(msg, menu, telebot.ModeMarkdown)
	case StateWaitingExitPositionConfirm:
		return c.Send("👆 Silakan pilih salah satu opsi di atas, atau kirim /cancel untuk membatalkan.")
	}
	return nil
}

func (t *TelegramBotService) handleBtnSaveExitPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	data := t.userExitPositionData[userID]
	if data == nil {
		return c.Send(commonMessageInternalError)
	}
	if data.ExitPrice == 0 || data.ExitDate.IsZero() {
		return c.Send("❌ Data tidak lengkap, silakan masukkan harga exit dan tanggal exit.")
	}

	stopChan := make(chan struct{})
	msg := t.showLoadingGeneral(c, stopChan)

	go func() {
		newCtx, cancel := context.WithTimeout(t.ctx, t.config.TimeoutDuration)
		defer cancel()

		if err := t.stockService.UpdateStockPositionTelegramUser(newCtx, userID, data.StockPositionID, &models.StockPositionUpdateRequest{
			ExitPrice: utils.ToPointer(data.ExitPrice),
			ExitDate:  utils.ToPointer(data.ExitDate),
			IsActive:  utils.ToPointer(false),
		}); err != nil {
			close(stopChan)
			if err := c.Send(commonMessageInternalError); err != nil {
				t.logger.WithError(err).Error("Failed to update stock position")
			}
		}
		time.Sleep(1 * time.Second)
		close(stopChan)
		t.bot.Edit(msg, "✅ Exit posisi berhasil disimpan.")
		time.Sleep(1 * time.Second)
		t.handleMyPositionWithEditMessage(newCtx, c, true)

	}()
	t.ResetUserState(userID)

	return nil
}
