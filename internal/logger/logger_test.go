package logger

import "testing"

func TestLogger(t *testing.T) {
    t.Run("Info", func(t *testing.T) {
        Info("Тестовое сообщение")
    })

    t.Run("Error", func(t *testing.T) {
        Error(nil)
    })
}