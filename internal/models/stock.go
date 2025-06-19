package models

import (
	"time"

	"github.com/lib/pq"
)

type StockPositionEntity struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	UserID                uint       `gorm:"not null" json:"user_id"`
	StockCode             string     `gorm:"not null" json:"stock_code"`
	BuyPrice              float64    `gorm:"not null" json:"buy_price"`
	TakeProfitPrice       float64    `gorm:"not null" json:"take_profit_price"`
	StopLossPrice         float64    `gorm:"not null" json:"stop_loss_price"`
	BuyDate               time.Time  `gorm:"not null" json:"buy_date"`
	MaxHoldingPeriodDays  int        `gorm:"not null" json:"max_holding_period_days"`
	IsActive              bool       `gorm:"not null" json:"is_active"`
	ExitPrice             *float64   `json:"exit_price"`
	ExitDate              *time.Time `json:"exit_date"`
	PriceAlert            bool       `gorm:"not null" json:"price_alert"`
	LastPriceAlertAt      *time.Time `json:"last_price_alert_at"`
	MonitorPosition       bool       `gorm:"not null" json:"monitor_position"`
	LastMonitorPositionAt *time.Time `json:"last_monitor_position_at"`
	User                  UserEntity `gorm:"foreignKey:UserID;references:ID"`
	CreatedAt             time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (StockPositionEntity) TableName() string {
	return "stock_positions"
}

type StockPositionQueryParam struct {
	TelegramIDs []int64  `json:"telegram_ids"`
	StockCodes  []string `json:"stock_codes"`
	IsActive    bool     `json:"is_active"`
}

// StockNewsSummary represents a summary of news articles for a specific stock.
type StockNewsSummaryEntity struct {
	ID                     uint           `gorm:"primaryKey" json:"id"`
	StockCode              string         `gorm:"type:varchar(50);not null" json:"stock_code"`
	SummarySentiment       string         `gorm:"type:varchar(50)" json:"summary_sentiment"`
	SummaryImpact          string         `gorm:"type:varchar(50)" json:"summary_impact"`
	SummaryConfidenceScore float64        `json:"summary_confidence_score"`
	KeyIssues              pq.StringArray `gorm:"type:text[]" json:"key_issues"`
	SuggestedAction        string         `gorm:"type:varchar(10)" json:"suggested_action"`
	Reasoning              string         `gorm:"type:text" json:"reasoning"`
	ShortSummary           string         `gorm:"type:text" json:"short_summary"`
	SummaryStart           time.Time      `json:"summary_start"`
	SummaryEnd             time.Time      `json:"summary_end"`
	CreatedAt              time.Time      `gorm:"autoCreateTime" json:"created_at"`
}

// TableName specifies the table name for the StockNewsSummary model.
func (StockNewsSummaryEntity) TableName() string {
	return "stock_news_summary"
}
