package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"golang-swing-trading-signal/internal/api/handlers"
	"golang-swing-trading-signal/internal/api/routes"
	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/services/gemini_ai"
	"golang-swing-trading-signal/internal/services/telegram_bot"
	"golang-swing-trading-signal/internal/services/trading_analysis"
	"golang-swing-trading-signal/internal/services/yahoo_finance"
)

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	// Set Gin mode based on environment
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize services
	yahooClient := yahoo_finance.NewClient(&cfg.Yahoo)
	geminiClient := gemini_ai.NewClient(&cfg.Gemini)
	analyzer := trading_analysis.NewAnalyzer(yahooClient, geminiClient)

	// Initialize router
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Setup CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Initialize Telegram bot service
	var telegramService *telegram_bot.TelegramBotService
	if cfg.Telegram.BotToken != "" {
		telegramService, err = telegram_bot.NewTelegramBotService(&cfg.Telegram, &cfg.Trading, logger, analyzer, router)
		if err != nil {
			logger.WithError(err).Warn("Failed to initialize Telegram bot service, continuing without Telegram integration")
		} else {
			logger.Info("Telegram bot service initialized successfully")
		}
	} else {
		logger.Warn("Telegram bot token not configured, skipping Telegram integration")
	}

	// Initialize handlers
	tradingHandler := handlers.NewTradingHandler(analyzer, telegramService, logger, cfg)
	telegramHandler := handlers.NewTelegramHandler(telegramService, logger)

	// Setup routes
	routes.SetupRoutes(router, tradingHandler, telegramHandler)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// Start Telegram bot in a goroutine if available
	if telegramService != nil {
		go func() {
			logger.Info("Starting Telegram bot...")
			telegramService.Start()
		}()
	}

	// Start server in a goroutine
	go func() {
		logger.WithField("port", cfg.Server.Port).Info("Starting server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Stop Telegram bot if running with timeout
	if telegramService != nil {
		logger.Info("Stopping Telegram bot...")
		telegramDone := make(chan struct{})
		go func() {
			telegramService.Stop()
			close(telegramDone)
		}()

		// Wait for Telegram bot to stop with timeout
		select {
		case <-telegramDone:
			logger.Info("Telegram bot stopped successfully")
		case <-time.After(15 * time.Second):
			logger.Warn("Timeout waiting for Telegram bot to stop, proceeding with server shutdown")
		}
	}

	// Give outstanding requests a deadline for completion
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("Shutting down HTTP server...")
	if err := server.Shutdown(ctx); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	} else {
		logger.Info("HTTP server shutdown completed successfully")
	}

	logger.Info("Server exited")
}
