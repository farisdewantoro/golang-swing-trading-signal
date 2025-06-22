package repository

import (
	"context"
	"golang-swing-trading-signal/internal/models"
	"strings"

	"gorm.io/gorm"
)

type StocksRepository interface {
	GetStocks(ctx context.Context, param models.GetStocksParam) ([]models.StockEntity, error)
}

type stocksRepository struct {
	db *gorm.DB
}

func NewStocksRepository(db *gorm.DB) StocksRepository {
	return &stocksRepository{db: db}
}

func (s *stocksRepository) GetStocks(ctx context.Context, param models.GetStocksParam) ([]models.StockEntity, error) {
	var stocks []models.StockEntity
	baseQuery := "SELECT * FROM stocks"
	conditions := []string{}
	params := []interface{}{}

	if len(param.StockCodes) > 0 {
		conditions = append(conditions, "stock_code IN ?")
		params = append(params, param.StockCodes)
	}

	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}

	if err := s.db.WithContext(ctx).Raw(baseQuery, params...).Scan(&stocks).Error; err != nil {
		return nil, err
	}
	return stocks, nil
}
