package models

import (
	"time"
)

// OHLCV Data Structure
type OHLCVData struct {
	Timestamp int64   `json:"timestamp"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    int64   `json:"volume"`
}

// Yahoo Finance API Response
type YahooFinanceResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol             string  `json:"symbol"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

// OHLCV Analysis
type OHLCVAnalysis struct {
	Open        float64 `json:"open"`
	High        float64 `json:"high"`
	Low         float64 `json:"low"`
	Close       float64 `json:"close"`
	Volume      int64   `json:"volume"`
	Explanation string  `json:"explanation"`
}

type TechnicalAnalysis struct {
	Trend                  string   `json:"trend"`
	Momentum               string   `json:"momentum"`
	EMASignal              string   `json:"ema_signal"`
	RSISignal              string   `json:"rsi_signal"`
	MACDSignal             string   `json:"macd_signal"`
	BollingerBandsPosition string   `json:"bollinger_bands_position"`
	SupportLevel           float64  `json:"support_level"`
	ResistanceLevel        float64  `json:"resistance_level"`
	TechnicalScore         int      `json:"technical_score"`
	KeyInsights            []string `json:"key_insights"`
}

// Risk Reward Analysis
type RiskRewardAnalysis struct {
	PotentialProfit           float64 `json:"potential_profit"`
	PotentialProfitPercentage float64 `json:"potential_profit_percentage"`
	PotentialLoss             float64 `json:"potential_loss"`
	PotentialLossPercentage   float64 `json:"potential_loss_percentage"`
	RiskRewardRatio           float64 `json:"risk_reward_ratio"`
	RiskLevel                 string  `json:"risk_level"`
	ExpectedHoldingPeriod     string  `json:"expected_holding_period"`
	SuccessProbability        int     `json:"success_probability"`
	TrendProbability          int     `json:"trend_probability"`
	VolumeProbability         int     `json:"volume_probability"`
	TechnicalProbability      int     `json:"technical_probability"`
}

// Position Risk Reward Analysis
type PositionRiskRewardAnalysis struct {
	CurrentProfit                      float64            `json:"current_profit"`
	CurrentProfitPercentage            float64            `json:"current_profit_percentage"`
	RemainingPotentialProfit           float64            `json:"remaining_potential_profit"`
	RemainingPotentialProfitPercentage float64            `json:"remaining_potential_profit_percentage"`
	CurrentRisk                        float64            `json:"current_risk"`
	CurrentRiskPercentage              float64            `json:"current_risk_percentage"`
	RiskRewardRatio                    float64            `json:"risk_reward_ratio"`
	RiskLevel                          string             `json:"risk_level"`
	DaysRemaining                      int                `json:"days_remaining"`
	SuccessProbability                 int                `json:"success_probability"`
	TrendProbability                   int                `json:"trend_probability"`
	VolumeProbability                  int                `json:"volume_probability"`
	TechnicalProbability               int                `json:"technical_probability"`
	ExitRecommendation                 ExitRecommendation `json:"exit_recommendation"`
}

// Exit Recommendation
type ExitRecommendation struct {
	TargetExitPrice float64  `json:"target_exit_price"`
	StopLossPrice   float64  `json:"stop_loss_price"`
	TimeBasedExit   string   `json:"time_based_exit"`
	ExitReasoning   string   `json:"exit_reasoning"`
	ExitConditions  []string `json:"exit_conditions"`
}

// Recommendation
type Recommendation struct {
	Action          string  `json:"action"`
	BuyPrice        float64 `json:"buy_price,omitempty"`
	TargetPrice     float64 `json:"target_price,omitempty"`
	CutLoss         float64 `json:"cut_loss,omitempty"`
	ConfidenceLevel int     `json:"confidence_level"`
	Reasoning       string  `json:"reasoning"`
	RiskRewardRatio float64 `json:"risk_reward_ratio"`
}

// Position Recommendation
type PositionRecommendation struct {
	Action             string                     `json:"action"`
	Reasoning          string                     `json:"reasoning"`
	TechnicalAnalysis  TechnicalAnalysis          `json:"technical_analysis"`
	RiskRewardAnalysis PositionRiskRewardAnalysis `json:"risk_reward_analysis"`
}

// Position Metrics
type PositionMetrics struct {
	UnrealizedPnL           float64 `json:"unrealized_pnl"`
	UnrealizedPnLPercentage float64 `json:"unrealized_pnl_percentage"`
	DaysRemaining           int     `json:"days_remaining"`
	RiskAssessment          string  `json:"risk_assessment"`
	PositionHealth          string  `json:"position_health"`
	TrendAlignment          string  `json:"trend_alignment"`
	VolumeSupport           string  `json:"volume_support"`
}

type IndividualAnalysisResponse struct {
	MarketPrice       float64           `json:"market_price"`
	Symbol            string            `json:"symbol"`
	AnalysisDate      time.Time         `json:"analysis_date"`
	TechnicalAnalysis TechnicalAnalysis `json:"technical_analysis"`
	Recommendation    Recommendation    `json:"recommendation"`
	NewsSummary       NewsSummary       `json:"news_summary,omitempty"`
}

// Data Information
type DataInfo struct {
	Interval    string    `json:"interval"`
	Range       string    `json:"range"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	DataPoints  int       `json:"data_points"`
	Source      string    `json:"source"`
	MarketPrice float64   `json:"market_price"`
}

// Position Monitoring Request
type PositionMonitoringRequest struct {
	Symbol               string    `json:"symbol" binding:"required"`
	BuyPrice             float64   `json:"buy_price" binding:"required"`
	BuyTime              time.Time `json:"buy_time" binding:"required"`
	MaxHoldingPeriodDays int       `json:"max_holding_period_days" binding:"required"`
	Interval             string    `json:"interval"`
	Period               string    `json:"period"`
}

// Technical Summary
type TechnicalSummary struct {
	OverallSignal   string   `json:"overall_signal"`
	TrendStrength   string   `json:"trend_strength"`
	VolumeSupport   string   `json:"volume_support"`
	Momentum        string   `json:"momentum"`
	RiskLevel       string   `json:"risk_level"`
	ConfidenceLevel int      `json:"confidence_level"`
	KeyInsights     []string `json:"key_insights"`
}

// News Summary
type NewsSummary struct {
	ConfidenceScore float64  `json:"confidence_score"`
	Sentiment       string   `json:"sentiment"`
	Impact          string   `json:"impact"`
	KeyIssues       []string `json:"key_issues"`
	Reasoning       string   `json:"reasoning"`
}

// Position Monitoring Response
type PositionMonitoringResponse struct {
	AnalysisDate         time.Time              `json:"analysis_date"`
	TargetPrice          float64                `json:"target_price"`
	CutLoss              float64                `json:"cut_loss"`
	MarketPrice          float64                `json:"market_price"`
	BuyDate              time.Time              `json:"buy_date"`
	BuyPrice             float64                `json:"buy_price"`
	MaxHoldingPeriodDays int                    `json:"max_holding_period_days"`
	Symbol               string                 `json:"symbol"`
	TechnicalAnalysis    TechnicalAnalysis      `json:"technical_analysis"`
	NewsSummary          NewsSummary            `json:"news_summary,omitempty"`
	Recommendation       RecommendationPosition `json:"recommendation,omitempty"`
}

// Gemini AI Request
type GeminiAIRequest struct {
	GenerationConfig *GeminiGenerationConfig `json:"generationConfig"`
	Contents         []GeminiContent         `json:"contents"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
}

type GeminiGenerationConfig struct {
	Temperature float64 `json:"temperature"`
}

type GeminiPart struct {
	Text string `json:"text"`
}

// Gemini AI Response
type GeminiAIResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
}

// Error Response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Summary Analysis Response
type SummaryAnalysisResponse struct {
	AnalysisDate time.Time         `json:"analysis_date"`
	BuyList      []StockSummary    `json:"buy_list"`
	HoldList     []StockSummary    `json:"hold_list"`
	TotalStocks  int               `json:"total_stocks"`
	BuyCount     int               `json:"buy_count"`
	HoldCount    int               `json:"hold_count"`
	Summary      SummaryStatistics `json:"summary"`
}

// Stock Summary for Buy/Hold List
type StockSummary struct {
	Symbol           string  `json:"symbol"`
	Signal           string  `json:"signal"`
	CurrentPrice     float64 `json:"current_price"`
	BuyPrice         float64 `json:"buy_price"`
	TargetPrice      float64 `json:"target_price"`
	StopLoss         float64 `json:"stop_loss"`
	MaxHoldingDays   int     `json:"max_holding_days"`
	ProfitPercentage float64 `json:"profit_percentage"`
	RiskRewardRatio  float64 `json:"risk_reward_ratio"`
	Confidence       int     `json:"confidence"`
	RiskLevel        string  `json:"risk_level"`
}

// Summary Statistics
type SummaryStatistics struct {
	BestOpportunity  string `json:"best_opportunity"`
	WorstOpportunity string `json:"worst_opportunity"`
}

type PositionMonitoringTelegramUserRequest struct {
	TelegramID int64  `json:"telegram_id" binding:"required"`
	Symbol     string `json:"symbol" binding:"required"`
	Interval   string `json:"interval"`
	Period     string `json:"period"`
}

type RecommendationPosition struct {
	Action          string   `json:"action"`
	TargetPrice     float64  `json:"target_price,omitempty"`
	CutLoss         float64  `json:"cut_loss,omitempty"`
	ConfidenceLevel int      `json:"confidence_level"`
	ExitReasoning   string   `json:"exit_reasoning"`
	ExitConditions  []string `json:"exit_conditions"`
	RiskRewardRatio float64  `json:"risk_reward_ratio"`
}
