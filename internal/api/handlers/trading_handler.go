package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/models"
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

// AnalyzeStock handles GET /analyze?symbol={SYMBOL}
func (h *TradingHandler) AnalyzeStock(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "MISSING_SYMBOL",
			Message: "Symbol parameter is required",
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validate symbol
	if err := h.analyzer.ValidateSymbol(symbol); err != nil {
		h.logger.WithError(err).WithField("symbol", symbol).Error("Invalid symbol")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "INVALID_SYMBOL",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Perform analysis
	analysis, err := h.analyzer.AnalyzeStock(symbol)
	if err != nil {
		h.logger.WithError(err).WithField("symbol", symbol).Error("Failed to analyze stock")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "ANALYSIS_FAILED",
			Message: "Failed to analyze stock: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	h.logger.WithField("symbol", symbol).Info("Stock analysis completed successfully")
	c.JSON(http.StatusOK, analysis)
}

// MonitorPosition handles POST /monitor-position
func (h *TradingHandler) MonitorPosition(c *gin.Context) {
	var request models.PositionMonitoringRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.logger.WithError(err).Error("Failed to bind request body")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: "Invalid request body: " + err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validate request
	if err := h.validatePositionRequest(&request); err != nil {
		h.logger.WithError(err).WithField("symbol", request.Symbol).Error("Invalid position request")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "INVALID_REQUEST",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validate symbol
	if err := h.analyzer.ValidateSymbol(request.Symbol); err != nil {
		h.logger.WithError(err).WithField("symbol", request.Symbol).Error("Invalid symbol")
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "INVALID_SYMBOL",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Perform position monitoring
	analysis, err := h.analyzer.MonitorPosition(request)
	if err != nil {
		h.logger.WithError(err).WithField("symbol", request.Symbol).Error("Failed to monitor position")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "MONITORING_FAILED",
			Message: "Failed to monitor position: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Send Telegram notification
	if h.telegramService != nil {
		go func() {
			if err := h.telegramService.SendPositionMonitoringNotification(analysis); err != nil {
				h.logger.WithError(err).WithField("symbol", request.Symbol).Error("Failed to send Telegram notification")
			} else {
				h.logger.WithField("symbol", request.Symbol).Info("Telegram notification sent successfully")
			}
		}()
	}

	h.logger.WithField("symbol", request.Symbol).Info("Position monitoring completed successfully")
	c.JSON(http.StatusOK, analysis)
}

// HealthCheck handles GET /health
func (h *TradingHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "swing-trading-signal",
	})
}

// validatePositionRequest validates the position monitoring request
func (h *TradingHandler) validatePositionRequest(request *models.PositionMonitoringRequest) error {
	if request.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if request.BuyPrice <= 0 {
		return fmt.Errorf("buy price must be greater than 0")
	}

	if request.BuyTime.IsZero() {
		return fmt.Errorf("buy time is required")
	}

	if request.BuyTime.After(time.Now()) {
		return fmt.Errorf("buy time cannot be in the future")
	}

	if request.MaxHoldingPeriodDays <= 0 {
		return fmt.Errorf("max holding period days must be greater than 0")
	}

	if request.MaxHoldingPeriodDays > 30 {
		return fmt.Errorf("max holding period days cannot exceed 30 days")
	}

	return nil
}

// AnalyzeAllStocks handles GET /analyze-all
func (h *TradingHandler) AnalyzeAllStocks(c *gin.Context) {

	// Perform analysis on all stocks
	summary, err := h.analyzer.AnalyzeAllStocks(h.cfg.Trading.StockList)
	if err != nil {
		h.logger.WithError(err).Error("Failed to analyze all stocks")
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "ANALYSIS_FAILED",
			Message: "Failed to analyze all stocks: " + err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	h.logger.WithField("total_stocks", summary.TotalStocks).Info("All stocks analysis completed successfully")
	c.JSON(http.StatusOK, summary)
}
