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

	monitoringParam := &models.StockPositionMonitoringQueryParam{
		ShowNewest: utils.ToPointer(true),
	}

	positions, err := t.stockService.GetStockPositionsTelegramUser(ctx, userID, monitoringParam)
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	if len(positions) == 0 {
		return c.Send("❌ Tidak ada saham aktif yang kamu set position saat ini.")
	}

	stockCodes := []string{}

	for _, position := range positions {
		stockCodes = append(stockCodes, position.StockCode)
	}

	lastMarketPriceMap, err := t.getLastMarketPrice(ctx, stockCodes)
	if err != nil {
		t.logger.WithError(err).Error("Failed to get last market prices")
		return c.Send(commonMessageInternalError)
	}

	sb := strings.Builder{}
	header := `📊 Posisi Saham yang Kamu Pantau Saat ini:`
	sb.WriteString(header)
	sb.WriteString("\n")
	body := t.FormatMyPositionListMessage(positions, lastMarketPriceMap)
	sb.WriteString(body)
	footer := "\n👉 Tekan tombol di bawah untuk melihat detail lengkap atau mengelola posisi."
	sb.WriteString(footer)
	menu := &telebot.ReplyMarkup{}
	rows := []telebot.Row{}
	var tempRow []telebot.Btn

	for _, position := range positions {
		btn := menu.Data(position.StockCode, btnToDetailStockPosition.Unique, fmt.Sprintf("%d", position.ID))
		tempRow = append(tempRow, btn)
		if len(tempRow) == 2 {
			rows = append(rows, menu.Row(tempRow...))
			tempRow = []telebot.Btn{}
		}
	}

	btnDelete := menu.Data("🗑 Hapus Pesan", btnDeleteMessage.Unique)

	if len(tempRow) > 0 {
		tempRow = append(tempRow, btnDelete)
		rows = append(rows, menu.Row(tempRow...))
	} else {
		rows = append(rows, menu.Row(btnDelete))
	}

	menu.Inline(rows...)

	if isEditMessage {
		return c.Edit(sb.String(), menu, telebot.ModeHTML)
	}

	return c.Send(sb.String(), menu, telebot.ModeHTML)

}

func (t *TelegramBotService) handleBtnToDetailStockPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	id, err := strconv.Atoi(c.Data())
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	position, err := t.stockService.GetStockPositionWithHistoryMonitoring(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{userID},
		IsActive:    true,
		IDs:         []uint{uint(id)},
		Monitoring: &models.StockPositionMonitoringQueryParam{
			Limit:      utils.ToPointer(t.config.MaxShowHistoryAnalysis),
			ShowNewest: utils.ToPointer(true),
		},
	})
	if err != nil {
		return c.Send(commonMessageInternalError)
	}

	if position == nil {
		return c.Send("❌ Tidak ada posisi yang ditemukan.")
	}

	// Tombol analisa
	menu := &telebot.ReplyMarkup{}
	btn := menu.Data("🔍 Analisa", btnStockPositionMonitoring.Unique, position.StockCode)
	btnManage := menu.Data(btnManageStockPosition.Text, btnManageStockPosition.Unique, strconv.FormatUint(uint64(position.ID), 10))
	btnBack := menu.Data(btnBackStockPosition.Text, btnBackStockPosition.Unique)
	btnNews := menu.Data(btnNewsStockPosition.Text, btnNewsStockPosition.Unique, position.StockCode)

	menu.Inline(
		menu.Row(btn, btnManage),
		menu.Row(btnNews, btnBack),
	)
	lastPrices, _ := t.getLastMarketPrice(ctx, []string{position.StockCode})
	var marketPrice *models.RedisLastPrice
	if len(lastPrices) > 0 {
		if val, ok := lastPrices[position.StockCode]; ok {
			marketPrice = &val
		}
	}

	return c.Edit(t.FormatMyStockPositionMessage(position, marketPrice), menu, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnBackStockPosition(ctx context.Context, c telebot.Context) error {
	return t.handleMyPositionWithEditMessage(ctx, c, true)
}
