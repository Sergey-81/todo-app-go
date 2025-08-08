package manager

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"todo-app/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Task представляет структуру задачи
type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Completed   bool      `json:"completed"`
}

// UpdateTaskRequest содержит поля для обновления задачи
type UpdateTaskRequest struct {
	Description *string `json:"description,omitempty"`
	Completed   *bool   `json:"completed,omitempty"`
}

var (
	addTaskCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "todoapp_tasks_added_total",
			Help: "Total number of AddTask operations",
		},
		[]string{"status"},
	)

	updateTaskCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "todoapp_tasks_updated_total",
			Help: "Total number of UpdateTask operations",
		},
		[]string{"status"},
	)

	taskDescLength = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "todoapp_task_desc_length_bytes",
			Help:    "Length distribution of task descriptions",
			Buckets: []float64{50, 100, 500, 1000},
		},
	)

	addTaskDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "todoapp_add_task_duration_seconds",
			Help:    "Duration of AddTask operation in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	updateTaskDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "todoapp_update_task_duration_seconds",
			Help:    "Duration of UpdateTask operation in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)

// TaskManager управляет задачами
type TaskManager struct {
	mu     sync.Mutex
	tasks  map[int]Task
	nextID int
}

// NewTaskManager создает новый менеджер задач
func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks:  make(map[int]Task),
		nextID: 1,
	}
}

// AddTask добавляет новую задачу
func (tm *TaskManager) AddTask(description string) (int, error) {
	startTime := time.Now()
	defer func() {
		addTaskDuration.Observe(time.Since(startTime).Seconds())
	}()

	if description == "" {
		addTaskCount.WithLabelValues("error").Inc()
		return 0, errors.New("описание задачи обязательно")
	}

	if len(description) > 1000 {
		addTaskCount.WithLabelValues("error").Inc()
		return 0, errors.New("описание не может превышать 1000 символов")
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	id := tm.nextID
	tm.tasks[id] = Task{
		ID:          id,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Completed:   false,
	}
	tm.nextID++

	addTaskCount.WithLabelValues("success").Inc()
	taskDescLength.Observe(float64(len(description)))
	
	logger.Info(context.Background(), "Задача добавлена", "taskID", id)
	return id, nil
}

// UpdateTask обновляет существующую задачу
func (tm *TaskManager) UpdateTask(id int, req UpdateTaskRequest) (*Task, error) {
	startTime := time.Now()
	defer func() {
		updateTaskDuration.Observe(time.Since(startTime).Seconds())
	}()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		updateTaskCount.WithLabelValues("error").Inc()
		logger.Error(context.Background(), fmt.Errorf("задача не найдена"), "UpdateTask failed", "taskID", id)
		return nil, fmt.Errorf("задача с ID %d не найдена", id)
	}

	if req.Description != nil {
		if *req.Description == "" {
			updateTaskCount.WithLabelValues("error").Inc()
			return nil, errors.New("описание не может быть пустым")
		}
		if len(*req.Description) > 1000 {
			updateTaskCount.WithLabelValues("error").Inc()
			return nil, errors.New("описание не может превышать 1000 символов")
		}
		task.Description = *req.Description
	}

	if req.Completed != nil {
		task.Completed = *req.Completed
	}

	task.UpdatedAt = time.Now()
	tm.tasks[id] = task
	
	updateTaskCount.WithLabelValues("success").Inc()
	logger.Info(context.Background(), "Задача обновлена", "taskID", id)
	return &task, nil
}

// GetTask возвращает задачу по ID
func (tm *TaskManager) GetTask(id int) (*Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		return nil, fmt.Errorf("задача не найдена")
	}
	return &task, nil
}