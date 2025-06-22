package repository

import (
	"context"
	"golang-swing-trading-signal/internal/models"
	"time"

	"gorm.io/gorm"
)

type StockNewsSummaryRepository interface {
	GetLast(ctx context.Context, before time.Time, stockCode string) (*models.StockNewsSummaryEntity, error)
}

type stockNewsSummaryRepository struct {
	db *gorm.DB
}

func NewStockNewsSummaryRepository(db *gorm.DB) StockNewsSummaryRepository {
	return &stockNewsSummaryRepository{
		db: db,
	}
}

func (r *stockNewsSummaryRepository) GetLast(ctx context.Context, before time.Time, stockCode string) (*models.StockNewsSummaryEntity, error) {
	var summary models.StockNewsSummaryEntity
	result := r.db.WithContext(ctx).Where("created_at >= ? AND stock_code = ?", before, stockCode).Order("created_at desc").First(&summary)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, result.Error
	}
	return &summary, nil
}
