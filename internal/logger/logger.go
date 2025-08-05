package logger

import "log"

func Info(msg string) {
    log.Printf("[INFO] %s", msg)
}

func Error(err error) {
    if err != nil {
        log.Printf("[ERROR] %v", err)
    }
}