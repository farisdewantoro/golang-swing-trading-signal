package trading_analysis

import (
	"context"
	"fmt"
	"log"
	"math"
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
}

func NewAnalyzer(yahooClient *yahoo_finance.Client, geminiClient *gemini_ai.Client, logger *logrus.Logger, stockNewsSummaryRepository repository.StockNewsSummaryRepository) *Analyzer {
	return &Analyzer{
		yahooClient:                yahooClient,
		geminiClient:               geminiClient,
		logger:                     logger,
		stockNewsSummaryRepository: stockNewsSummaryRepository,
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
func (a *Analyzer) AnalyzeAllStocks(ctx context.Context, stockList []string) (*models.SummaryAnalysisResponse, error) {
	var buyList []models.StockSummary
	var holdList []models.StockSummary
	var totalConfidence int
	var totalProfitPercent float64
	var totalRiskRatio float64

	bestOpportunity := ""
	worstOpportunity := ""
	bestProfit := -999.0
	worstProfit := 999.0

	log.Printf("Starting analysis of %d stocks", len(stockList))

	// Analyze each stock
	for i, symbol := range stockList {
		log.Printf("[%d/%d] Analyzing stock: %s", i+1, len(stockList), symbol)

		analysis, err := a.AnalyzeStock(ctx, symbol, "", "")
		if err != nil {
			// Log error but continue with other stocks
			log.Printf("Error analyzing %s: %v", symbol, err)
			continue
		}

		log.Printf("Successfully analyzed %s - Signal: %s, Confidence: %d", symbol, analysis.Signal, analysis.TechnicalSummary.ConfidenceLevel)

		// Get current price from Yahoo Finance data directly
		var currentPrice float64
		// Get the latest OHLC data to extract current price
		ohlcvDataWithInfo, err := a.yahooClient.GetRecentOHLCData(symbol, "", "")
		if err == nil && len(ohlcvDataWithInfo.Data) > 0 {
			// Use the most recent close price as current price
			currentPrice = ohlcvDataWithInfo.Data[len(ohlcvDataWithInfo.Data)-1].Close
		}

		// Calculate profit percentage and risk ratio
		profitPercentage := ((analysis.Recommendation.TargetPrice - analysis.Recommendation.BuyPrice) / analysis.Recommendation.BuyPrice) * 100
		riskRatio := profitPercentage / math.Abs(((analysis.Recommendation.CutLoss-analysis.Recommendation.BuyPrice)/analysis.Recommendation.BuyPrice)*100)

		stockSummary := models.StockSummary{
			Symbol:           symbol,
			Signal:           analysis.Signal,
			CurrentPrice:     currentPrice,
			BuyPrice:         analysis.Recommendation.BuyPrice,
			TargetPrice:      analysis.Recommendation.TargetPrice,
			StopLoss:         analysis.Recommendation.CutLoss,
			MaxHoldingDays:   analysis.MaxHoldingPeriodDays,
			ProfitPercentage: profitPercentage,
			RiskRatio:        riskRatio,
			Confidence:       analysis.TechnicalSummary.ConfidenceLevel,
			RiskLevel:        analysis.RiskLevel,
		}

		// Categorize by signal
		if analysis.Signal == "BUY" {
			buyList = append(buyList, stockSummary)
		} else if analysis.Signal == "HOLD" {
			holdList = append(holdList, stockSummary)
		}

		// Update statistics
		totalConfidence += analysis.TechnicalSummary.ConfidenceLevel
		totalProfitPercent += profitPercentage
		totalRiskRatio += riskRatio

		// Track best and worst opportunities
		if profitPercentage > bestProfit {
			bestProfit = profitPercentage
			bestOpportunity = symbol
		}
		if profitPercentage < worstProfit {
			worstProfit = profitPercentage
			worstOpportunity = symbol
		}

		// Sleep for 2 seconds before processing next stock (except for the last one)
		if i < len(stockList)-1 {
			log.Printf("Waiting 2 seconds before processing next stock...")
			time.Sleep(2 * time.Second)
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
