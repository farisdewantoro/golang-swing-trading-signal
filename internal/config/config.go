package config

import (
	"golang-swing-trading-signal/pkg/postgres"
	"golang-swing-trading-signal/pkg/redis"
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
	Redis    redis.Config       `mapstructure:"redis"`
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
	GetLatestSignalBefore       time.Duration
	GetBuyListSignalBefore      time.Duration
}

type TelegramConfig struct {
	BotToken                  string
	ChatID                    string
	WebhookURL                string
	TimeoutDuration           time.Duration
	TimeoutBuyListDuration    time.Duration
	MaxGlobalRequestPerSecond int
	MaxUserRequestPerSecond   int
	MaxEditMessagePerSecond   int
	RatelimitExpireDuration   time.Duration
	RateLimitCleanupDuration  time.Duration
	FeatureNewsMaxAgeInDays   int
	FeatureNewsLimitStockNews int
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
			GetLatestSignalBefore:       viper.GetDuration("GET_LATEST_SIGNAL_BEFORE"),
			GetBuyListSignalBefore:      viper.GetDuration("GET_BUY_LIST_SIGNAL_BEFORE"),
		},
		Log: LogConfig{
			Level: viper.GetString("LOG_LEVEL"),
		},
		Telegram: TelegramConfig{
			BotToken:                  viper.GetString("TELEGRAM_BOT_TOKEN"),
			ChatID:                    viper.GetString("TELEGRAM_CHAT_ID"),
			WebhookURL:                viper.GetString("TELEGRAM_WEBHOOK_URL"),
			TimeoutDuration:           viper.GetDuration("TELEGRAM_TIMEOUT_DURATION"),
			TimeoutBuyListDuration:    viper.GetDuration("TELEGRAM_TIMEOUT_BUY_LIST_DURATION"),
			MaxGlobalRequestPerSecond: viper.GetInt("TELEGRAM_MAX_GLOBAL_REQUEST_PER_SECOND"),
			MaxUserRequestPerSecond:   viper.GetInt("TELEGRAM_MAX_USER_REQUEST_PER_SECOND"),
			MaxEditMessagePerSecond:   viper.GetInt("TELEGRAM_MAX_EDIT_MESSAGE_PER_SECOND"),
			RatelimitExpireDuration:   viper.GetDuration("TELEGRAM_RATELIMIT_EXPIRE_DURATION"),
			RateLimitCleanupDuration:  viper.GetDuration("TELEGRAM_RATE_LIMIT_CLEANUP_DURATION"),
			FeatureNewsMaxAgeInDays:   viper.GetInt("TELEGRAM_FEATURE_NEWS_MAX_AGE_IN_DAYS"),
			FeatureNewsLimitStockNews: viper.GetInt("TELEGRAM_FEATURE_NEWS_LIMIT_STOCK_NEWS"),
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
		Redis: redis.Config{
			Host:     viper.GetString("REDIS_HOST"),
			Port:     viper.GetInt("REDIS_PORT"),
			Password: viper.GetString("REDIS_PASSWORD"),
			DB:       viper.GetInt("REDIS_DB"),
			PoolSize: viper.GetInt("REDIS_POOL_SIZE"),
		},
	}

	return config, nil
}
