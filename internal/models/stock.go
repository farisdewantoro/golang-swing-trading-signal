package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type StockEntity struct {
	Code      string         `gorm:"primaryKey;" json:"code"`
	Name      string         `gorm:"not null" json:"name"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"autoDeleteTime" json:"deleted_at"`
}

func (StockEntity) TableName() string {
	return "stocks"
}

type StockPositionEntity struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	UserID                uint       `gorm:"not null" json:"user_id"`
	StockCode             string     `gorm:"not null" json:"stock_code"`
	BuyPrice              float64    `gorm:"not null" json:"buy_price"`
	TakeProfitPrice       float64    `gorm:"not null" json:"take_profit_price"`
	StopLossPrice         float64    `gorm:"not null" json:"stop_loss_price"`
	BuyDate               time.Time  `gorm:"not null" json:"buy_date"`
	MaxHoldingPeriodDays  int        `gorm:"not null" json:"max_holding_period_days"`
	IsActive              *bool      `gorm:"not null" json:"is_active"`
	ExitPrice             *float64   `json:"exit_price"`
	ExitDate              *time.Time `json:"exit_date"`
	PriceAlert            *bool      `gorm:"not null" json:"price_alert"`
	LastPriceAlertAt      *time.Time `json:"last_price_alert_at"`
	MonitorPosition       *bool      `gorm:"not null" json:"monitor_position"`
	LastMonitorPositionAt *time.Time `json:"last_monitor_position_at"`
	User                  UserEntity `gorm:"foreignKey:UserID;references:ID"`
	CreatedAt             time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (StockPositionEntity) TableName() string {
	return "stock_positions"
}

type StockPositionUpdateRequest struct {
	BuyPrice             *float64   `json:"buy_price"`
	BuyDate              *time.Time `json:"buy_date"`
	MaxHoldingPeriodDays *int       `json:"max_holding_period_days"`
	PriceAlert           *bool      `json:"price_alert"`
	MonitorPosition      *bool      `json:"monitor_position"`
	ExitPrice            *float64   `json:"exit_price"`
	ExitDate             *time.Time `json:"exit_date"`
	IsActive             *bool      `json:"is_active"`
}

type StockPositionQueryParam struct {
	IDs         []uint   `json:"ids"`
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

type GetStocksParam struct {
	StockCodes []string `json:"stock_codes"`
}

// StockNews represents a news article related to stocks.
type StockNewsEntity struct {
	ID             uint                 `gorm:"primaryKey" json:"id"`
	Title          string               `gorm:"not null" json:"title"`
	Link           string               `gorm:"unique;not null" json:"link"`
	PublishedAt    *time.Time           `json:"published_at,omitempty"`
	RawContent     string               `json:"raw_content"`
	Summary        string               `json:"summary"`
	HashIdentifier string               `gorm:"unique;not null" json:"hash_identifier"`
	Source         string               `json:"source"`
	GoogleRSSLink  string               `json:"google_rss_link"`
	ImpactScore    float64              `json:"impact_score"`
	KeyIssue       pq.StringArray       `gorm:"key_issue;type:text[]" json:"key_issue"`
	CreatedAt      time.Time            `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time            `gorm:"autoUpdateTime" json:"updated_at"`
	StockMentions  []StockMentionEntity `gorm:"foreignKey:StockNewsID" json:"stock_mentions"`

	// Fields populated by custom query for ranking
	StockCode       string  `gorm:"-" json:"stock_code,omitempty"`
	Sentiment       string  `gorm:"-" json:"sentiment,omitempty"`
	Impact          string  `gorm:"-" json:"impact,omitempty"`
	ConfidenceScore float64 `gorm:"-" json:"confidence_score,omitempty"`
	FinalScore      float64 `gorm:"-" json:"final_score,omitempty"`
}

// TableName specifies the table name for the StockNews model.
func (StockNewsEntity) TableName() string {
	return "stock_news"
}

// StockMention represents a mention of a stock in a news article.
type StockMentionEntity struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	StockNewsID     uint      `json:"stock_news_id"`
	StockCode       string    `gorm:"not null" json:"stock_code"`
	Sentiment       string    `gorm:"not null" json:"sentiment"`
	Impact          string    `gorm:"not null" json:"impact"`
	ConfidenceScore float64   `gorm:"not null" json:"confidence_score"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (StockMentionEntity) TableName() string {
	return "stock_mentions"
}

type StockNewsQueryParam struct {
	StockCodes       []string `json:"stock_codes"`
	Limit            int      `json:"limit"`
	MaxNewsAgeInDays int      `json:"max_news_age_in_days"`
	PriorityDomains  []string `json:"priority_domains"`
}
