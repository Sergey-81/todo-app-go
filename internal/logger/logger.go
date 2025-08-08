package logger

import (
	"context"
	"log"
	//"os"
)

// Уровни логирования
const (
	LevelDebug = iota
	LevelInfo
	LevelError
)

var (
	logLevel = LevelInfo
)

// SetLevel устанавливает уровень логирования
func SetLevel(level int) {
	logLevel = level
}

// Debug логирует отладочные сообщения
func Debug(ctx context.Context, msg string, args ...interface{}) {
	if logLevel <= LevelDebug {
		log.Printf("[DEBUG] "+msg, args...)
	}
}

// Info логирует информационные сообщения
func Info(ctx context.Context, msg string, args ...interface{}) {
	if logLevel <= LevelInfo {
		log.Printf("[INFO] "+msg, args...)
	}
}

// Error логирует ошибки
func Error(ctx context.Context, err error, msg string, args ...interface{}) {
	if logLevel <= LevelError {
		if err != nil {
			log.Printf("[ERROR] "+msg+": %v", append(args, err)...)
		} else {
			log.Printf("[ERROR] "+msg, args...)
		}
	}
}