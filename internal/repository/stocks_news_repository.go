package repository

import (
	"context"
	"fmt"
	"log"
	"strings"

	"golang-swing-trading-signal/internal/models"

	"gorm.io/gorm"
)

type StocksNewsRepository interface {
	GetTopNews(ctx context.Context, param models.StockNewsQueryParam) ([]models.StockNewsEntity, error)
}

type stocksNewsRepository struct {
	db *gorm.DB
}

func NewStocksNewsRepository(db *gorm.DB) StocksNewsRepository {
	return &stocksNewsRepository{db: db}
}

func (s *stocksNewsRepository) GetTopNews(ctx context.Context, param models.StockNewsQueryParam) ([]models.StockNewsEntity, error) {
	var (
		qBuilder strings.Builder
		qParam   = []interface{}{}
	)

	qBuilder.WriteString(fmt.Sprintf(`
	SELECT
		sn.id,
		sn.title,
		sn.link,
		sn.published_at,
		sn.raw_content,
		sn.summary,
		sn.hash_identifier,
		sn.source,
		sn.google_rss_link,
		sn.impact_score,
		sn.key_issue,
		sn.created_at,
		sn.updated_at,
		sm.stock_code,
		sm.sentiment,
		sm.impact,
		sm.confidence_score,
		sm.reason,
		(0.5 * sm.confidence_score + 0.3 * sn.impact_score + 0.2 * GREATEST(0, 1 - (EXTRACT(EPOCH FROM (NOW() - sn.published_at)) / 86400) / %d)) AS final_score
	FROM stock_news AS sn
	JOIN stock_mentions AS sm ON sm.stock_news_id = sn.id
	WHERE sm.stock_code IN ?
	AND sn.published_at >= NOW() - INTERVAL '%d days'
`, param.MaxNewsAgeInDays, param.MaxNewsAgeInDays))

	qParam = append(qParam, param.StockCodes)
	if len(param.PriorityDomains) > 0 {
		qBuilder.WriteString(" ORDER BY CASE WHEN sn.source IN ? THEN 0 ELSE 1 END, final_score DESC")
		qParam = append(qParam, param.PriorityDomains)
	} else {
		qBuilder.WriteString(" ORDER BY final_score DESC")
	}
	qBuilder.WriteString(" LIMIT ?")
	qParam = append(qParam, param.Limit)

	type scanResult struct {
		models.StockNewsEntity
		StockCode       string  `gorm:"column:stock_code"`
		Sentiment       string  `gorm:"column:sentiment"`
		Impact          string  `gorm:"column:impact"`
		ConfidenceScore float64 `gorm:"column:confidence_score"`
		FinalScore      float64 `gorm:"column:final_score"`
		Reason          string  `gorm:"column:reason"`
	}

	var results []scanResult
	err := s.db.Debug().WithContext(ctx).Raw(qBuilder.String(), qParam...).Scan(&results).Error
	if err != nil {
		log.Fatal("Query error: ", err)
	}

	news := make([]models.StockNewsEntity, len(results))
	for i, r := range results {
		news[i] = r.StockNewsEntity
		news[i].StockCode = r.StockCode
		news[i].Sentiment = r.Sentiment
		news[i].Impact = r.Impact
		news[i].ConfidenceScore = r.ConfidenceScore
		news[i].FinalScore = r.FinalScore
		news[i].Reason = r.Reason
	}

	return news, nil
}
