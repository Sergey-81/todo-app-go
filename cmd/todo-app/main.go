package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TaskManager с метриками Prometheus
type TaskManager struct {
	mu    sync.Mutex
	tasks map[int]string
	nextID int

	// Метрики Prometheus
	tasksAdded      prometheus.Counter
	taskDuration    prometheus.Histogram
	taskDescLength  prometheus.Histogram
}

// NewTaskManager создает менеджер задач с инициализированными метриками
func NewTaskManager() *TaskManager {
	tm := &TaskManager{
		tasks: make(map[int]string),
		nextID: 1,
		tasksAdded: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "todoapp_tasks_added_total",
			Help: "Total number of tasks added",
		}),
		taskDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "todoapp_add_task_duration_seconds",
			Help:    "Time taken to add a task",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1},
		}),
		taskDescLength: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "todoapp_task_desc_length_bytes",
			Help:    "Length of task descriptions",
			Buckets: []float64{10, 50, 100, 500, 1000},
		}),
	}

	// Регистрация метрик
	prometheus.MustRegister(tm.tasksAdded)
	prometheus.MustRegister(tm.taskDuration)
	prometheus.MustRegister(tm.taskDescLength)

	return tm
}

func (tm *TaskManager) AddTask(desc string) (int, error) {
	start := time.Now()
	defer func() {
		tm.taskDuration.Observe(time.Since(start).Seconds())
	}()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// Валидация
	if desc == "" {
		return 0, fmt.Errorf("описание задачи обязательно")
	}
	if len(desc) > 1000 {
		return 0, fmt.Errorf("описание не может превышать 1000 символов")
	}

	// Логика добавления
	id := tm.nextID
	tm.tasks[id] = desc
	tm.nextID++

	// Обновление метрик
	tm.tasksAdded.Inc()
	tm.taskDescLength.Observe(float64(len(desc)))

	return id, nil
}

func main() {
    // 1. Инициализация менеджера задач с метриками
    tm := NewTaskManager()

    // 2. Настройка HTTP-сервера для метрик
    reg := prometheus.NewRegistry()
    reg.MustRegister(tm.tasksAdded, tm.taskDuration, tm.taskDescLength)

    mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

    srv := &http.Server{
        Addr:    ":2112",
        Handler: mux,
    }

    // 3. Запуск сервера в отдельной goroutine
    go func() {
        log.Println("Starting metrics server on :2112")
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Metrics server failed: %v", err)
        }
    }()

    // 4. Даем серверу время на запуск
    time.Sleep(500 * time.Millisecond)

    // 5. Демонстрационные операции
    demoOperations(tm)

    // 6. Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server shutdown failed: %v", err)
    }
    log.Println("Server gracefully stopped")
}

func demoOperations(tm *TaskManager) {
    // Тест 1: Успешное добавление задачи
    id, err := tm.AddTask("Купить молоко")  // Первое использование := (объявление)
    if err != nil {
        log.Printf("❌ Ошибка добавления: %v", err)
    } else {
        log.Printf("✅ Добавлена задача ID: %d", id)
    }

    // Тест 2: Пустое описание - используем = вместо :=
    _, err = tm.AddTask("")  // Присваивание существующей err
    if err != nil {
        log.Printf("✅ Валидация пустого описания работает: %v", err)
    }

    // Тест 3: Длинное описание - также используем =
    longDesc := strings.Repeat("a", 1001)
    _, err = tm.AddTask(longDesc)  // Присваивание существующей err
    if err != nil {
        log.Printf("✅ Валидация длины описания работает: %v", err)
    }

    // Тест 4: Множественные задачи
    for i := 0; i < 5; i++ {
        start := time.Now()
        taskID, taskErr := tm.AddTask(fmt.Sprintf("Задача %d", i+1))  // Новые переменные
        duration := time.Since(start)
        
        if taskErr != nil {
            log.Printf("⚠️ Ошибка при добавлении задачи %d: %v", i+1, taskErr)
        } else {
            log.Printf("➕ Добавлена задача %d (время: %v)", taskID, duration)
        }
    }
}
