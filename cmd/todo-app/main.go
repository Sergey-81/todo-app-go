package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TaskManager с метриками Prometheus
type TaskManager struct {
	mu            sync.Mutex
	tasks         map[int]string
	nextID        int
	tasksAdded    prometheus.Counter
	taskDuration  prometheus.Histogram
	taskDescLength prometheus.Histogram
}

func NewTaskManager() *TaskManager {
	tm := &TaskManager{
		tasks:  make(map[int]string),
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

	if desc == "" {
		return 0, fmt.Errorf("описание задачи обязательно")
	}
	if len(desc) > 1000 {
		return 0, fmt.Errorf("описание не может превышать 1000 символов")
	}

	id := tm.nextID
	tm.tasks[id] = desc
	tm.nextID++

	tm.tasksAdded.Inc()
	tm.taskDescLength.Observe(float64(len(desc)))
	return id, nil
}

func addTaskHandler(tm *TaskManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Description string `json:"description"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if _, err := tm.AddTask(req.Description); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func main() {
	tm := NewTaskManager()
	r := chi.NewRouter()

	r.Post("/tasks", addTaskHandler(tm))
	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":2112",
		Handler: r,
	}

	go func() {
		log.Println("Server started on :2112")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	go demoOperations(tm)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Server stopped")
}

func demoOperations(tm *TaskManager) {
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
		taskID, taskErr := tm.AddTask(fmt.Sprintf("Задача %d", i+1))
		duration := time.Since(start)
		
		if taskErr != nil {
			log.Printf("⚠️ Ошибка при добавлении задачи %d: %v", i+1, taskErr)
		} else {
			log.Printf("➕ Добавлена задача %d (время: %v)", taskID, duration)
		}
	}
}
