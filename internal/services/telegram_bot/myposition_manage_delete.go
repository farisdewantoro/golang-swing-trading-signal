package telegram_bot

import (
	"context"
	"fmt"
	"strconv"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleBtnDeleteStockPosition(ctx context.Context, c telebot.Context) error {
	c.Respond(&telebot.CallbackResponse{
		Text:      "🔄 Menghapus....",
		ShowAlert: false,
	})

	stockPositionID := c.Data()

	stockPositionIDInt, err := strconv.Atoi(stockPositionID)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal mengambil posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	if err = t.stockService.DeleteStockPositionTelegramUser(ctx, c.Sender().ID, uint(stockPositionIDInt)); err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal menghapus posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	return c.Edit("✅ Posisi saham berhasil dihapus.")
}
