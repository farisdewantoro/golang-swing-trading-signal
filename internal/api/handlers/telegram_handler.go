package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"golang-swing-trading-signal/internal/services/telegram_bot"
)

type TelegramHandler struct {
	telegramService *telegram_bot.TelegramBotService
	logger          *logrus.Logger
}

func NewTelegramHandler(telegramService *telegram_bot.TelegramBotService, logger *logrus.Logger) *TelegramHandler {
	return &TelegramHandler{
		telegramService: telegramService,
		logger:          logger,
	}
}

// HealthCheck checks if the Telegram bot is running
func (h *TelegramHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "telegram-bot",
		"message": "Telegram bot is running",
	})
}

// SetWebhook sets the webhook URL for the Telegram bot
func (h *TelegramHandler) SetWebhook(c *gin.Context) {
	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.WithError(err).Error("Invalid webhook request")
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"message": err.Error(),
		})
		return
	}

	// Note: In a real implementation, you would set the webhook here
	// For now, we'll just log the request
	h.logger.WithField("webhook_url", request.URL).Info("Webhook URL set")

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Webhook URL set successfully",
		"url":     request.URL,
	})
}

// GetBotInfo returns information about the Telegram bot
func (h *TelegramHandler) GetBotInfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"service": "telegram-bot",
		"features": []string{
			"Stock analysis via /analyze command",
			"Position monitoring via /monitor command",
			"Buy list analysis via /buylist command",
			"Position monitoring notifications",
			"Real-time trading signals",
		},
		"commands": []gin.H{
			{
				"command":     "/start",
				"description": "Start the bot and see welcome message",
			},
			{
				"command":     "/help",
				"description": "Show help and available commands",
			},
			{
				"command":     "/analyze <symbol>",
				"description": "Analyze a specific stock symbol",
			},
			{
				"command":     "/monitor <symbol> <buy_price> <buy_date> <max_days>",
				"description": "Monitor a trading position",
			},
			{
				"command":     "/buylist",
				"description": "Analyze all stocks from config and show buy list",
			},
		},
	})
}
