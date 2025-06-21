package repository

import (
	"context"
	"golang-swing-trading-signal/internal/models"

	"gorm.io/gorm"
)

type StocksRepository interface {
	GetStocks(ctx context.Context) ([]models.StockEntity, error)
}

type stocksRepository struct {
	db *gorm.DB
}

func NewStocksRepository(db *gorm.DB) StocksRepository {
	return &stocksRepository{db: db}
}

func (s *stocksRepository) GetStocks(ctx context.Context) ([]models.StockEntity, error) {
	var stocks []models.StockEntity
	if err := s.db.WithContext(ctx).Find(&stocks).Error; err != nil {
		return nil, err
	}
	return stocks, nil
}
