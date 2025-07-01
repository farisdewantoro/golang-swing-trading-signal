package stocks

import (
	"context"
	"encoding/json"
	"fmt"
	"golang-swing-trading-signal/internal/config"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/repository"
	"golang-swing-trading-signal/internal/utils"
	"golang-swing-trading-signal/pkg/redis"

	goRedis "github.com/redis/go-redis/v9"

	"github.com/sirupsen/logrus"
)

type StockService interface {
	GetStocks(ctx context.Context) ([]models.StockEntity, error)
	SetStockPosition(ctx context.Context, request *models.RequestSetPositionData) error
	UpdateStockPositionTelegramUser(ctx context.Context, telegramID int64, stockPositionID uint, update *models.StockPositionUpdateRequest) error
	DeleteStockPositionTelegramUser(ctx context.Context, telegramID int64, stockPositionID uint) error
	GetStockPositionsTelegramUser(ctx context.Context, telegramID int64, monitoring *models.StockPositionMonitoringQueryParam) ([]models.StockPositionEntity, error)
	GetStockPosition(ctx context.Context, param models.StockPositionQueryParam) ([]models.StockPositionEntity, error)
	GetByParam(ctx context.Context, param models.GetStocksParam) ([]models.StockEntity, error)
	GetTopNews(ctx context.Context, param models.StockNewsQueryParam) ([]models.StockNewsEntity, error)
	GetLastStockNewsSummary(ctx context.Context, age int, stockCode string) (*models.StockNewsSummaryEntity, error)
	GetLatestStockSignal(ctx context.Context, param models.GetStockBuySignalParam) ([]models.StockSignalEntity, error)
	GetLatestStockPositionMonitoring(ctx context.Context, param models.GetStockPositionMonitoringParam) ([]models.StockPositionMonitoringEntity, error)
	RequestStockPositionMonitoring(ctx context.Context, param *models.RequestStockPositionMonitoring) error
	RequestStockAnalyzer(ctx context.Context, param *models.RequestStockAnalyzer) error
	GetTopNewsGlobal(ctx context.Context, limit int, age int) ([]models.TopNewsCustomResult, error)
	GetStockPositionWithHistoryMonitoring(ctx context.Context, param models.StockPositionQueryParam) (*models.StockPositionEntity, error)
}

type stockService struct {
	cfg                               *config.Config
	logger                            *logrus.Logger
	stocksRepository                  repository.StocksRepository
	stockNewsSummaryRepository        repository.StockNewsSummaryRepository
	stockPositionRepository           repository.StockPositionRepository
	userRepository                    repository.UserRepository
	unitOfWork                        repository.UnitOfWork
	stockNewsRepository               repository.StocksNewsRepository
	stockSignalRepository             repository.StockSignalRepository
	stockPositionMonitoringRepository repository.StockPositionMonitoringRepository
	redisClient                       *redis.Client
}

func NewStockService(
	cfg *config.Config,
	stocksRepository repository.StocksRepository,
	stockNewsSummaryRepository repository.StockNewsSummaryRepository,
	stockPositionRepository repository.StockPositionRepository,
	userRepository repository.UserRepository,
	logger *logrus.Logger,
	unitOfWork repository.UnitOfWork,
	stockNewsRepository repository.StocksNewsRepository,
	stockSignalRepository repository.StockSignalRepository,
	stockPositionMonitoringRepository repository.StockPositionMonitoringRepository,
	redisClient *redis.Client,
) StockService {
	return &stockService{
		cfg:                               cfg,
		stocksRepository:                  stocksRepository,
		stockNewsSummaryRepository:        stockNewsSummaryRepository,
		stockPositionRepository:           stockPositionRepository,
		userRepository:                    userRepository,
		logger:                            logger,
		unitOfWork:                        unitOfWork,
		stockNewsRepository:               stockNewsRepository,
		stockSignalRepository:             stockSignalRepository,
		stockPositionMonitoringRepository: stockPositionMonitoringRepository,
		redisClient:                       redisClient,
	}
}

func (s *stockService) GetStocks(ctx context.Context) ([]models.StockEntity, error) {
	return s.stocksRepository.GetStocks(ctx, models.GetStocksParam{})
}

func (s *stockService) GetByParam(ctx context.Context, param models.GetStocksParam) ([]models.StockEntity, error) {
	return s.stocksRepository.GetStocks(ctx, param)
}

func (s *stockService) GetTopNews(ctx context.Context, param models.StockNewsQueryParam) ([]models.StockNewsEntity, error) {
	return s.stockNewsRepository.GetTopNews(ctx, param)
}

func (s *stockService) GetTopNewsGlobal(ctx context.Context, limit int, age int) ([]models.TopNewsCustomResult, error) {
	return s.stockNewsRepository.GetTopNewsGlobal(ctx, limit, age)
}

func (s *stockService) GetLastStockNewsSummary(ctx context.Context, age int, stockCode string) (*models.StockNewsSummaryEntity, error) {
	return s.stockNewsSummaryRepository.GetLast(ctx, utils.TimeNowWIB().AddDate(0, 0, -age), stockCode)
}

func (s *stockService) GetLatestStockSignal(ctx context.Context, param models.GetStockBuySignalParam) ([]models.StockSignalEntity, error) {
	result, err := s.stockSignalRepository.GetLatestSignal(ctx, param)
	if err != nil {
		s.logger.Error("failed to get latest stock signal", logrus.Fields{
			"error": err,
		})
		return nil, fmt.Errorf("failed to get latest stock signal: %w", err)
	}

	if len(result) == 0 && param.ReqAnalyzer != nil {
		param.ReqAnalyzer = &models.RequestStockAnalyzer{
			StockCode:  param.StockCode,
			TelegramID: param.ReqAnalyzer.TelegramID,
			NotifyUser: param.ReqAnalyzer.NotifyUser,
		}
		s.RequestStockAnalyzer(ctx, param.ReqAnalyzer)
	}

	return result, nil
}

func (s *stockService) GetLatestStockPositionMonitoring(ctx context.Context, param models.GetStockPositionMonitoringParam) ([]models.StockPositionMonitoringEntity, error) {
	return s.stockPositionMonitoringRepository.GetLatestMonitoring(ctx, param)
}

func (s *stockService) RequestStockPositionMonitoring(ctx context.Context, param *models.RequestStockPositionMonitoring) error {

	if param.StockPositionID == 0 {
		positions, err := s.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
			TelegramIDs: []int64{param.TelegramID},
			StockCodes:  []string{param.StockCode},
		})
		if err != nil {
			s.logger.Error("failed to get stock position", logrus.Fields{
				"error": err,
			})
			return err
		}
		if len(positions) == 0 {
			s.logger.Warn("stock position not found", logrus.Fields{
				"telegram_id": param.TelegramID,
				"stock_code":  param.StockCode,
			})
			return fmt.Errorf("stock position not found")
		}
		param.StockPositionID = positions[0].ID
	}

	streamDataJSON, err := json.Marshal(param)
	if err != nil {
		s.logger.Error("failed to parse json data request stock position monitoring", logrus.Fields{
			"error": err,
		})
		return err
	}
	if err := s.redisClient.XAdd(ctx, &goRedis.XAddArgs{
		Stream: models.RedisStreamStockPositionMonitor,
		Values: map[string]interface{}{"payload": streamDataJSON},
	}).Err(); err != nil {
		s.logger.Error("failed to send redis stream stock position monitoring", logrus.Fields{
			"error": err,
		})
		return err
	}

	return nil
}

func (s *stockService) RequestStockAnalyzer(ctx context.Context, param *models.RequestStockAnalyzer) error {

	streamDataJSON, err := json.Marshal(param)
	if err != nil {
		s.logger.Error("failed to parse json data request stock analyzer", logrus.Fields{
			"error": err,
		})
		return err
	}
	if err := s.redisClient.XAdd(ctx, &goRedis.XAddArgs{
		Stream: models.RedisStreamStockAnalyzer,
		Values: map[string]interface{}{"payload": streamDataJSON},
	}).Err(); err != nil {
		s.logger.Error("failed to send redis stream stock analyzer", logrus.Fields{
			"error": err,
		})
		return err
	}

	return nil
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

func (s *stockService) GetStockPositionWithHistoryMonitoring(ctx context.Context, param models.StockPositionQueryParam) (*models.StockPositionEntity, error) {
	monitoring := param.Monitoring

	if monitoring == nil {
		return nil, fmt.Errorf("param monitoring not found")
	}

	param.Monitoring = nil
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

	positions[0].StockPositionMonitorings, err = s.stockPositionMonitoringRepository.GetRecentDistinctMonitorings(ctx, models.StockPositionMonitoringQueryParam{
		StockPositionID: positions[0].ID,
		Limit:           monitoring.Limit,
		ShowNewest:      monitoring.ShowNewest,
	})
	if err != nil {
		s.logger.Error("failed to get stock position monitoring", logrus.Fields{
			"error": err,
		})
		return nil, fmt.Errorf("failed to get stock position monitoring: %w", err)
	}

	return &positions[0], nil
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

	if update.TargetPrice != nil {
		newUpdate.TakeProfitPrice = *update.TargetPrice
	}

	if update.StopLossPrice != nil {
		newUpdate.StopLossPrice = *update.StopLossPrice
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

func (s *stockService) GetStockPositionsTelegramUser(ctx context.Context, telegramID int64, monitoring *models.StockPositionMonitoringQueryParam) ([]models.StockPositionEntity, error) {

	position, err := s.stockPositionRepository.GetList(ctx, models.StockPositionQueryParam{
		TelegramIDs: []int64{telegramID},
		IsActive:    true,
		Monitoring:  monitoring,
	})

	if err != nil {
		s.logger.Error("failed to get stock positions", logrus.Fields{
			"error": err,
		})
		return nil, fmt.Errorf("failed to get stock positions: %w", err)
	}
	return position, nil
}
