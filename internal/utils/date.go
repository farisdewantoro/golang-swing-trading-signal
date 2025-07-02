package utils

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"time"
)

func TimeNowWIB() time.Time {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		log.Fatal("Failed to load location", err)
	}
	return time.Now().In(loc)
}

func GetNowWithOnlyHour() time.Time {
	now := TimeNowWIB()
	return time.Date(
		now.Year(),
		now.Month(),
		now.Day(),
		now.Hour(),
		0, 0, 0,
		now.Location(),
	)
}

func PrettyDate(date time.Time) string {
	return fmt.Sprintf("🗓️ %02d %s %d - %02d:%02d WIB",
		date.Day(),
		GetIndonesianMonth(date.Month()),
		date.Year(),
		date.Hour(),
		date.Minute(),
	)
}

func PrettyDateWithIcon(date time.Time) string {
	return fmt.Sprintf("%02d %s %d - %02d:%02d WIB",
		date.Day(),
		GetIndonesianMonth(date.Month()),
		date.Year(),
		date.Hour(),
		date.Minute(),
	)
}

func GetIndonesianMonth(month time.Month) string {
	months := map[time.Month]string{
		time.January:   "Januari",
		time.February:  "Februari",
		time.March:     "Maret",
		time.April:     "April",
		time.May:       "Mei",
		time.June:      "Juni",
		time.July:      "Juli",
		time.August:    "Agustus",
		time.September: "September",
		time.October:   "Oktober",
		time.November:  "November",
		time.December:  "Desember",
	}
	return months[month]
}

func MustParseDate(strTime string) time.Time {
	date, _ := time.Parse("2006-01-02", strTime)

	return date
}

// GetTimeBefore menerima string seperti "1d", "3m", "1h" dan mengembalikan time sebelum x
func GetTimeBefore(input string) (time.Time, error) {
	if len(input) < 2 {
		return time.Time{}, fmt.Errorf("invalid input")
	}

	unit := input[len(input)-1:]  // ambil 1 huruf terakhir (d, m, h)
	value := input[:len(input)-1] // ambil angka di depannya
	num, err := strconv.Atoi(value)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid number: %w", err)
	}

	now := TimeNowWIB()

	switch unit {
	case "d":
		return now.AddDate(0, 0, -num), nil // kurangi hari
	case "m":
		return now.AddDate(0, -num, 0), nil // kurangi bulan
	case "h":
		return now.Add(-time.Duration(num) * time.Hour), nil // kurangi jam
	default:
		return time.Time{}, fmt.Errorf("unsupported unit: %s", unit)
	}
}

func RemainingDays(maxHoldingDays int, buyTime time.Time) int {
	// Hitung waktu expired
	expiredTime := buyTime.AddDate(0, 0, maxHoldingDays)

	// Hitung selisih hari dari sekarang
	now := TimeNowWIB()
	remaining := int(math.Ceil(expiredTime.Sub(now).Hours() / 24))

	return remaining
}
