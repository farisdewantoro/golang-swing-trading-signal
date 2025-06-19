package repository

import (
	"context"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strings"

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

	baseQuery := `
		SELECT sp.*
		FROM stock_positions sp
	`

	conditions := []string{}
	params := []interface{}{}

	// Jika filter TelegramIDs, maka JOIN ke tabel users
	if len(queryParam.TelegramIDs) > 0 {
		baseQuery += " JOIN users u ON u.id = sp.user_id"
		conditions = append(conditions, "u.telegram_id IN ?")
		params = append(params, queryParam.TelegramIDs)
	}

	if len(queryParam.StockCodes) > 0 {
		conditions = append(conditions, "sp.stock_code IN ?")
		params = append(params, queryParam.StockCodes)
	}

	if queryParam.IsActive {
		conditions = append(conditions, "sp.is_active = ?")
		params = append(params, true)
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	db := utils.ApplyOptions(r.db.WithContext(ctx), opts...)
	result := db.Debug().Raw(baseQuery, params...).Scan(&stockPositions)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}

	return stockPositions, nil
}
