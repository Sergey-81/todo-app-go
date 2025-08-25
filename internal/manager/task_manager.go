package manager

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"log"

	"todo-app/internal/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

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
	UserID      int       `json:"user_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Completed   bool      `json:"completed"`
	Priority    Priority  `json:"priority"`
	DueDate     time.Time `json:"due_date"`
	Tags        []string  `json:"tags"`
}

type SubTask struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	TaskID      int       `json:"task_id"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Completed   bool      `json:"completed"`
}

type UpdateTaskRequest struct {
	Description *string    `json:"description,omitempty"`
	Completed   *bool      `json:"completed,omitempty"`
	Priority    *Priority  `json:"priority,omitempty"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Tags        *[]string  `json:"tags,omitempty"`
}

type TaskManager struct {
	mu     sync.Mutex
	tasks  map[int]Task
	nextID int
	storage Storage
}

type SubTaskManager struct {
	mu       sync.Mutex
	subtasks map[int]SubTask
	nextID   int
	storage  Storage
}

type FilterOptions struct {
	Completed   *bool      `json:"completed,omitempty"`
	Priority    *Priority  `json:"priority,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	HasDueDate  *bool      `json:"has_due_date,omitempty"`
}

type User struct {
    ID           int       `json:"id"`
    DeviceID     string    `json:"device_id"`
    TelegramID   int64     `json:"telegram_id,omitempty"`
    FCMToken     string    `json:"fcm_token,omitempty"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks:  make(map[int]Task),
		nextID: 1,
		storage: nil,
	}
}

func NewSubTaskManager() *SubTaskManager {
	return &SubTaskManager{
		subtasks: make(map[int]SubTask),
		nextID:   1,
		storage:  nil,
	}
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		lowerTag := strings.ToLower(tag)
		if tag != "" && !seen[lowerTag] {
			seen[lowerTag] = true
			result = append(result, tag)
		}
	}
	return result
}

// AddTaskForUser - новый метод для добавления задач с указанием пользователя
func (tm *TaskManager) AddTaskForUser(userID int, description string, tags []string) (int, error) {
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

	if tm.storage != nil {
		log.Printf("📦 Используем SQLite хранилище для задачи пользователя %d: %s", userID, description)
		id, err := tm.storage.AddTaskForUser(userID, description, tags)
		if err != nil {
			log.Printf("❌ Ошибка добавления в хранилище: %v", err)
			AddTaskCount.WithLabelValues("error").Inc()
			return 0, err
		}
		log.Printf("✅ Задача #%d добавлена в SQLite хранилище для пользователя %d", id, userID)
		TaskDescLength.Observe(float64(len(description)))
		AddTaskCount.WithLabelValues("success").Inc()
		logger.Info(context.Background(), "Задача добавлена в хранилище", "taskID", id, "userID", userID, "tags", tags)
		return id, nil
	}

	log.Printf("💾 Используем in-memory хранилище для задачи пользователя %d: %s", userID, description)
	id := tm.nextID
	tm.tasks[id] = Task{
		ID:          id,
		UserID:      userID,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Completed:   false,
		Priority:    PriorityMedium,
		Tags:        normalizeTags(tags),
	}
	tm.nextID++
	log.Printf("✅ Задача #%d добавлена в память для пользователя %d", id, userID)
	TaskDescLength.Observe(float64(len(description)))
	AddTaskCount.WithLabelValues("success").Inc()
	logger.Info(context.Background(), "Задача добавлена в память", "taskID", id, "userID", userID, "tags", tags)
	return id, nil
}

func (tm *TaskManager) AddTask(description string, tags []string) (int, error) {
	// Для обратной совместимости - используем user_id = 1
	return tm.AddTaskForUser(1, description, tags)
}

func (tm *TaskManager) UpdateTask(id int, req UpdateTaskRequest) (*Task, error) {
	start := time.Now()
	defer func() {
		UpdateTaskDuration.Observe(time.Since(start).Seconds())
	}()
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	if tm.storage != nil {
		log.Printf("📦 Используем хранилище для обновления задачи #%d", id)
		task, err := tm.storage.UpdateTask(id, req)
		if err != nil {
			UpdateTaskCount.WithLabelValues("error").Inc()
			return nil, err
		}
		UpdateTaskCount.WithLabelValues("success").Inc()
		logger.Info(context.Background(), "Задача обновлена в хранилище", "taskID", id, "tags", task.Tags)
		return task, nil
	}
	
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
	
	if req.Tags != nil {
		task.Tags = normalizeTags(*req.Tags)
	}
	
	task.UpdatedAt = time.Now()
	tm.tasks[id] = task
	UpdateTaskCount.WithLabelValues("success").Inc()
	logger.Info(context.Background(), "Задача обновлена", "taskID", id, "tags", task.Tags)
	return &task, nil
}

func (tm *TaskManager) DeleteTask(id int) error {
	start := time.Now()
	defer func() {
		DeleteTaskDuration.Observe(time.Since(start).Seconds())
	}()
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	if tm.storage != nil {
		log.Printf("📦 Используем хранилище для удаления задачи #%d", id)
		err := tm.storage.DeleteTask(id)
		if err != nil {
			DeleteTaskCount.WithLabelValues("error").Inc()
			return err
		}
		DeleteTaskCount.WithLabelValues("success").Inc()
		logger.Info(context.Background(), "Задача удалена из хранилище", "taskID", id)
		return nil
	}
	
	if _, exists := tm.tasks[id]; !exists {
		DeleteTaskCount.WithLabelValues("error").Inc()
		return fmt.Errorf("задача с ID %d не найдена", id)
	}
	delete(tm.tasks, id)
	DeleteTaskCount.WithLabelValues("success").Inc()
	logger.Info(context.Background(), "Задача удалена из памяти", "taskID", id)
	return nil
}

func (tm *TaskManager) GetAllTasks() []Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.storage != nil {
		log.Printf("📦 Загружаем задачи из SQLite хранилища")
		tasks, err := tm.storage.GetAllTasks()
		if err != nil {
			log.Printf("❌ Ошибка загрузки из хранилища: %v", err)
			return []Task{}
		}
		log.Printf("✅ Загружено %d задач из SQLite хранилища", len(tasks))
		return tasks
	}

	log.Printf("💾 Загружаем задачи из памяти")
	tasks := make([]Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}
	log.Printf("✅ Загружено %d задач из памяти", len(tasks))
	return tasks
}

func (tm *TaskManager) ToggleComplete(id int) (*Task, error) {
	start := time.Now()
	defer func() {
		UpdateTaskDuration.Observe(time.Since(start).Seconds())
	}()
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	if tm.storage != nil {
		log.Printf("📦 Используем хранилище для переключения задачи #%d", id)
		task, err := tm.storage.ToggleComplete(id)
		if err != nil {
			UpdateTaskCount.WithLabelValues("error").Inc()
			return nil, err
		}
		UpdateTaskCount.WithLabelValues("success").Inc()
		logger.Info(context.Background(), "Статус задачи изменен в хранилище", "taskID", id, "completed", task.Completed)
		return task, nil
	}
	
	task, exists := tm.tasks[id]
	if !exists {
		UpdateTaskCount.WithLabelValues("error").Inc()
		return nil, fmt.Errorf("задача с ID %d не найдена", id)
	}
	task.Completed = !task.Completed
	task.UpdatedAt = time.Now()
	tm.tasks[id] = task
	UpdateTaskCount.WithLabelValues("success").Inc()
	logger.Info(context.Background(), "Статус задачи изменен в памяти", "taskID", id, "completed", task.Completed)
	return &task, nil
}

func (tm *TaskManager) FilterTasks(completed *bool) []Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	if tm.storage != nil {
		log.Printf("📦 Используем хранилище для фильтрации задач")
		tasks, err := tm.storage.FilterTasks(completed)
		if err != nil {
			log.Printf("❌ Ошибка фильтрации в хранилище: %v", err)
			return []Task{}
		}
		log.Printf("✅ Отфильтровано %d задач из хранилища", len(tasks))
		return tasks
	}
	
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
    
    if tm.storage != nil {
        tasks, err := tm.storage.FilterByPriority(priority)
        if err != nil {
            log.Printf("❌ Ошибка фильтрации по приоритету: %v", err)
            return []Task{}
        }
        return tasks
    }

	tasks := make([]Task, 0)
	for _, task := range tm.tasks {
		if task.Priority == priority {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (tm *TaskManager) FilterByTag(tag string) []Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tag = strings.TrimSpace(strings.ToLower(tag))
	var result []Task
	
	for _, task := range tm.tasks {
		for _, t := range task.Tags {
			if strings.ToLower(t) == tag {
				result = append(result, task)
				break
			}
		}
	}
	return result
}

func (tm *TaskManager) GetAllTags() []string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	tagsMap := make(map[string]bool)
	for _, task := range tm.tasks {
		for _, tag := range task.Tags {
			normalized := strings.ToLower(strings.TrimSpace(tag))
			if normalized != "" {
				tagsMap[normalized] = true
			}
		}
	}
	
	tags := make([]string, 0, len(tagsMap))
	for tag := range tagsMap {
		tags = append(tags, tag)
	}
	
	sort.Strings(tags)
	return tags
}

func (tm *TaskManager) GetUpcomingTasks(days int) []Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endDate := today.AddDate(0, 0, days+1)
	tasks := make([]Task, 0)
	for _, task := range tm.tasks {
		if task.DueDate.IsZero() || task.Completed {
			continue
		}
		taskDate := time.Date(
			task.DueDate.Year(),
			task.DueDate.Month(),
			task.DueDate.Day(),
			0, 0, 0, 0,
			task.DueDate.Location(),
		)
		if taskDate.After(today.Add(-time.Nanosecond)) && taskDate.Before(endDate) {
			tasks = append(tasks, task)
		}
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].DueDate.Before(tasks[j].DueDate)
	})
	return tasks
}

func (tm *TaskManager) FilterByDateRange(start, end time.Time) []Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	
	var result []Task
	for _, task := range tm.tasks {
		if !task.DueDate.IsZero() && 
		   !task.DueDate.Before(start) && 
		   !task.DueDate.After(end) {
			result = append(result, task)
		}
	}
	
	sort.Slice(result, func(i, j int) bool {
		return result[i].DueDate.Before(result[j].DueDate)
	})
	
	return result
}

func (stm *SubTaskManager) AddSubTask(taskID int, description string) (int, error) {
	if description == "" {
		return 0, errors.New("описание подзадачи обязательно")
	}
	
	stm.mu.Lock()
	defer stm.mu.Unlock()
	
	id := stm.nextID
	stm.subtasks[id] = SubTask{
		ID:          id,
		TaskID:      taskID,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Completed:   false,
	}
	stm.nextID++
	
	logger.Info(context.Background(), "Подзадача добавлена", "subtaskID", id, "taskID", taskID)
	return id, nil
}

func (stm *SubTaskManager) GetSubTasks(taskID int) []SubTask {
	stm.mu.Lock()
	defer stm.mu.Unlock()
	
	var result []SubTask
	for _, subtask := range stm.subtasks {
		if subtask.TaskID == taskID {
			result = append(result, subtask)
		}
	}
	
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	
	return result
}

func (stm *SubTaskManager) ToggleSubTask(id int) error {
	stm.mu.Lock()
	defer stm.mu.Unlock()
	
	subtask, exists := stm.subtasks[id]
	if !exists {
		return fmt.Errorf("подзадача с ID %d не найдена", id)
	}
	
	subtask.Completed = !subtask.Completed
	subtask.UpdatedAt = time.Now()
	stm.subtasks[id] = subtask
	
	logger.Info(context.Background(), "Статус подзадачи изменен", "subtaskID", id, "completed", subtask.Completed)
	return nil
}

func (stm *SubTaskManager) DeleteSubTask(id int) error {
	stm.mu.Lock()
	defer stm.mu.Unlock()
	
	if _, exists := stm.subtasks[id]; !exists {
		return fmt.Errorf("подзадача с ID %d не найдена", id)
	}
	
	delete(stm.subtasks, id)
	logger.Info(context.Background(), "Подзадача удалена", "subtaskID", id)
	return nil
}

func (tm *TaskManager) FilterTasksAdvanced(options FilterOptions) []Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	 if tm.storage != nil {
        log.Printf("📦 Используем хранилище для расширенной фильтрации")
        tasks, err := tm.storage.FilterTasksAdvanced(options)
        if err != nil {
            log.Printf("❌ Ошибка расширенной фильтрации в хранилище: %v", err)
            return []Task{}
        }
        log.Printf("✅ Отфильтровано %d задач расширенным фильтром", len(tasks))
        return tasks
    }
	tasks := make([]Task, 0)
	
	for _, task := range tm.tasks {
		if options.Completed != nil && task.Completed != *options.Completed {
			continue
		}
		
		if options.Priority != nil && task.Priority != *options.Priority {
			continue
		}
		
		if len(options.Tags) > 0 {
			hasMatchingTag := false
			for _, filterTag := range options.Tags {
				filterTag = strings.TrimSpace(strings.ToLower(filterTag))
				for _, taskTag := range task.Tags {
					if strings.ToLower(taskTag) == filterTag {
						hasMatchingTag = true
						break
					}
				}
				if hasMatchingTag {
					break
				}
			}
			if !hasMatchingTag {
				continue
			}
		}
		
		if options.HasDueDate != nil {
			hasDueDate := !task.DueDate.IsZero()
			if hasDueDate != *options.HasDueDate {
				continue
			}
		}
		
		if options.StartDate != nil || options.EndDate != nil {
			if task.DueDate.IsZero() {
				continue
			}
			
			if options.StartDate != nil && task.DueDate.Before(*options.StartDate) {
				continue
			}
			if options.EndDate != nil && task.DueDate.After(*options.EndDate) {
				continue
			}
		}
		
		tasks = append(tasks, task)
	}
	
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].DueDate.IsZero() && !tasks[j].DueDate.IsZero() {
			return false
		}
		if !tasks[i].DueDate.IsZero() && tasks[j].DueDate.IsZero() {
			return true
		}
		if tasks[i].DueDate.IsZero() && tasks[j].DueDate.IsZero() {
			return tasks[i].ID < tasks[j].ID
		}
		return tasks[i].DueDate.Before(tasks[j].DueDate)
	})
	
	return tasks
}

func NewTaskManagerWithStorage(storage Storage) *TaskManager {
	return &TaskManager{
		tasks:  make(map[int]Task),
		nextID: 1,
		storage: storage,
	}
}

func NewSubTaskManagerWithStorage(storage Storage) *SubTaskManager {
	return &SubTaskManager{
		subtasks: make(map[int]SubTask),
		nextID:   1,
		storage:  storage,
	}
}

func (tm *TaskManager) GetStorage() Storage {
	return tm.storage
}

// 🆕 Добавляем метод для получения задач пользователя
func (tm *TaskManager) GetAllTasksForUser(userID int) ([]Task, error) {
    if tm.storage != nil {
        return tm.storage.GetAllTasksForUser(userID)
    }
    
    tm.mu.Lock()
    defer tm.mu.Unlock()
    
    var userTasks []Task
    for _, task := range tm.tasks {
        if task.UserID == userID {
            userTasks = append(userTasks, task)
        }
    }
    return userTasks, nil
}

type Storage interface {
	AddTask(description string, tags []string) (int, error)
	AddTaskForUser(userID int, description string, tags []string) (int, error)
	GetAllTasks() ([]Task, error)
	GetTask(id int) (*Task, error)
	UpdateTask(id int, req UpdateTaskRequest) (*Task, error)
	DeleteTask(id int) error
	ToggleComplete(id int) (*Task, error)
	
	FilterTasks(completed *bool) ([]Task, error)
	FilterByPriority(priority Priority) ([]Task, error)
	FilterByTag(tag string) ([]Task, error)
	GetUpcomingTasks(days int) ([]Task, error)
	FilterByDateRange(start, end time.Time) ([]Task, error)
	FilterTasksAdvanced(options FilterOptions) ([]Task, error)

	AddSubTask(taskID int, description string) (int, error)
	GetSubTasks(taskID int) ([]SubTask, error)
	ToggleSubTask(id int) error
	DeleteSubTask(id int) error

    CreateUser(user *User) (int, error)
    GetUserByDeviceID(deviceID string) (*User, error)
    GetUserByTelegramID(telegramID int64) (*User, error)
	GetUserByID(userID int) (*User, error)
    UpdateUser(user *User) error

    GetAllTasksForUser(userID int) ([]Task, error)
    
    MigrateExistingTasksToUser(userID int, deviceID string) error

	Close() error
}
