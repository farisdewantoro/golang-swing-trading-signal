package models

import "time"

type RedisLastPrice struct {
	StockCode string    `json:"stock_code"`
	Price     float64   `json:"price"`
	Timestamp int64     `json:"timestamp"`
	Time      time.Time `json:"time"`
}
