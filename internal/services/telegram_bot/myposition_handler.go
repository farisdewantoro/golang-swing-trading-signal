package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"strconv"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleMyPosition(ctx context.Context, c telebot.Context) error {
	return t.handleMyPositionWithEditMessage(ctx, c, false)
}

func (t *TelegramBotService) handleMyPositionWithEditMessage(ctx context.Context, c telebot.Context, isEditMessage bool) error {
	userID := c.Sender().ID

	positions, err := t.stockService.GetStockPositionsTelegramUser(ctx, userID)
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	if len(positions) == 0 {
		return c.Send("❌ Tidak ada saham aktif yang kamu set position saat ini.")
	}

	header := `📊 Posisi Saham yang Kamu Pantau

Berikut adalah daftar saham yang sedang kamu monitor.

Tekan salah satu saham di bawah ini untuk melihat detail lengkap posisinya — termasuk harga beli, target jual, stop loss, umur posisi, status alert, dan monitoring.

Kamu juga bisa mengatur ulang atau menghapus posisi setelah membukanya.`

	menu := &telebot.ReplyMarkup{}
	rows := []telebot.Row{}

	for _, p := range positions {
		btn := menu.Data(fmt.Sprintf("➤ %s ($%.2f)", p.StockCode, p.BuyPrice), btnToDetailStockPosition.Unique, fmt.Sprintf("%d", p.ID))
		rows = append(rows, menu.Row(btn))
	}
	btnDeleteMessage := menu.Data(btnDeleteMessage.Text, btnDeleteMessage.Unique)
	rows = append(rows, menu.Row(btnDeleteMessage))

	menu.Inline(rows...)

	if isEditMessage {
		return c.Edit(header, menu, telebot.ModeMarkdown)
	}

	return c.Send(header, menu, telebot.ModeMarkdown)

}

func (t *TelegramBotService) handleBtnToDetailStockPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	id, err := strconv.Atoi(c.Data())
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	positions, err := t.stockService.GetStockPosition(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{userID},
		IsActive:    true,
		IDs:         []uint{uint(id)},
	})
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	if len(positions) == 0 {
		return c.Send("❌ Tidak ada posisi yang ditemukan.")
	}

	// Tombol analisa
	menu := &telebot.ReplyMarkup{}
	btn := menu.Data("🔍 Analisa", btnStockPositionMonitoring.Unique, positions[0].StockCode)
	btnManage := menu.Data(btnManageStockPosition.Text, btnManageStockPosition.Unique, strconv.FormatUint(uint64(positions[0].ID), 10))
	btnBack := menu.Data(btnBackStockPosition.Text, btnBackStockPosition.Unique)
	btnNews := menu.Data(btnNewsStockPosition.Text, btnNewsStockPosition.Unique, positions[0].StockCode)

	menu.Inline(
		menu.Row(btn, btnManage),
		menu.Row(btnNews, btnBack),
	)

	return c.Edit(t.FormatMyStockPositionMessage(positions[0]), menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnBackStockPosition(ctx context.Context, c telebot.Context) error {
	return t.handleMyPositionWithEditMessage(ctx, c, true)
}
