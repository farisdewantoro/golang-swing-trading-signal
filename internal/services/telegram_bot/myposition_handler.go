package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strconv"
	"strings"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleMyPosition(ctx context.Context, c telebot.Context) error {
	return t.handleMyPositionWithEditMessage(ctx, c, false)
}

func (t *TelegramBotService) handleMyPositionWithEditMessage(ctx context.Context, c telebot.Context, isEditMessage bool) error {
	userID := c.Sender().ID

	parts := strings.Split(dataInputTimeFrameExit, "|")

	if len(parts) != 3 {
		return t.telegramRateLimiter.EditWithoutMsg(ctx, c, commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	interval, rng := parts[1], parts[2]

	monitoringParam := &models.StockPositionMonitoringQueryParam{
		Interval: utils.ToPointer(interval),
		Range:    utils.ToPointer(rng),
		Limit:    utils.ToPointer(1),
	}

	positions, err := t.stockService.GetStockPositionsTelegramUser(ctx, userID, monitoringParam)
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	if len(positions) == 0 {
		return c.Send("❌ Tidak ada saham aktif yang kamu set position saat ini.")
	}

	sb := strings.Builder{}
	header := `📊 Posisi Saham yang Kamu Pantau Saat ini:`
	sb.WriteString(header)
	sb.WriteString("\n\n")
	body := t.FormatMyPositionListMessage(positions)
	sb.WriteString(body)
	footer := `👉 Tekan tombol di bawah untuk melihat detail lengkap atau mengelola posisi.`
	sb.WriteString(footer)
	menu := &telebot.ReplyMarkup{}
	rows := []telebot.Row{}

	for i := 0; i < len(positions); i += 2 {
		if i+1 < len(positions) {
			btn1 := menu.Data(positions[i].StockCode, btnToDetailStockPosition.Unique, fmt.Sprintf("%d", positions[i].ID))
			btn2 := menu.Data(positions[i+1].StockCode, btnToDetailStockPosition.Unique, fmt.Sprintf("%d", positions[i+1].ID))
			rows = append(rows, menu.Row(btn1, btn2))
		} else {
			btn := menu.Data(positions[i].StockCode, btnToDetailStockPosition.Unique, fmt.Sprintf("%d", positions[i].ID))
			rows = append(rows, menu.Row(btn))
		}
	}

	btnDelete := menu.Data("🗑 Hapus Pesan", btnDeleteMessage.Unique)
	rows = append(rows, menu.Row(btnDelete))

	menu.Inline(rows...)

	if isEditMessage {
		return c.Edit(sb.String(), menu, telebot.ModeMarkdown)
	}

	return c.Send(sb.String(), menu, telebot.ModeMarkdown)

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
