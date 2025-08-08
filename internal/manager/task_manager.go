package manager

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

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

type UpdateTaskRequest struct {
	Description *string `json:"description,omitempty"`
	Completed   *bool   `json:"completed,omitempty"`
}

type Task struct {
	ID          int       `json:"id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Completed   bool      `json:"completed"`
}

type TaskManager struct {
	tasks  []Task
	lastID int
	mu     sync.Mutex
}

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

	tm.lastID++
	task := Task{
		ID:          tm.lastID,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Completed:   false,
	}

	tm.tasks = append(tm.tasks, task)
	
	addTaskCount.WithLabelValues("success").Inc()
	taskDescLength.Observe(float64(len(description)))
	
	return task.ID, nil
}

func (tm *TaskManager) UpdateTask(id int, req UpdateTaskRequest) (*Task, error) {
	startTime := time.Now()
	defer func() {
		updateTaskDuration.Observe(time.Since(startTime).Seconds())
	}()

	tm.mu.Lock()
	defer tm.mu.Unlock()

	for i := range tm.tasks {
		if tm.tasks[i].ID == id {
			if req.Description != nil {
				if *req.Description == "" {
					updateTaskCount.WithLabelValues("error").Inc()
					return nil, errors.New("описание не может быть пустым")
				}
				if len(*req.Description) > 1000 {
					updateTaskCount.WithLabelValues("error").Inc()
					return nil, errors.New("описание не может превышать 1000 символов")
				}
				tm.tasks[i].Description = *req.Description
			}

			if req.Completed != nil {
				tm.tasks[i].Completed = *req.Completed
			}

			tm.tasks[i].UpdatedAt = time.Now()
			updateTaskCount.WithLabelValues("success").Inc()
			return &tm.tasks[i], nil
		}
	}

	updateTaskCount.WithLabelValues("error").Inc()
	return nil, fmt.Errorf("задача с ID %d не найдена", id)
}

func (tm *TaskManager) GetTask(id int) (*Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	for _, task := range tm.tasks {
		if task.ID == id {
			return &task, nil
		}
	}
	return nil, fmt.Errorf("задача не найдена")
}