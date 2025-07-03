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
	"google.golang.org/genai"
	"gopkg.in/telebot.v3"

	"golang-swing-trading-signal/internal/api/handlers"
	"golang-swing-trading-signal/internal/api/routes"
	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/repository"
	"golang-swing-trading-signal/internal/services/gemini_ai"
	"golang-swing-trading-signal/internal/services/jobs"
	"golang-swing-trading-signal/internal/services/stocks"
	"golang-swing-trading-signal/internal/services/telegram_bot"
	"golang-swing-trading-signal/internal/services/trading_analysis"
	"golang-swing-trading-signal/internal/services/yahoo_finance"
	"golang-swing-trading-signal/pkg/postgres"
	"golang-swing-trading-signal/pkg/ratelimit"
	"golang-swing-trading-signal/pkg/redis"
)

func main() {
	ctxCancel, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Initialize logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.WithError(err).Fatal("Failed to load configuration")
	}

	logrusLevel, err := logrus.ParseLevel(cfg.Log.Level)
	if err != nil {
		logger.WithError(err).Fatal("Failed to parse log level")
	}

	logger.SetLevel(logrusLevel)

	// Set Gin mode based on environment
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

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

	// Initialize database
	db, err := postgres.NewDB(cfg.Database)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize database")
	}
	if sqlDB, err := db.DB.DB(); err == nil {
		defer sqlDB.Close()
	}

	// Initialize repositories
	stockNewsSummaryRepo := repository.NewStockNewsSummaryRepository(db.DB)
	stockPositionRepo := repository.NewStockPositionRepository(db.DB)
	stockNewsRepo := repository.NewStocksNewsRepository(db.DB)
	stockRepo := repository.NewStocksRepository(db.DB)
	userRepo := repository.NewUserRepository(db.DB)
	unitOfWork := repository.NewUnitOfWork(db.DB)
	stockSignalRepo := repository.NewStockSignalRepository(db.DB)
	jobsRepository := repository.NewJobsRepository(db.DB)
	stockPositionMonitoringRepo := repository.NewStockPositionMonitoringRepository(db.DB)
	genClient, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: cfg.Gemini.APIKey,
	})
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Gemini client")
	}
	redisClient, err := redis.NewClient(cfg.Redis)
	if err != nil {
		logger.WithError(err).Fatal("Failed to initialize Redis client")
	}

	// Initialize services
	yahooClient := yahoo_finance.NewClient(&cfg.Yahoo, logger)
	geminiClient := gemini_ai.NewClient(&cfg.Gemini, logger, genClient)
	analyzer := trading_analysis.NewAnalyzer(yahooClient, geminiClient, logger, stockNewsSummaryRepo, stockPositionRepo, userRepo, unitOfWork)

	// Initialize Telegram bot service

	if cfg.Telegram.BotToken == "" {
		logger.Fatal("telegram bot token is required")
	}

	// Always start with long polling to avoid webhook conflicts
	pref := telebot.Settings{
		Token:  cfg.Telegram.BotToken,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		OnError: func(err error, c telebot.Context) {
			logger.WithError(err).Error("Telegram bot error")
		},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		logger.WithError(err).Fatal("failed to create telegram bot")
	}
	telegramRateLimiter := ratelimit.NewTelegramRateLimiter(&cfg.Telegram, logger, bot)
	telegramRateLimiter.StartCleanupExpired(ctxCancel)

	stockService := stocks.NewStockService(cfg, stockRepo, stockNewsSummaryRepo, stockPositionRepo, userRepo, logger, unitOfWork, stockNewsRepo, stockSignalRepo, stockPositionMonitoringRepo, redisClient)
	jobService := jobs.NewJobService(cfg, logger, jobsRepository)
	telegramService := telegram_bot.NewTelegramBotService(&cfg.Telegram, ctxCancel, &cfg.Trading, logger, analyzer, stockService, jobService, redisClient, bot, telegramRateLimiter, router)

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

	// Stop server
	cancel()

	telegramRateLimiter.StopCleanupExpired()
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

	logger.Info("Shutting down HTTP server...")
	if err := server.Shutdown(ctxCancel); err != nil {
		logger.WithError(err).Error("Server forced to shutdown")
	} else {
		logger.Info("HTTP server shutdown completed successfully")
	}

	logger.Info("Server exited")
}
