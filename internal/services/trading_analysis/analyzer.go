package trading_analysis

import (
	"context"
	"fmt"
	"time"

	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/repository"
	"golang-swing-trading-signal/internal/services/gemini_ai"
	"golang-swing-trading-signal/internal/services/yahoo_finance"
	"golang-swing-trading-signal/internal/utils"

	"github.com/sirupsen/logrus"
)

type Analyzer struct {
	yahooClient                *yahoo_finance.Client
	geminiClient               *gemini_ai.Client
	logger                     *logrus.Logger
	stockNewsSummaryRepository repository.StockNewsSummaryRepository
	stockPositionRepository    repository.StockPositionRepository
	userRepository             repository.UserRepository
	unitOfWork                 repository.UnitOfWork
}

func NewAnalyzer(yahooClient *yahoo_finance.Client, geminiClient *gemini_ai.Client, logger *logrus.Logger, stockNewsSummaryRepository repository.StockNewsSummaryRepository, stockPositionRepository repository.StockPositionRepository, userRepository repository.UserRepository, unitOfWork repository.UnitOfWork) *Analyzer {
	return &Analyzer{
		yahooClient:                yahooClient,
		geminiClient:               geminiClient,
		logger:                     logger,
		stockNewsSummaryRepository: stockNewsSummaryRepository,
		stockPositionRepository:    stockPositionRepository,
		userRepository:             userRepository,
		unitOfWork:                 unitOfWork,
	}
}

// AnalyzeStock performs complete analysis of a stock
func (a *Analyzer) AnalyzeStock(ctx context.Context, symbol string, interval, period string) (*models.IndividualAnalysisResponse, error) {
	// Get OHLC data from Yahoo Finance
	ohlcvDataWithInfo, err := a.yahooClient.GetRecentOHLCData(symbol, interval, period)
	if err != nil {
		return nil, fmt.Errorf("failed to get OHLC data: %w", err)
	}

	// Get last stock news summary
	beforeTime := utils.GetNowWithOnlyHour().Add(-time.Hour * 24 * 3)
	lastStockNewsSummary, err := a.stockNewsSummaryRepository.GetLast(ctx, beforeTime, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get last stock news summary: %w", err)
	}

	// Send to Gemini AI for analysis
	analysis, err := a.geminiClient.AnalyzeStock(ctx, symbol, ohlcvDataWithInfo.Data, ohlcvDataWithInfo.DataInfo, lastStockNewsSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze stock: %w", err)
	}

	return analysis, nil
}

// MonitorPosition monitors an existing trading position
func (a *Analyzer) MonitorPosition(ctx context.Context, request models.PositionMonitoringRequest) (*models.PositionMonitoringResponse, error) {
	// Get latest OHLC data from Yahoo Finance
	ohlcvDataWithInfo, err := a.yahooClient.GetRecentOHLCData(request.Symbol, request.Interval, request.Period)
	if err != nil {
		return nil, fmt.Errorf("failed to get OHLC data: %w", err)
	}

	// Get last stock news summary
	beforeTime := utils.GetNowWithOnlyHour().Add(-time.Hour * 24 * 3)
	lastStockNewsSummary, err := a.stockNewsSummaryRepository.GetLast(ctx, beforeTime, request.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get last stock news summary: %w", err)
	}

	// Send to Gemini AI for position analysis
	analysis, err := a.geminiClient.MonitorPosition(ctx, request, ohlcvDataWithInfo.Data, ohlcvDataWithInfo.DataInfo, lastStockNewsSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze position: %w", err)
	}

	// Set data info
	analysis.DataInfo = ohlcvDataWithInfo.DataInfo

	// Set buying price from request
	analysis.BuyPrice = request.BuyPrice
	analysis.CurrentPrice = ohlcvDataWithInfo.DataInfo.MarketPrice

	// Calculate position age
	positionAge := int(time.Since(request.BuyTime).Hours() / 24)
	analysis.PositionAgeDays = positionAge

	// Calculate days remaining
	daysRemaining := request.MaxHoldingPeriodDays - positionAge
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	// Update position metrics
	if analysis.PositionMetrics.DaysRemaining == 0 {
		analysis.PositionMetrics.DaysRemaining = daysRemaining
	}

	return analysis, nil
}

// BulkMonitorPosition monitors multiple positions at once
func (a *Analyzer) BulkMonitorPosition(ctx context.Context, requests []models.PositionMonitoringRequest) ([]models.PositionMonitoringResponse, []error) {
	var analyses []models.PositionMonitoringResponse
	var errors []error

	for _, request := range requests {
		analysis, err := a.MonitorPosition(ctx, request)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		analyses = append(analyses, *analysis)
	}

	return analyses, errors
}

// ValidateSymbol validates if the symbol is valid for Indonesian stocks
func (a *Analyzer) ValidateSymbol(ctx context.Context, symbol string) error {
	if symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}

	// Try to get data to validate symbol exists
	_, err := a.yahooClient.GetLatestOHLCData(symbol)
	if err != nil {
		return fmt.Errorf("invalid symbol or symbol not found: %s", symbol)
	}

	return nil
}

func (a *Analyzer) MonitorPositionTelegramUser(ctx context.Context, request *models.PositionMonitoringTelegramUserRequest) (*models.PositionMonitoringResponse, error) {
	a.logger.Debug("monitoring position", logrus.Fields{
		"telegram_id": request.TelegramID,
		"symbol":      request.Symbol,
	})
	positions, err := a.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{request.TelegramID},
		IsActive:    true,
		StockCodes:  []string{request.Symbol},
	})
	if err != nil {
		a.logger.Error("failed to get stock positions", logrus.Fields{
			"error": err,
		})
		return nil, fmt.Errorf("failed to get stock positions: %w", err)
	}

	if len(positions) == 0 {
		a.logger.Warn("position not found", logrus.Fields{
			"telegram_id": request.TelegramID,
			"symbol":      request.Symbol,
		})
		return nil, fmt.Errorf("position not found")
	}

	return a.MonitorPosition(ctx, models.PositionMonitoringRequest{
		Symbol:               request.Symbol,
		BuyPrice:             positions[0].BuyPrice,
		BuyTime:              positions[0].BuyDate,
		MaxHoldingPeriodDays: positions[0].MaxHoldingPeriodDays,
		Interval:             request.Interval,
		Period:               request.Period,
	})
}

func (a *Analyzer) CreateTelegramUserIfNotExist(ctx context.Context, req *models.RequestUserTelegram) error {
	user, err := a.userRepository.GetUserByTelegramID(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if user == nil {
		if err := a.userRepository.CreateUser(ctx, req.ToUserEntity()); err != nil {
			return fmt.Errorf("failed to create user: %w", err)
		}
	}
	return nil
}
