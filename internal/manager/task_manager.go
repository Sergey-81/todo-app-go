package manager

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"todo-app/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Метрики уровня пакета
var (
	AddTaskCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "todoapp_tasks_added_total",
			Help: "Total number of AddTask operations",
		},
		[]string{"status"},
	)

	UpdateTaskCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "todoapp_tasks_updated_total",
			Help: "Total number of UpdateTask operations",
		},
		[]string{"status"},
	)

	DeleteTaskCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "todoapp_tasks_deleted_total",
			Help: "Total number of DeleteTask operations",
		},
		[]string{"status"},
	)

	TaskDescLength = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "todoapp_task_desc_length_bytes",
			Help:    "Length distribution of task descriptions",
			Buckets: []float64{50, 100, 500, 1000},
		},
	)

	AddTaskDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "todoapp_add_task_duration_seconds",
			Help:    "Duration of AddTask operation in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	UpdateTaskDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "todoapp_update_task_duration_seconds",
			Help:    "Duration of UpdateTask operation in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)

	DeleteTaskDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "todoapp_delete_task_duration_seconds",
			Help:    "Duration of DeleteTask operation in seconds",
			Buckets: prometheus.DefBuckets,
		},
	)
)

type Priority string

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Completed   bool      `json:"completed"`
	Priority    Priority  `json:"priority"`
	DueDate     time.Time `json:"due_date"`
}

type UpdateTaskRequest struct {
	Description *string    `json:"description,omitempty"`
	Completed   *bool      `json:"completed,omitempty"`
	Priority    *Priority  `json:"priority,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
}

type TaskManager struct {
	mu     sync.Mutex
	tasks  map[int]Task
	nextID int
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks:  make(map[int]Task),
		nextID: 1,
	}
}

func (tm *TaskManager) AddTask(description string) (int, error) {
	start := time.Now()
	defer func() {
		AddTaskDuration.Observe(time.Since(start).Seconds())
	}()

	if description == "" {
		AddTaskCount.WithLabelValues("error").Inc()
		return 0, errors.New("описание задачи обязательно")
	}

	if len(description) > 1000 {
		AddTaskCount.WithLabelValues("error").Inc()
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
		Priority:    PriorityMedium,
	}
	tm.nextID++

	TaskDescLength.Observe(float64(len(description)))
	AddTaskCount.WithLabelValues("success").Inc()
	logger.Info(context.Background(), "Задача добавлена", "taskID", id)
	return id, nil
}

func (tm *TaskManager) UpdateTask(id int, req UpdateTaskRequest) (*Task, error) {
	start := time.Now()
	defer func() {
		UpdateTaskDuration.Observe(time.Since(start).Seconds())
	}()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		UpdateTaskCount.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("задача с ID %d не найдена", id)
	}

	if req.Description != nil {
		if *req.Description == "" {
			UpdateTaskCount.WithLabelValues("error").Inc()
			return nil, errors.New("описание не может быть пустым")
		}
		if len(*req.Description) > 1000 {
			UpdateTaskCount.WithLabelValues("error").Inc()
			return nil, errors.New("описание не может превышать 1000 символов")
		}
		task.Description = *req.Description
	}

	if req.Completed != nil {
		task.Completed = *req.Completed
	}

	if req.Priority != nil {
		task.Priority = *req.Priority
	}

	if req.DueDate != nil {
		task.DueDate = *req.DueDate
	}

	task.UpdatedAt = time.Now()
	tm.tasks[id] = task

	UpdateTaskCount.WithLabelValues("success").Inc()
	logger.Info(context.Background(), "Задача обновлена", "taskID", id)
	return &task, nil
}

func (tm *TaskManager) DeleteTask(id int) error {
	start := time.Now()
	defer func() {
		DeleteTaskDuration.Observe(time.Since(start).Seconds())
	}()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tasks[id]; !exists {
		DeleteTaskCount.WithLabelValues("error").Inc()
		return fmt.Errorf("задача с ID %d не найдена", id)
	}

	delete(tm.tasks, id)
	DeleteTaskCount.WithLabelValues("success").Inc()
	logger.Info(context.Background(), "Задача удалена", "taskID", id)
	return nil
}

func (tm *TaskManager) GetTask(id int) (*Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		return nil, fmt.Errorf("задача не найдена")
	}
	return &task, nil
}

func (tm *TaskManager) GetAllTasks() []Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tasks := make([]Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (tm *TaskManager) ToggleComplete(id int) (*Task, error) {
	start := time.Now()
	defer func() {
		UpdateTaskDuration.Observe(time.Since(start).Seconds())
	}()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		UpdateTaskCount.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("задача с ID %d не найдена", id)
	}

	task.Completed = !task.Completed
	task.UpdatedAt = time.Now()
	tm.tasks[id] = task

	UpdateTaskCount.WithLabelValues("success").Inc()
	logger.Info(context.Background(), "Статус задачи изменен", 
		"taskID", id, "completed", task.Completed)
	return &task, nil
}

func (tm *TaskManager) FilterTasks(completed *bool) []Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tasks := make([]Task, 0)
	for _, task := range tm.tasks {
		if completed == nil || task.Completed == *completed {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (tm *TaskManager) FilterByPriority(priority Priority) []Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tasks := make([]Task, 0)
	for _, task := range tm.tasks {
		if task.Priority == priority {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (tm *TaskManager) GetUpcomingTasks(days int) []Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	limit := now.AddDate(0, 0, days)
	tasks := make([]Task, 0)
	
	for _, task := range tm.tasks {
		if !task.DueDate.IsZero() && 
		   task.DueDate.After(now) && 
		   task.DueDate.Before(limit) && 
		   !task.Completed {
			tasks = append(tasks, task)
		}
	}
	
	// Сортируем по дате выполнения (ближайшие сначала)
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].DueDate.Before(tasks[j].DueDate)
	})
	
	return tasks
}