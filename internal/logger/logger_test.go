package logger

import (
	"errors" // Добавлен недостающий импорт
	"testing"
)

func TestLogger(t *testing.T) {
	t.Run("Info", func(t *testing.T) {
		Info("Тестовое сообщение")
	})

	t.Run("Error", func(t *testing.T) {
		err := errors.New("тестовая ошибка") // Теперь errors доступен
		Error(err)
	})
}

func TestDebug(t *testing.T) {
    t.Run("Debug", func(t *testing.T) {
        Debug("Тестовое debug-сообщение")
    })
}

func TestError(t *testing.T) {
    t.Run("With real error", func(t *testing.T) {
        err := errors.New("test error")
        Error(err) // Должно логировать ошибку
    })
}