package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	//"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Completed   bool      `json:"completed"`
}

type TaskManager struct {
	mu            sync.Mutex
	tasks         map[int]Task
	nextID        int
	tasksAdded    prometheus.Counter
	tasksUpdated  prometheus.Counter
	taskDuration  prometheus.Histogram
	taskDescLength prometheus.Histogram
	updateDuration prometheus.Histogram
}

func NewTaskManager() *TaskManager {
	tm := &TaskManager{
		tasks:  make(map[int]Task),
		nextID: 1,
		tasksAdded: promauto.NewCounter(prometheus.CounterOpts{
			Name: "todoapp_tasks_added_total",
			Help: "Total number of tasks added",
		}),
		tasksUpdated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "todoapp_tasks_updated_total",
			Help: "Total number of tasks updated",
		}),
		taskDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "todoapp_add_task_duration_seconds",
			Help:    "Time taken to add a task",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1},
		}),
		taskDescLength: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "todoapp_task_desc_length_bytes",
			Help:    "Length of task descriptions",
			Buckets: []float64{10, 50, 100, 500, 1000},
		}),
		updateDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "todoapp_update_task_duration_seconds",
			Help:    "Time taken to update a task",
			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1},
		}),
	}

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
	tm.tasks[id] = Task{
		ID:          id,
		Description: desc,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Completed:   false,
	}
	tm.nextID++

	tm.tasksAdded.Inc()
	tm.taskDescLength.Observe(float64(len(desc)))
	return id, nil
}

func (tm *TaskManager) UpdateTask(id int, desc *string, completed *bool) (*Task, error) {
	start := time.Now()
	defer func() {
		tm.updateDuration.Observe(time.Since(start).Seconds())
	}()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		return nil, fmt.Errorf("задача не найдена")
	}

	if desc != nil {
		if *desc == "" {
			return nil, fmt.Errorf("описание не может быть пустым")
		}
		if len(*desc) > 1000 {
			return nil, fmt.Errorf("описание не может превышать 1000 символов")
		}
		task.Description = *desc
	}

	if completed != nil {
		task.Completed = *completed
	}

	task.UpdatedAt = time.Now()
	tm.tasks[id] = task
	tm.tasksUpdated.Inc()

	return &task, nil
}

func addTaskHandler(tm *TaskManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Description string `json:"description"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}
		defer r.Body.Close()

		id, err := tm.AddTask(req.Description)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(tm.tasks[id])
	}
}

func updateTaskHandler(tm *TaskManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid task ID"})
			return
		}

		var req struct {
			Description *string `json:"description,omitempty"`
			Completed   *bool   `json:"completed,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}
		defer r.Body.Close()

		updatedTask, err := tm.UpdateTask(id, req.Description, req.Completed)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(updatedTask)
	}
}

func main() {
	tm := NewTaskManager()
	r := chi.NewRouter()

	// Маршруты API
	r.Post("/tasks", addTaskHandler(tm))
	r.Patch("/tasks/{id}", updateTaskHandler(tm))
	
	// Метрики Prometheus
	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:    ":2112",
		Handler: r,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Server started on :2112")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	go demoOperations(tm)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Println("Server stopped")
}

func demoOperations(tm *TaskManager) {
	// Тестовые операции
	time.Sleep(500 * time.Millisecond) // Даем серверу время запуститься
	
	// Добавление задач
	id, err := tm.AddTask("Купить молоко")
	if err != nil {
		log.Printf("❌ Ошибка добавления: %v", err)
	} else {
		log.Printf("✅ Добавлена задача ID: %d", id)
	}

	// Обновление задачи
	time.Sleep(1 * time.Second)
	newDesc := "Купить молоко и хлеб"
	_, err = tm.UpdateTask(id, &newDesc, nil)
	if err != nil {
		log.Printf("❌ Ошибка обновления: %v", err)
	} else {
		log.Printf("🔄 Задача %d обновлена", id)
	}

	// Попытка обновления несуществующей задачи
	_, err = tm.UpdateTask(999, nil, nil)
	if err != nil {
		log.Printf("✅ Проверка несуществующей задачи: %v", err)
	}
}
