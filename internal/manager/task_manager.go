package manager

import (
	"errors"
	"sync"
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Метрики Prometheus
var (
	addTaskCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "todoapp_tasks_added_total",
			Help: "Total number of AddTask operations",
		},
		[]string{"status"}, // success/error
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
)

type Task struct {
	ID          int
	Description string
	CreatedAt   time.Time
	Completed   bool
}

type TaskManager struct {
	tasks  []Task
	lastID int
	mu     sync.Mutex
}

func (tm *TaskManager) AddTask(description string) (int, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		addTaskDuration.Observe(duration)
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
		Completed:   false,
	}

	tm.tasks = append(tm.tasks, task)
	
	addTaskCount.WithLabelValues("success").Inc()
	taskDescLength.Observe(float64(len(description)))
	
	return task.ID, nil
}