package telegram_bot

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strconv"
	"strings"
	"time"

	"gopkg.in/telebot.v3"
)

func (t *TelegramBotService) handleBtnManageStockPosition(ctx context.Context, c telebot.Context) error {
	stockPositionID := c.Data()

	stockPositionIDInt, err := strconv.Atoi(stockPositionID)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal mengambil posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	stockPosition, err := t.stockService.GetStockPosition(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{c.Sender().ID},
		IDs:         []uint{uint(stockPositionIDInt)},
	})
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal mengambil posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	if len(stockPosition) == 0 {
		return c.Edit(fmt.Sprintf("❌ Tidak ditemukan posisi untuk %s", stockPositionID))
	}

	stockCode := stockPosition[0].StockCode
	msgText := fmt.Sprintf(
		"⚙️ Kelola Posisi Saham *%s*\n\nPilih aksi yang ingin kamu lakukan terhadap saham ini 👇\n\n",
		stockCode,
	)

	// Buat keyboard
	keyboard := &telebot.ReplyMarkup{}

	// Tombol aksi utama
	btnExit := keyboard.Data("🚪 Exit dari Posisi", btnExitStockPosition.Unique, fmt.Sprintf("%s|%d", stockCode, stockPositionIDInt))
	btnDelete := keyboard.Data(btnDeleteStockPosition.Text, btnDeleteStockPosition.Unique, stockPositionID)

	// Toggle Alert
	var btnAlert telebot.Btn
	isAlertOn := stockPosition[0].PriceAlert != nil && *stockPosition[0].PriceAlert
	if isAlertOn {
		btnAlert = keyboard.Data("🔕 Nonaktifkan Alert", btnUpdateAlertPrice.Unique, fmt.Sprintf("%s|false", stockPositionID))
	} else {
		btnAlert = keyboard.Data("🔔 Aktifkan Alert", btnUpdateAlertPrice.Unique, fmt.Sprintf("%s|true", stockPositionID))
	}

	// Toggle Monitoring
	var btnMonitor telebot.Btn
	isMonitoringOn := stockPosition[0].MonitorPosition != nil && *stockPosition[0].MonitorPosition
	if isMonitoringOn {
		btnMonitor = keyboard.Data("❌ Nonaktifkan Monitoring", btnUpdateAlertMonitor.Unique, fmt.Sprintf("%s|false", stockPositionID))
	} else {
		btnMonitor = keyboard.Data("📡 Aktifkan Monitoring", btnUpdateAlertMonitor.Unique, fmt.Sprintf("%s|true", stockPositionID))
	}

	// Tombol kembali
	btnBack := keyboard.Data(btnBackActionStockPosition.Text, btnBackActionStockPosition.Unique, stockPositionID)
	btnAdjustTarget := keyboard.Data(btnAdjustTargetPosition.Text, btnAdjustTargetPosition.Unique, stockPositionID)

	// Susun tombol: satu per baris
	keyboard.Inline(
		keyboard.Row(btnExit),
		keyboard.Row(btnAdjustTarget),
		keyboard.Row(btnDelete),
		keyboard.Row(btnAlert),
		keyboard.Row(btnMonitor),
		keyboard.Row(btnBack),
	)

	return c.Edit(msgText, keyboard, telebot.ModeMarkdown)
}

func (t *TelegramBotService) handleBtnExitStockPosition(ctx context.Context, c telebot.Context) error {
	userID := c.Sender().ID
	data := c.Data()

	userState := t.userStates[userID]
	if userState != StateIdle {
		t.ResetUserState(userID)
		return c.Send(commonMessageInternalError)
	}

	parts := strings.Split(data, "|")
	if len(parts) != 2 {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}

	stockPositionIDInt, err := strconv.Atoi(parts[1])
	if err != nil {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}

	msg := fmt.Sprintf(`🚀 Exit posisi saham *%s (1/2)*

Masukkan *harga jual* kamu di bawah ini (dalam angka).  
Contoh: 175.00

`, parts[0])

	err = c.Edit(msg, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
	if err != nil {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	t.userStates[userID] = StateWaitingExitPositionInputExitPrice
	t.userExitPositionData[userID] = &models.RequestExitPositionData{
		Symbol:          parts[0],
		StockPositionID: uint(stockPositionIDInt),
	}

	return nil
}

func (t *TelegramBotService) handleBtnUpdateAlertPrice(ctx context.Context, c telebot.Context) error {
	data := c.Data()
	parts := strings.Split(data, "|")
	if len(parts) != 2 {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	stockPositionID := parts[0]
	isAlertOn := parts[1]

	stockPositionIDInt, err := strconv.Atoi(stockPositionID)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal mengambil posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	isAlertOnBool, err := strconv.ParseBool(isAlertOn)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal parse data untuk %s: %s", stockPositionID, err.Error()))
	}

	if err = t.stockService.UpdateStockPositionTelegramUser(ctx, c.Sender().ID, uint(stockPositionIDInt), &models.StockPositionUpdateRequest{
		PriceAlert: &isAlertOnBool,
	}); err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal update status alert untuk %s: %s", stockPositionID, err.Error()))
	}

	isActive := "✅ Alert Price berhasil diaktifkan."
	if !isAlertOnBool {
		isActive = "❌ Alert Price berhasil dinonaktifkan."
	}

	c.Edit(isActive)
	time.Sleep(200 * time.Millisecond)
	return t.handleBtnBackDetailStockPositionWithParam(ctx, c, nil, utils.ToPointer(uint(stockPositionIDInt)))
}

func (t *TelegramBotService) handleBtnUpdateAlertMonitor(ctx context.Context, c telebot.Context) error {
	data := c.Data()
	parts := strings.Split(data, "|")
	if len(parts) != 2 {
		return c.Edit(commonMessageInternalError, &telebot.ReplyMarkup{}, telebot.ModeMarkdown)
	}
	stockPositionID := parts[0]
	isMonitorOn := parts[1]

	stockPositionIDInt, err := strconv.Atoi(stockPositionID)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal mengambil posisi untuk %s: %s", stockPositionID, err.Error()))
	}

	isMonitorOnBool, err := strconv.ParseBool(isMonitorOn)
	if err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal parse data untuk %s: %s", stockPositionID, err.Error()))
	}

	if err = t.stockService.UpdateStockPositionTelegramUser(ctx, c.Sender().ID, uint(stockPositionIDInt), &models.StockPositionUpdateRequest{
		MonitorPosition: &isMonitorOnBool,
	}); err != nil {
		return c.Edit(fmt.Sprintf("❌ Gagal update status monitoring untuk %s: %s", stockPositionID, err.Error()))
	}

	isActive := "✅ Stock Monitoring berhasil diaktifkan."
	if !isMonitorOnBool {
		isActive = "❌ Stock Monitoring berhasil dinonaktifkan."
	}

	c.Edit(isActive)
	time.Sleep(200 * time.Millisecond)
	return t.handleBtnBackDetailStockPositionWithParam(ctx, c, nil, utils.ToPointer(uint(stockPositionIDInt)))

}

func (t *TelegramBotService) handleBtnBackActionStockPosition(ctx context.Context, c telebot.Context) error {
	return t.handleBtnToDetailStockPosition(ctx, c)
}

func (t *TelegramBotService) handleBtnBackDetailStockPositionWithParam(ctx context.Context, c telebot.Context, symbol *string, stockPosisitionID *uint) error {
	senderID := c.Sender().ID

	param := models.StockPositionQueryParam{
		TelegramIDs: []int64{senderID},
		IsActive:    true,
	}
	if symbol != nil {
		param.StockCodes = []string{*symbol}
	}
	if stockPosisitionID != nil {
		param.IDs = []uint{*stockPosisitionID}
	}
	positions, err := t.stockService.GetStockPosition(ctx, param)
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

	return c.Edit(t.FormatMyStockPositionMessage(&positions[0]), menu, telebot.ModeMarkdown)
}
