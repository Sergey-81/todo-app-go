package logger

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
)

func TestLogger(t *testing.T) {
	// Сохраняем оригинальный output
	oldOutput := log.Writer()
	defer log.SetOutput(oldOutput)

	// Перехватываем вывод
	var buf bytes.Buffer
	log.SetOutput(&buf)

	ctx := context.Background()

	t.Run("Info", func(t *testing.T) {
		buf.Reset()
		Info(ctx, "Тестовое сообщение")
		if !strings.Contains(buf.String(), "[INFO] Тестовое сообщение") {
			t.Errorf("Неверный формат лога Info: %s", buf.String())
		}
	})

	t.Run("Error with error", func(t *testing.T) {
		buf.Reset()
		err := errors.New("тестовая ошибка")
		Error(ctx, err, "Дополнительное сообщение")
		if !strings.Contains(buf.String(), "[ERROR] Дополнительное сообщение: тестовая ошибка") {
			t.Errorf("Неверный формат лога Error: %s", buf.String())
		}
	})

	t.Run("Error without error", func(t *testing.T) {
		buf.Reset()
		Error(ctx, nil, "Сообщение без ошибки")
		if !strings.Contains(buf.String(), "[ERROR] Сообщение без ошибки") {
			t.Errorf("Неверный формат лога Error без ошибки: %s", buf.String())
		}
	})

	t.Run("Debug with level", func(t *testing.T) {
		buf.Reset()
		SetLevel(LevelDebug)
		defer SetLevel(LevelInfo)

		Debug(ctx, "Тестовое debug-сообщение")
		if !strings.Contains(buf.String(), "[DEBUG] Тестовое debug-сообщение") {
			t.Errorf("Неверный формат лога Debug: %s", buf.String())
		}
	})

	t.Run("Debug without level", func(t *testing.T) {
		buf.Reset()
		SetLevel(LevelInfo)

		Debug(ctx, "Это не должно логироваться")
		if buf.String() != "" {
			t.Errorf("Debug сообщение не должно логироваться при LevelInfo: %s", buf.String())
		}
	})
}

func TestLoggerWithFields(t *testing.T) {
	// Перехватываем вывод
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	ctx := context.Background()

	t.Run("Info with fields", func(t *testing.T) {
		buf.Reset()
		Info(ctx, "Сообщение с полями", "key1", "value1", "key2", 42)
		output := buf.String()
		if !strings.Contains(output, "[INFO] Сообщение с полями") ||
			!strings.Contains(output, "key1") ||
			!strings.Contains(output, "value1") ||
			!strings.Contains(output, "42") {
			t.Errorf("Неверный формат лога с полями: %s", output)
		}
	})
}