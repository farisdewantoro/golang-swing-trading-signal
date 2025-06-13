package config

import (
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig
	Yahoo    YahooFinanceConfig
	Gemini   GeminiConfig
	Trading  TradingConfig
	Telegram TelegramConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

type YahooFinanceConfig struct {
	BaseURL string
}

type GeminiConfig struct {
	APIKey  string
	BaseURL string
	Model   string
}

type TradingConfig struct {
	DefaultMaxHoldingPeriodDays int
	ConfidenceThreshold         int
	StockList                   []string
}

type TelegramConfig struct {
	BotToken   string
	ChatID     string
	WebhookURL string
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// If .env file doesn't exist, use environment variables
		viper.SetDefault("PORT", "8080")
		viper.SetDefault("ENV", "development")
		viper.SetDefault("YAHOO_FINANCE_BASE_URL", "https://query1.finance.yahoo.com/v8/finance/chart")
		viper.SetDefault("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/models")
		viper.SetDefault("GEMINI_MODEL", "gemini-pro")
		viper.SetDefault("DEFAULT_MAX_HOLDING_PERIOD_DAYS", 5)
		viper.SetDefault("CONFIDENCE_THRESHOLD", 70)
		viper.SetDefault("STOCK_LIST", "BBCA,BBRI,ANTM,ASII,ICBP,INDF,KLBF,PGAS,PTBA,SMGR,TLKM,UNTR,UNVR,WSKT")
		viper.SetDefault("TELEGRAM_WEBHOOK_URL", "")
	}

	// Parse stock list from comma-separated string
	stockListStr := viper.GetString("STOCK_LIST")
	var stockList []string
	if stockListStr != "" {
		stockList = strings.Split(stockListStr, ",")
		// Trim whitespace from each symbol
		for i, symbol := range stockList {
			stockList[i] = strings.TrimSpace(symbol)
		}
	}

	config := &Config{
		Server: ServerConfig{
			Port: viper.GetString("PORT"),
			Env:  viper.GetString("ENV"),
		},
		Yahoo: YahooFinanceConfig{
			BaseURL: viper.GetString("YAHOO_FINANCE_BASE_URL"),
		},
		Gemini: GeminiConfig{
			APIKey:  viper.GetString("GEMINI_API_KEY"),
			BaseURL: viper.GetString("GEMINI_BASE_URL"),
			Model:   viper.GetString("GEMINI_MODEL"),
		},
		Trading: TradingConfig{
			DefaultMaxHoldingPeriodDays: viper.GetInt("DEFAULT_MAX_HOLDING_PERIOD_DAYS"),
			ConfidenceThreshold:         viper.GetInt("CONFIDENCE_THRESHOLD"),
			StockList:                   stockList,
		},
		Telegram: TelegramConfig{
			BotToken:   viper.GetString("TELEGRAM_BOT_TOKEN"),
			ChatID:     viper.GetString("TELEGRAM_CHAT_ID"),
			WebhookURL: viper.GetString("TELEGRAM_WEBHOOK_URL"),
		},
	}

	return config, nil
}
