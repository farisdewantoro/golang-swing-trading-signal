package repository

import (
	"context"
	"golang-swing-trading-signal/internal/models"
	"strings"

	"gorm.io/gorm"
)

type StockSignalRepository interface {
	GetLatestSignal(ctx context.Context, param models.GetStockBuySignalParam) ([]models.StockSignalEntity, error)
}

type stockSignalRepository struct {
	db *gorm.DB
}

func NewStockSignalRepository(db *gorm.DB) StockSignalRepository {
	return &stockSignalRepository{db: db}
}

func (s *stockSignalRepository) GetLatestSignal(ctx context.Context, param models.GetStockBuySignalParam) ([]models.StockSignalEntity, error) {
	var stockSignals []models.StockSignalEntity
	basedQuery := `SELECT DISTINCT ON (ss.stock_code) * FROM stock_signals as ss`
	orderQuery := "ORDER BY ss.stock_code, ss.created_at DESC"
	filterQuery := []string{}
	filterParams := []interface{}{}

	if param.Signal != "" {
		filterQuery = append(filterQuery, "ss.signal = ?")
		filterParams = append(filterParams, param.Signal)
	}

	if !param.After.IsZero() {
		filterQuery = append(filterQuery, "ss.created_at >= ?")
		filterParams = append(filterParams, param.After)
	}
	if param.StockCode != "" {
		filterQuery = append(filterQuery, "ss.stock_code = ?")
		filterParams = append(filterParams, param.StockCode)
	}

	basedQuery += " WHERE ss.deleted_at IS NULL"

	if len(filterQuery) > 0 {
		basedQuery += " AND " + strings.Join(filterQuery, " AND ")
	}

	if err := s.db.WithContext(ctx).Debug().Raw(basedQuery+" "+orderQuery, filterParams...).Scan(&stockSignals).Error; err != nil {
		return nil, err
	}
	return stockSignals, nil
}
