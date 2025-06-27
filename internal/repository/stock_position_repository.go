package repository

import (
	"context"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"

	"gorm.io/gorm"
)

type StockPositionRepository interface {
	Create(ctx context.Context, stockPosition *models.StockPositionEntity, opts ...utils.DBOption) error
	Update(ctx context.Context, stockPosition *models.StockPositionEntity, opts ...utils.DBOption) error
	Delete(ctx context.Context, stockPosition *models.StockPositionEntity, opts ...utils.DBOption) error
	GetList(ctx context.Context, queryParam models.StockPositionQueryParam, opts ...utils.DBOption) ([]models.StockPositionEntity, error)
}

type stockPositionRepository struct {
	db *gorm.DB
}

func NewStockPositionRepository(db *gorm.DB) StockPositionRepository {
	return &stockPositionRepository{
		db: db,
	}
}

func (r *stockPositionRepository) Create(ctx context.Context, stockPosition *models.StockPositionEntity, opts ...utils.DBOption) error {
	tx := utils.ApplyOptions(r.db.WithContext(ctx), opts...)
	return tx.Create(stockPosition).Error
}

func (r *stockPositionRepository) Update(ctx context.Context, stockPosition *models.StockPositionEntity, opts ...utils.DBOption) error {
	tx := utils.ApplyOptions(r.db.WithContext(ctx), opts...)
	return tx.Updates(stockPosition).Error
}

func (r *stockPositionRepository) Delete(ctx context.Context, stockPosition *models.StockPositionEntity, opts ...utils.DBOption) error {
	tx := utils.ApplyOptions(r.db.WithContext(ctx), opts...)
	return tx.Delete(stockPosition).Error
}

func (r *stockPositionRepository) GetList(ctx context.Context, queryParam models.StockPositionQueryParam, opts ...utils.DBOption) ([]models.StockPositionEntity, error) {
	var stockPositions []models.StockPositionEntity

	db := utils.ApplyOptions(r.db.WithContext(ctx), opts...)
	db = db.Model(&models.StockPositionEntity{})

	// JOIN ke users jika ada filter TelegramID
	if len(queryParam.TelegramIDs) > 0 {
		db = db.Joins("JOIN users u ON u.id = stock_positions.user_id").
			Where("u.telegram_id IN ?", queryParam.TelegramIDs)
	}

	if len(queryParam.StockCodes) > 0 {
		db = db.Where("stock_positions.stock_code IN ?", queryParam.StockCodes)
	}

	if len(queryParam.IDs) > 0 {
		db = db.Where("stock_positions.id IN ?", queryParam.IDs)
	}

	if queryParam.IsActive {
		db = db.Where("stock_positions.is_active = ?", true)
	}

	if queryParam.IsExit != nil && *queryParam.IsExit {
		db = db.Where("stock_positions.exit_price is not null")
	}

	if queryParam.Monitoring != nil {
		// Preload monitoring dengan order by
		db = db.Preload("StockPositionMonitorings", func(db *gorm.DB) *gorm.DB {

			if queryParam.Monitoring.ShowNewest != nil && *queryParam.Monitoring.ShowNewest {
				db = db.Order("created_at DESC")
			}

			if queryParam.Monitoring.Limit != nil && *queryParam.Monitoring.Limit > 0 {
				db = db.Limit(*queryParam.Monitoring.Limit)
			}
			return db
		})
	}

	result := db.Find(&stockPositions)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	return stockPositions, nil
}
