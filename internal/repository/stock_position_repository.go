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

	// Preload monitoring dengan order by
	db = db.Preload("StockPositionMonitorings", func(db *gorm.DB) *gorm.DB {
		if queryParam.Monitoring != nil {
			if queryParam.Monitoring.Interval != nil {
				db = db.Where("interval = ?", *queryParam.Monitoring.Interval)
			}
			if queryParam.Monitoring.Range != nil {
				db = db.Where("range = ?", *queryParam.Monitoring.Range)
			}
			if queryParam.Monitoring.Limit != nil {
				db = db.Limit(*queryParam.Monitoring.Limit)
			}
		}
		return db.Order("created_at DESC")
	})

	result := db.Debug().Find(&stockPositions)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	return stockPositions, nil
}
