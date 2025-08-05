package logger

import "log"

func Info(msg string) {
    log.Printf("[INFO] %s", msg)
}

func Error(err error) {
    if err != nil { // Добавьте эту проверку
        log.Printf("[ERROR] %v", err)
    }
}

func Debug(msg string) {
    log.Printf("[DEBUG] %s", msg)
}