package stocks

import (
	"context"
	"fmt"
	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/repository"
	"golang-swing-trading-signal/internal/utils"

	"github.com/sirupsen/logrus"
)

type StockService interface {
	GetStocks(ctx context.Context) ([]models.StockEntity, error)
	SetStockPosition(ctx context.Context, request *models.RequestSetPositionData) error
	UpdateStockPositionTelegramUser(ctx context.Context, telegramID int64, stockPositionID uint, update *models.StockPositionUpdateRequest) error
	DeleteStockPositionTelegramUser(ctx context.Context, telegramID int64, stockPositionID uint) error
	GetStockPositionsTelegramUser(ctx context.Context, telegramID int64) ([]models.StockPositionEntity, error)
	GetStockPosition(ctx context.Context, param models.StockPositionQueryParam) ([]models.StockPositionEntity, error)
}

type stockService struct {
	cfg                        *config.Config
	logger                     *logrus.Logger
	stocksRepository           repository.StocksRepository
	stockNewsSummaryRepository repository.StockNewsSummaryRepository
	stockPositionRepository    repository.StockPositionRepository
	userRepository             repository.UserRepository
	unitOfWork                 repository.UnitOfWork
}

func NewStockService(cfg *config.Config, stocksRepository repository.StocksRepository, stockNewsSummaryRepository repository.StockNewsSummaryRepository, stockPositionRepository repository.StockPositionRepository, userRepository repository.UserRepository, logger *logrus.Logger, unitOfWork repository.UnitOfWork) StockService {
	return &stockService{
		cfg:                        cfg,
		stocksRepository:           stocksRepository,
		stockNewsSummaryRepository: stockNewsSummaryRepository,
		stockPositionRepository:    stockPositionRepository,
		userRepository:             userRepository,
		logger:                     logger,
		unitOfWork:                 unitOfWork,
	}
}

func (s *stockService) GetStocks(ctx context.Context) ([]models.StockEntity, error) {
	return s.stocksRepository.GetStocks(ctx)
}

func (s *stockService) GetStockPosition(ctx context.Context, param models.StockPositionQueryParam) ([]models.StockPositionEntity, error) {
	positions, err := s.stockPositionRepository.GetList(ctx, param)
	if err != nil {
		s.logger.Error("failed to get stock positions", logrus.Fields{
			"error": err,
		})
		return nil, fmt.Errorf("failed to get stock positions: %w", err)
	}

	if len(positions) == 0 {
		s.logger.Warn("position not found", logrus.Fields{
			"telegram_id": param.TelegramIDs,
			"id":          param.IDs,
		})
		return nil, fmt.Errorf("position not found")
	}

	return positions, nil
}

func (s *stockService) DeleteStockPositionTelegramUser(ctx context.Context, telegramID int64, stockPositionID uint) error {
	positions, err := s.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{telegramID},
		IDs:         []uint{stockPositionID},
	})
	if err != nil {
		return fmt.Errorf("failed to get stock positions: %w", err)
	}

	if len(positions) == 0 {
		return fmt.Errorf("position not found")
	}

	return s.stockPositionRepository.Delete(ctx, &positions[0])
}

func (s *stockService) UpdateStockPositionTelegramUser(ctx context.Context, telegramID int64, stockPositionID uint, update *models.StockPositionUpdateRequest) error {
	positions, err := s.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
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

	return s.stockPositionRepository.Update(ctx, &newUpdate)
}

// SetPosition
func (s *stockService) SetStockPosition(ctx context.Context, request *models.RequestSetPositionData) error {
	user, err := s.userRepository.GetUserByTelegramID(ctx, request.UserTelegram.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	positions, err := s.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{request.UserTelegram.ID},
		IsActive:    true,
		StockCodes:  []string{request.Symbol},
	})

	if err != nil {
		s.logger.WithError(err).Error("failed to get positions", logrus.Fields{
			"telegram_id": request.UserTelegram.ID,
			"symbol":      request.Symbol,
		})
		return fmt.Errorf("failed to get positions: %w", err)
	}

	if len(positions) > 0 {
		s.logger.Warn("position already exists", logrus.Fields{
			"telegram_id": request.UserTelegram.ID,
			"symbol":      request.Symbol,
		})
		return fmt.Errorf("position already exists")
	}

	err = s.unitOfWork.Run(func(opts ...utils.DBOption) error {
		if user == nil {
			user = request.UserTelegram.ToUserEntity()
			s.logger.Infof("User %d not found, creating new user", request.UserTelegram.ID)
			if errInner := s.userRepository.CreateUser(ctx, user, opts...); errInner != nil {
				return errInner
			}
		}

		stockPosition := request.ToStockPositionEntity()
		stockPosition.UserID = user.ID
		stockPosition.IsActive = utils.ToPointer(true)
		return s.stockPositionRepository.Create(ctx, stockPosition, opts...)
	})

	if err != nil {
		s.logger.Error("failed to set position", logrus.Fields{
			"error": err,
		})
		return fmt.Errorf("failed to set position: %w", err)
	}
	return nil
}

func (s *stockService) GetStockPositionsTelegramUser(ctx context.Context, telegramID int64) ([]models.StockPositionEntity, error) {

	position, err := s.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{telegramID},
		IsActive:    true,
	})

	if err != nil {
		s.logger.Error("failed to get stock positions", logrus.Fields{
			"error": err,
		})
		return nil, fmt.Errorf("failed to get stock positions: %w", err)
	}
	return position, nil
}
