package trading_analysis

import (
	"context"
	"fmt"

	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/repository"
	"golang-swing-trading-signal/internal/services/gemini_ai"
	"golang-swing-trading-signal/internal/services/yahoo_finance"

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
