package repository

import (
	"context"
	"golang-swing-trading-signal/internal/models"
	"golang-swing-trading-signal/internal/utils"
	"strings"

	"gorm.io/gorm"
)

type StockPositionMonitoringRepository interface {
	GetLatestMonitoring(ctx context.Context, param models.GetStockPositionMonitoringParam) ([]models.StockPositionMonitoringEntity, error)
	GetRecentDistinctMonitorings(ctx context.Context, param models.StockPositionMonitoringQueryParam, opts ...utils.DBOption) ([]models.StockPositionMonitoringEntity, error)
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
	if !param.AfterTime.IsZero() {
		qFilter = append(qFilter, "spm.created_at >= ?")
		params = append(params, param.AfterTime)
	}

	baseQuery += " WHERE spm.deleted_at IS NULL"

	if len(qFilter) > 0 {
		baseQuery += " AND " + strings.Join(qFilter, " AND ")
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

func (r *stockPositionMonitoringRepository) GetRecentDistinctMonitorings(ctx context.Context, param models.StockPositionMonitoringQueryParam, opts ...utils.DBOption) ([]models.StockPositionMonitoringEntity, error) {
	var results []models.StockPositionMonitoringEntity

	query := `
	WITH ranked AS (
	  SELECT
	    id,
	    stock_position_id,
	    signal,
	    confidence_score,
	    data,
	    created_at,
	    LAG(signal) OVER (PARTITION BY stock_position_id ORDER BY created_at DESC) AS prev_signal,
	    LAG(data->>'market_price') OVER (PARTITION BY stock_position_id ORDER BY created_at DESC) AS prev_price,
	    data->>'market_price' AS price
	  FROM stock_position_monitorings
	  WHERE stock_position_id = ? AND deleted_at IS NULL
	),
	filtered AS (
	  SELECT *
	  FROM ranked
	  WHERE
	    prev_signal IS NULL OR
	    signal IS DISTINCT FROM prev_signal OR
	    price IS DISTINCT FROM prev_price
	)
	SELECT *
	FROM filtered
	ORDER BY created_at DESC
	LIMIT ?
	`

	err := utils.ApplyOptions(r.db.WithContext(ctx), opts...).Raw(query, param.StockPositionID, param.Limit).Scan(&results).Error
	return results, err
}
