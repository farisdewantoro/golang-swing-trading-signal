package config

import (
	"golang-swing-trading-signal/pkg/postgres"
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig       `mapstructure:"server"`
	Yahoo    YahooFinanceConfig `mapstructure:"yahoo"`
	Gemini   GeminiConfig       `mapstructure:"gemini"`
	Trading  TradingConfig      `mapstructure:"trading"`
	Telegram TelegramConfig     `mapstructure:"telegram"`
	Database postgres.Config    `mapstructure:"database"`
	Log      LogConfig          `mapstructure:"log"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

type ServerConfig struct {
	Port string
	Env  string
}

type YahooFinanceConfig struct {
	BaseURL string
}

type GeminiConfig struct {
	APIKey              string
	BaseURL             string
	Model               string
	MaxRequestPerMinute int
	MaxTokenPerMinute   int
	RequestTemperature  float64
}

type TradingConfig struct {
	DefaultMaxHoldingPeriodDays int
	ConfidenceThreshold         int
	StockList                   []string
}

type TelegramConfig struct {
	BotToken               string
	ChatID                 string
	WebhookURL             string
	TimeoutDuration        time.Duration
	TimeoutBuyListDuration time.Duration
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Println("Failed to read config file .env config try read from environment variables")
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
			APIKey:              viper.GetString("GEMINI_API_KEY"),
			BaseURL:             viper.GetString("GEMINI_BASE_URL"),
			Model:               viper.GetString("GEMINI_MODEL"),
			MaxRequestPerMinute: viper.GetInt("GEMINI_MAX_REQUEST_PER_MINUTE"),
			MaxTokenPerMinute:   viper.GetInt("GEMINI_MAX_TOKEN_PER_MINUTE"),
			RequestTemperature:  viper.GetFloat64("GEMINI_REQUEST_TEMPERATURE"),
		},
		Trading: TradingConfig{
			DefaultMaxHoldingPeriodDays: viper.GetInt("DEFAULT_MAX_HOLDING_PERIOD_DAYS"),
			ConfidenceThreshold:         viper.GetInt("CONFIDENCE_THRESHOLD"),
			StockList:                   stockList,
		},
		Log: LogConfig{
			Level: viper.GetString("LOG_LEVEL"),
		},
		Telegram: TelegramConfig{
			BotToken:               viper.GetString("TELEGRAM_BOT_TOKEN"),
			ChatID:                 viper.GetString("TELEGRAM_CHAT_ID"),
			WebhookURL:             viper.GetString("TELEGRAM_WEBHOOK_URL"),
			TimeoutDuration:        viper.GetDuration("TELEGRAM_TIMEOUT_DURATION"),
			TimeoutBuyListDuration: viper.GetDuration("TELEGRAM_TIMEOUT_BUY_LIST_DURATION"),
		},
		Database: postgres.Config{
			Host:            viper.GetString("DATABASE_HOST"),
			Port:            viper.GetInt("DATABASE_PORT"),
			User:            viper.GetString("DATABASE_USER"),
			Password:        viper.GetString("DATABASE_PASSWORD"),
			DBName:          viper.GetString("DATABASE_NAME"),
			SSLMode:         viper.GetString("DATABASE_SSL_MODE"),
			TimeZone:        viper.GetString("DATABASE_TIME_ZONE"),
			MaxIdleConns:    viper.GetInt("DATABASE_MAX_IDLE_CONNS"),
			MaxOpenConns:    viper.GetInt("DATABASE_MAX_OPEN_CONNS"),
			ConnMaxLifetime: viper.GetString("DATABASE_CONN_MAX_LIFETIME"),
			LogLevel:        viper.GetString("DATABASE_LOG_LEVEL"),
		},
	}

	return config, nil
}
