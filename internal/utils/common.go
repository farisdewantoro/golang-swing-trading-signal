package utils

import (
	"log"
	"runtime/debug"
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
