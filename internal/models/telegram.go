package models

import (
	"golang-swing-trading-signal/internal/utils"
	"time"

	"gopkg.in/telebot.v3"
)

type RequestUserTelegram struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	LanguageCode string    `json:"language_code"`
	IsBot        bool      `json:"is_bot"`
	LastActiveAt time.Time `json:"last_active_at"`
}

func (r *RequestUserTelegram) ToUserEntity() *UserEntity {
	return &UserEntity{
		TelegramID:   r.ID,
		Username:     r.Username,
		FirstName:    r.FirstName,
		LastName:     r.LastName,
		LanguageCode: r.LanguageCode,
		IsBot:        r.IsBot,
		LastActiveAt: r.LastActiveAt,
	}
}

func ToRequestUserTelegram(user *telebot.User) *RequestUserTelegram {
	return &RequestUserTelegram{
		ID:           user.ID,
		Username:     user.Username,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		LanguageCode: user.LanguageCode,
		IsBot:        user.IsBot,
		LastActiveAt: utils.TimeNowWIB(),
	}
}

type RequestSetPositionData struct {
	Symbol       string
	BuyPrice     float64
	BuyDate      string
	TakeProfit   float64
	StopLoss     float64
	MaxHolding   int
	AlertPrice   bool
	AlertMonitor bool
	UserTelegram *RequestUserTelegram
}

func (r *RequestSetPositionData) ToStockPositionEntity() *StockPositionEntity {
	return &StockPositionEntity{
		StockCode:            r.Symbol,
		BuyPrice:             r.BuyPrice,
		BuyDate:              utils.MustParseDate(r.BuyDate),
		TakeProfitPrice:      r.TakeProfit,
		StopLossPrice:        r.StopLoss,
		MaxHoldingPeriodDays: r.MaxHolding,
		PriceAlert:           r.AlertPrice,
		MonitorPosition:      r.AlertMonitor,
	}
}

type RequestAnalysisPositionData struct {
	Symbol   string
	BuyPrice float64
	BuyDate  string
	MaxDays  int
	Interval string
	Period   string
}
