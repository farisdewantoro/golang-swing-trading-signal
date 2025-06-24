package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/services/telegram_bot"
	"golang-swing-trading-signal/internal/services/trading_analysis"
)

type TradingHandler struct {
	analyzer        *trading_analysis.Analyzer
	telegramService *telegram_bot.TelegramBotService
	logger          *logrus.Logger
	cfg             *config.Config
}

func NewTradingHandler(analyzer *trading_analysis.Analyzer, telegramService *telegram_bot.TelegramBotService, logger *logrus.Logger, cfg *config.Config) *TradingHandler {
	return &TradingHandler{
		analyzer:        analyzer,
		telegramService: telegramService,
		logger:          logger,
		cfg:             cfg,
	}
}

// HealthCheck handles GET /health
func (h *TradingHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "swing-trading-signal",
	})
}
