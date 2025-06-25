package repository

import (
	"context"
	"golang-swing-trading-signal/internal/models"
	"strings"

	"gorm.io/gorm"
)

type StockPositionMonitoringRepository interface {
	GetLatestMonitoring(ctx context.Context, param models.GetStockPositionMonitoringParam) ([]models.StockPositionMonitoringEntity, error)
}

type stockPositionMonitoringRepository struct {
	db *gorm.DB
}

func NewStockPositionMonitoringRepository(db *gorm.DB) StockPositionMonitoringRepository {
	return &stockPositionMonitoringRepository{db: db}
}

func (s *stockPositionMonitoringRepository) GetLatestMonitoring(ctx context.Context, param models.GetStockPositionMonitoringParam) ([]models.StockPositionMonitoringEntity, error) {
	baseQuery := `SELECT DISTINCT ON (sp.stock_code) spm.* FROM stock_position_monitorings spm 
					LEFT JOIN stock_positions sp ON sp.id =spm.stock_position_id
					LEFT JOIN users u ON u.id =sp.user_id				
	`

	orderQuery := "ORDER BY sp.stock_code, spm.created_at DESC"
	qFilter := []string{}
	params := []interface{}{}
	defaultLimit := 10

	if param.TelegramID > 0 {
		qFilter = append(qFilter, "u.telegram_id = ?")
		params = append(params, param.TelegramID)
	}

	if param.StockPositionID > 0 {
		qFilter = append(qFilter, "spm.stock_position_id = ?")
		params = append(params, param.StockPositionID)
	}

	if param.IsActive {
		qFilter = append(qFilter, "sp.is_active = ?")
		params = append(params, true)
	}
	if param.StockCode != "" {
		qFilter = append(qFilter, "sp.stock_code = ?")
		params = append(params, param.StockCode)
	}

	if len(qFilter) > 0 {
		baseQuery += " WHERE " + strings.Join(qFilter, " AND ")
	}
	if param.Limit > 0 {
		defaultLimit = param.Limit
	}
	params = append(params, defaultLimit)
	baseQuery += " " + orderQuery + " LIMIT ?"

	var stocks []models.StockPositionMonitoringEntity

	err := s.db.WithContext(ctx).Debug().Raw(baseQuery, params...).Scan(&stocks).Error
	return stocks, err
}
