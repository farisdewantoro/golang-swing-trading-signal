package repository

import (
	"golang-swing-trading-signal/internal/models"
	"time"

	"gorm.io/gorm"
)

type StockNewsSummaryRepository interface {
	GetLast(before time.Time, stockCode string) (*models.StockNewsSummaryEntity, error)
}

type stockNewsSummaryRepository struct {
	db *gorm.DB
}

func NewStockNewsSummaryRepository(db *gorm.DB) StockNewsSummaryRepository {
	return &stockNewsSummaryRepository{
		db: db,
	}
}

func (r *stockNewsSummaryRepository) GetLast(before time.Time, stockCode string) (*models.StockNewsSummaryEntity, error) {
	var summary models.StockNewsSummaryEntity
	result := r.db.Where("created_at >= ? AND stock_code = ?", before, stockCode).Order("created_at desc").First(&summary)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}

		return nil, result.Error
	}
	return &summary, nil
}
