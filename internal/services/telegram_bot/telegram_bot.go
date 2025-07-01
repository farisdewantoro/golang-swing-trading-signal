package telegram_bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/telebot.v3"

	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/services/stocks"
	"golang-swing-trading-signal/internal/services/trading_analysis"
	"golang-swing-trading-signal/pkg/ratelimit"
)

// Conversation states
const (
	StateIdle = iota // 0

	// /setposition states
	StateWaitingSetPositionSymbol       = 1
	StateWaitingSetPositionBuyPrice     = 2
	StateWaitingSetPositionBuyDate      = 3
	StateWaitingSetPositionTakeProfit   = 4
	StateWaitingSetPositionStopLoss     = 5
	StateWaitingSetPositionMaxHolding   = 6
	StateWaitingSetPositionAlertPrice   = 7
	StateWaitingSetPositionAlertMonitor = 8

	// /analyze position states
	StateWaitingAnalysisPositionSymbol   = 10
	StateWaitingAnalysisPositionBuyPrice = 11
	StateWaitingAnalysisPositionBuyDate  = 12
	StateWaitingAnalysisPositionMaxDays  = 13
	StateWaitingAnalysisPositionInterval = 14
	StateWaitingAnalysisPositionPeriod   = 15

	// /analyze main flow states
	StateWaitingAnalyzeSymbol = 20
	StateWaitingAnalysisType  = 21

	// exit position states
	StateWaitingExitPositionInputExitPrice = 30
	StateWaitingExitPositionInputExitDate  = 31
	StateWaitingExitPositionConfirm        = 32

	StateWaitingNewsFindSymbol                  = 40
	StateWaitingNewsFindSendSummaryConfirmation = 41

	StateWaitingAdjustTargetPositionInputTargetPrice   = 50
	StateWaitingAdjustTargetPositionInputStopLossPrice = 51
	StateWaitingAdjustTargetPositionMaxHoldingDays     = 52
	StateWaitingAdjustTargetPositionConfirm            = 53
)

type TelegramBotService struct {
	bot                          *telebot.Bot
	telegramRateLimiter          *ratelimit.TelegramRateLimiter
	config                       *config.TelegramConfig
	tradingConfig                *config.TradingConfig
	logger                       *logrus.Logger
	analyzer                     *trading_analysis.Analyzer
	stockService                 stocks.StockService
	router                       *gin.Engine
	userStates                   map[int64]int                                     // UserID -> State
	userPositionData             map[int64]*models.RequestSetPositionData          // UserID -> Data for /setposition
	userAnalysisPositionData     map[int64]*models.RequestAnalysisPositionData     // UserID -> Data for /analyze
	userExitPositionData         map[int64]*models.RequestExitPositionData         // UserID -> Data for /exitposition
	userAdjustTargetPositionData map[int64]*models.RequestAdjustTargetPositionData // UserID -> Data for /adjusttargetposition
	mu                           sync.Mutex                                        // Mutex for thread-safe operations
	userCancelFuncs              map[int64]context.CancelFunc                      // key: telegram user ID atau chat ID
	ctx                          context.Context
}

func NewTelegramBotService(
	cfg *config.TelegramConfig,
	ctx context.Context,
	tradingConfig *config.TradingConfig,
	logger *logrus.Logger,
	analyzer *trading_analysis.Analyzer,
	stockService stocks.StockService,
	bot *telebot.Bot,
	telegramRateLimiter *ratelimit.TelegramRateLimiter,
	router *gin.Engine) *TelegramBotService {

	service := &TelegramBotService{
		bot:                          bot,
		telegramRateLimiter:          telegramRateLimiter,
		config:                       cfg,
		tradingConfig:                tradingConfig,
		logger:                       logger,
		analyzer:                     analyzer,
		stockService:                 stockService,
		router:                       router,
		userStates:                   make(map[int64]int),
		userPositionData:             make(map[int64]*models.RequestSetPositionData),
		userAnalysisPositionData:     make(map[int64]*models.RequestAnalysisPositionData),
		userExitPositionData:         make(map[int64]*models.RequestExitPositionData),
		userAdjustTargetPositionData: make(map[int64]*models.RequestAdjustTargetPositionData),
		mu:                           sync.Mutex{},
		userCancelFuncs:              make(map[int64]context.CancelFunc),
		ctx:                          ctx,
	}

	// Register handlers
	service.registerHandlers()

	return service
}

func (t *TelegramBotService) Start() {
	t.logger.Info("Starting Telegram bot...")

	t.RegisterMiddleware()

	// If webhook URL is configured, set it up
	if t.config.WebhookURL != "" {
		t.logger.WithField("webhook_url", t.config.WebhookURL).Info("Setting up webhook...")

		// Set webhook URL directly using HTTP client to avoid telebot library issues
		client := &http.Client{Timeout: 10 * time.Second}

		// Create form data
		data := url.Values{}
		data.Set("url", t.config.WebhookURL)

		// Make request to Telegram API
		resp, err := client.PostForm(fmt.Sprintf("https://api.telegram.org/bot%s/setWebhook", t.config.BotToken), data)
		if err != nil {
			t.logger.WithError(err).Error("Failed to call Telegram API for webhook setup")
			return
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.logger.WithError(err).Error("Failed to read webhook response")
			return
		}

		// Check if successful
		if resp.StatusCode == http.StatusOK {
			t.logger.WithField("webhook_url", t.config.WebhookURL).Info("Webhook set successfully")
		} else {
			t.logger.WithError(fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))).Error("Failed to set webhook, falling back to long polling")
		}
	} else {
		t.logger.Info("No webhook URL configured, using long polling")
	}
}

func (t *TelegramBotService) Stop() {
	t.logger.Info("Stopping Telegram bot...")

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(t.ctx, 10*time.Second)
	defer cancel()

	// Stop the bot with timeout
	stopDone := make(chan error, 1)
	go func() {
		// Use a separate goroutine to avoid blocking
		t.bot.Stop()
		stopDone <- nil
	}()

	// Wait for bot to stop with timeout
	select {
	case <-stopDone:
		t.logger.Info("Telegram bot stopped successfully")
	case <-ctx.Done():
		t.logger.Warn("Timeout while stopping bot, forcing shutdown")
	}

	t.logger.Info("Telegram bot shutdown completed")
}

func (t *TelegramBotService) CleanUpUsersStates() {
	t.userStates = make(map[int64]int)
	t.userPositionData = make(map[int64]*models.RequestSetPositionData)
	t.userAnalysisPositionData = make(map[int64]*models.RequestAnalysisPositionData)
}

func (t *TelegramBotService) ResetUserState(userID int64) {
	t.mu.Lock()
	delete(t.userStates, userID)
	delete(t.userPositionData, userID)
	delete(t.userAnalysisPositionData, userID)
	delete(t.userExitPositionData, userID)
	delete(t.userAdjustTargetPositionData, userID)

	if cancel, exists := t.userCancelFuncs[userID]; exists {
		cancel()
		delete(t.userCancelFuncs, userID)
	}
	t.mu.Unlock()
}

func (t *TelegramBotService) SendPositionMonitoringNotification(position *models.PositionMonitoringResponseMultiTimeframe) error {
	if t.config.ChatID == "" {
		t.logger.Warn("Telegram chat ID not configured, skipping notification")
		return nil
	}

	chatID, err := strconv.ParseInt(t.config.ChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	message := t.FormatPositionMonitoringMessage(position)

	_, err = t.bot.Send(&telebot.Chat{ID: chatID}, message, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	})

	if err != nil {
		t.logger.WithError(err).Error("Failed to send position monitoring notification")
		return err
	}

	t.logger.WithField("symbol", position.Symbol).Info("Position monitoring notification sent")
	return nil
}
