package trading_analysis

import (
	"context"
	"fmt"
	"log"
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
	lastStockNewsSummary, err := a.stockNewsSummaryRepository.GetLast(beforeTime, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get last stock news summary: %w", err)
	}

	// Send to Gemini AI for analysis
	analysis, err := a.geminiClient.AnalyzeStock(ctx, symbol, ohlcvDataWithInfo.Data, ohlcvDataWithInfo.DataInfo, lastStockNewsSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze stock: %w", err)
	}

	// Set data info from Yahoo Finance response (even if not in Gemini response)
	analysis.DataInfo = ohlcvDataWithInfo.DataInfo

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
	lastStockNewsSummary, err := a.stockNewsSummaryRepository.GetLast(beforeTime, request.Symbol)
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

// AnalyzeAllStocks analyzes all stocks from configuration and returns summary
func (a *Analyzer) AnalyzeAllStocks(ctx context.Context, stockList []string, interval, period string) (*models.SummaryAnalysisResponse, error) {
	var buyList []models.StockSummary
	var holdList []models.StockSummary

	bestOpportunity := ""
	worstOpportunity := ""
	bestProfit := -999.0
	worstProfit := 999.0

	log.Printf("Starting analysis of %d stocks", len(stockList))

	// Analyze each stock
	for i, symbol := range stockList {
		log.Printf("[%d/%d] Analyzing stock: %s", i+1, len(stockList), symbol)

		analysis, err := a.AnalyzeStock(ctx, symbol, interval, period)
		if err != nil {
			// Log error but continue with other stocks
			log.Printf("Error analyzing %s: %v", symbol, err)
			continue
		}

		log.Printf("Successfully analyzed %s - Signal: %s, Confidence: %d", symbol, analysis.Signal, analysis.TechnicalSummary.ConfidenceLevel)

		stockSummary := models.StockSummary{
			Symbol:           symbol,
			Signal:           analysis.Signal,
			CurrentPrice:     analysis.DataInfo.MarketPrice,
			BuyPrice:         analysis.Recommendation.BuyPrice,
			TargetPrice:      analysis.Recommendation.TargetPrice,
			StopLoss:         analysis.Recommendation.CutLoss,
			MaxHoldingDays:   analysis.MaxHoldingPeriodDays,
			ProfitPercentage: analysis.Recommendation.RiskRewardAnalysis.PotentialProfit,
			RiskRewardRatio:  analysis.Recommendation.RiskRewardAnalysis.RiskRewardRatio,
			Confidence:       analysis.TechnicalSummary.ConfidenceLevel,
			RiskLevel:        analysis.RiskLevel,
		}

		// Categorize by signal
		if analysis.Signal == "BUY" {
			buyList = append(buyList, stockSummary)
		} else if analysis.Signal == "HOLD" {
			holdList = append(holdList, stockSummary)
		}

		// Track best and worst opportunities
		if analysis.Recommendation.RiskRewardAnalysis.PotentialProfit > bestProfit {
			bestProfit = analysis.Recommendation.RiskRewardAnalysis.PotentialProfit
			bestOpportunity = symbol
		}
		if analysis.Recommendation.RiskRewardAnalysis.PotentialLoss > worstProfit {
			worstProfit = analysis.Recommendation.RiskRewardAnalysis.PotentialLoss
			worstOpportunity = symbol
		}

	}

	log.Printf("Analysis completed. Total analyzed: %d, Buy signals: %d, Hold signals: %d", len(buyList)+len(holdList), len(buyList), len(holdList))

	totalStocks := len(buyList) + len(holdList)
	if totalStocks == 0 {
		return nil, fmt.Errorf("no stocks could be analyzed successfully")
	}

	summary := models.SummaryStatistics{
		BestOpportunity:  bestOpportunity,
		WorstOpportunity: worstOpportunity,
	}

	return &models.SummaryAnalysisResponse{
		AnalysisDate: time.Now(),
		BuyList:      buyList,
		HoldList:     holdList,
		TotalStocks:  totalStocks,
		BuyCount:     len(buyList),
		HoldCount:    len(holdList),
		Summary:      summary,
	}, nil
}

// SetPosition
func (a *Analyzer) SetStockPosition(ctx context.Context, request *models.RequestSetPositionData) error {
	user, err := a.userRepository.GetUserByTelegramID(ctx, request.UserTelegram.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	positions, err := a.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{request.UserTelegram.ID},
		IsActive:    true,
		StockCodes:  []string{request.Symbol},
	})

	if err != nil {
		a.logger.WithError(err).Error("failed to get positions", logrus.Fields{
			"telegram_id": request.UserTelegram.ID,
			"symbol":      request.Symbol,
		})
		return fmt.Errorf("failed to get positions: %w", err)
	}

	if len(positions) > 0 {
		a.logger.Warn("position already exists", logrus.Fields{
			"telegram_id": request.UserTelegram.ID,
			"symbol":      request.Symbol,
		})
		return fmt.Errorf("position already exists")
	}

	err = a.unitOfWork.Run(func(opts ...utils.DBOption) error {
		if user == nil {
			user = request.UserTelegram.ToUserEntity()
			a.logger.Infof("User %d not found, creating new user", request.UserTelegram.ID)
			if errInner := a.userRepository.CreateUser(ctx, user, opts...); errInner != nil {
				return errInner
			}
		}

		stockPosition := request.ToStockPositionEntity()
		stockPosition.UserID = user.ID
		stockPosition.IsActive = utils.ToPointer(true)
		return a.stockPositionRepository.Create(ctx, stockPosition, opts...)
	})

	if err != nil {
		a.logger.Error("failed to set position", logrus.Fields{
			"error": err,
		})
		return fmt.Errorf("failed to set position: %w", err)
	}
	return nil
}

func (a *Analyzer) GetStockPositionsTelegramUser(ctx context.Context, telegramID int64) ([]models.StockPositionEntity, error) {

	position, err := a.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{telegramID},
		IsActive:    true,
	})

	if err != nil {
		a.logger.Error("failed to get stock positions", logrus.Fields{
			"error": err,
		})
		return nil, fmt.Errorf("failed to get stock positions: %w", err)
	}
	return position, nil
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

func (a *Analyzer) GetStockPosition(ctx context.Context, param models.StockPositionQueryParam) ([]models.StockPositionEntity, error) {
	positions, err := a.stockPositionRepository.GetList(ctx, param)
	if err != nil {
		a.logger.Error("failed to get stock positions", logrus.Fields{
			"error": err,
		})
		return nil, fmt.Errorf("failed to get stock positions: %w", err)
	}

	if len(positions) == 0 {
		a.logger.Warn("position not found", logrus.Fields{
			"telegram_id": param.TelegramIDs,
			"id":          param.IDs,
		})
		return nil, fmt.Errorf("position not found")
	}

	return positions, nil
}

func (a *Analyzer) DeleteStockPositionTelegramUser(ctx context.Context, telegramID int64, stockPositionID uint) error {
	positions, err := a.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{telegramID},
		IDs:         []uint{stockPositionID},
	})
	if err != nil {
		return fmt.Errorf("failed to get stock positions: %w", err)
	}

	if len(positions) == 0 {
		return fmt.Errorf("position not found")
	}

	return a.stockPositionRepository.Delete(ctx, &positions[0])
}

func (a *Analyzer) UpdateStockPositionTelegramUser(ctx context.Context, telegramID int64, stockPositionID uint, update *models.StockPositionUpdateRequest) error {
	positions, err := a.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{telegramID},
		IDs:         []uint{stockPositionID},
	})
	if err != nil {
		return fmt.Errorf("failed to get stock positions: %w", err)
	}

	if len(positions) == 0 {
		return fmt.Errorf("position not found")
	}

	newUpdate := positions[0]

	if update.BuyPrice != nil {
		newUpdate.BuyPrice = *update.BuyPrice
	}

	if update.BuyDate != nil {
		newUpdate.BuyDate = *update.BuyDate
	}

	if update.MaxHoldingPeriodDays != nil {
		newUpdate.MaxHoldingPeriodDays = *update.MaxHoldingPeriodDays
	}

	if update.PriceAlert != nil {
		newUpdate.PriceAlert = update.PriceAlert
	}

	if update.MonitorPosition != nil {
		newUpdate.MonitorPosition = update.MonitorPosition
	}

	if update.ExitPrice != nil {
		newUpdate.ExitPrice = update.ExitPrice
	}

	if update.ExitDate != nil {
		newUpdate.ExitDate = update.ExitDate
	}

	if update.IsActive != nil {
		newUpdate.IsActive = update.IsActive
	}

	return a.stockPositionRepository.Update(ctx, &newUpdate)
}
