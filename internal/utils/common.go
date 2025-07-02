package utils

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/sirupsen/logrus"
)

func ToPointer[T any](value T) *T {
	return &value
}

func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[SafeGo] recovered from panic: %v\n%s", r, debug.Stack())
			}
		}()
		fn()
	}()
}

func ShouldStopChan(stop <-chan struct{}, log *logrus.Logger) bool {
	select {
	case <-stop:
		// Dapatkan nama fungsi caller
		pc, _, _, ok := runtime.Caller(1)
		funcName := "unknown"
		if ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				// Ambil hanya nama fungsi (tanpa path lengkap)
				parts := strings.Split(fn.Name(), "/")
				funcName = parts[len(parts)-1]
			}
		}

		log.Debug("Stop signal received",
			logrus.Fields{
				"caller": funcName,
			},
		)
		return true
	default:
		return false
	}
}

func ShouldStopCtx(ctx context.Context, log *logrus.Logger) (bool, error) {
	select {
	case <-ctx.Done():
		// Dapatkan nama fungsi caller
		pc, _, _, ok := runtime.Caller(1)
		funcName := "unknown"
		if ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				// Ambil hanya nama fungsi (tanpa path lengkap)
				parts := strings.Split(fn.Name(), "/")
				funcName = parts[len(parts)-1]
			}
		}

		log.Debug("Context done signal received",
			logrus.Fields{
				"caller": funcName,
				"error":  ctx.Err(),
			},
		)
		return true, ctx.Err()
	default:
		return false, nil
	}
}

func FormatPercentage(value float64) string {
	return fmt.Sprintf("%+.1f%%", value)
}
