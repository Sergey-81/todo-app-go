package main

import (
	"context"
	"fmt"
	"log"
	//"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"todo-app/internal/manager"
)

func main() {
	// Инициализация менеджера задач
	tm := &manager.TaskManager{}

	// Запуск HTTP-сервера для метрик Prometheus
	srv := startMetricsServer()

	// Демонстрация работы
	demoOperations(tm)

	// Ожидание сигнала завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %v", err)
	}
	log.Println("Server gracefully stopped")
}

func startMetricsServer() *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":2112",
		Handler: mux,
	}

	go func() {
		log.Println("Starting metrics server on :2112")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	return srv
}

func demoOperations(tm *manager.TaskManager) {
	// Тест 1: Успешное добавление задачи
	id, err := tm.AddTask("Купить молоко")
	if err != nil {
		log.Printf("❌ Ошибка добавления: %v", err)
	} else {
		log.Printf("✅ Добавлена задача ID: %d", id)
	}

	// Тест 2: Пустое описание
	_, err = tm.AddTask("")
	if err != nil {
		log.Printf("✅ Валидация пустого описания работает: %v", err)
	}

	// Тест 3: Длинное описание
	longDesc := strings.Repeat("a", 1001)
	_, err = tm.AddTask(longDesc)
	if err != nil {
		log.Printf("✅ Валидация длины описания работает: %v", err)
	}

	// Тест 4: Множественные задачи
	for i := 0; i < 5; i++ {
		start := time.Now()
		id, err := tm.AddTask(fmt.Sprintf("Задача %d", i+1))
		duration := time.Since(start)
		
		if err != nil {
			log.Printf("⚠️ Ошибка при добавлении задачи %d: %v", i+1, err)
		} else {
			log.Printf("➕ Добавлена задача %d (время: %v)", id, duration)
		}
	}
}
