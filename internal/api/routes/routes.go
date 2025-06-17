package routes

import (
	"github.com/gin-gonic/gin"

	"golang-swing-trading-signal/internal/api/handlers"
)

func SetupRoutes(router *gin.Engine, tradingHandler *handlers.TradingHandler, telegramHandler *handlers.TelegramHandler) {
	// Health check
	router.GET("/health", tradingHandler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Trading analysis endpoints
		v1.GET("/analyze", tradingHandler.AnalyzeStock)
		v1.GET("/analyze-all", tradingHandler.AnalyzeAllStocks)
		v1.POST("/monitor-position", tradingHandler.MonitorPosition)
		v1.POST("/bulk-monitor-position", tradingHandler.BulkMonitorPosition)

		// Telegram bot endpoints (basic info only, webhook handled by Telegram service)
		telegram := v1.Group("/telegram")
		{
			telegram.GET("/health", telegramHandler.HealthCheck)
			telegram.GET("/info", telegramHandler.GetBotInfo)
		}
	}
}
