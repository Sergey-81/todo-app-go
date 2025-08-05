package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"todo-app/internal/manager"
)

func main() {
	// Инициализация менеджера задач
	tm := &manager.TaskManager{}

	// Запуск HTTP-сервера для метрик Prometheus
	startMetricsServer()

	// Демонстрация работы
	demoOperations(tm)

	// Бесконечный цикл для работы сервера метрик
	select {}
}

func startMetricsServer() {
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		if err := http.ListenAndServe(":2112", nil); err != nil {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()
	log.Println("Metrics server started at :2112")
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
