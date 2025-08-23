package storage

import (
	"sync"
	"time"
	"todo-app/internal/manager"
)

// Storage интерфейс для абстракции хранилища
type Storage interface {
	// Tasks
	AddTask(description string, tags []string) (int, error)
	GetAllTasks() ([]manager.Task, error)
	GetTask(id int) (*manager.Task, error)
	UpdateTask(id int, req manager.UpdateTaskRequest) (*manager.Task, error)
	DeleteTask(id int) error
	ToggleComplete(id int) (*manager.Task, error)
	FilterTasks(completed *bool) ([]manager.Task, error)

	// Subtasks
	AddSubTask(taskID int, description string) (int, error)
	GetSubTasks(taskID int) ([]manager.SubTask, error)
	ToggleSubTask(id int) error
	DeleteSubTask(id int) error

	// Закрытие соединения
	Close() error
}

// In-memory хранилище для обратной совместимости
type MemoryStorage struct {
	tasks     map[int]manager.Task
	subtasks  map[int]manager.SubTask
	nextID    int
	nextSubID int
	mu        sync.Mutex
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		tasks:    make(map[int]manager.Task),
		subtasks: make(map[int]manager.SubTask),
		nextID:   1,
		nextSubID: 1,
	}
}

func (m *MemoryStorage) AddTask(description string, tags []string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nextID
	m.tasks[id] = manager.Task{
		ID:          id,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Completed:   false,
		Priority:    manager.PriorityMedium,
		Tags:        tags,
	}
	m.nextID++

	return id, nil
}

func (m *MemoryStorage) GetAllTasks() ([]manager.Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks := make([]manager.Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// Реализуем остальные методы MemoryStorage (можно постепенно)...
// Пока оставляем заглушки для остальных методов

func (m *MemoryStorage) GetTask(id int) (*manager.Task, error) {
	// Заглушка
	return nil, nil
}

func (m *MemoryStorage) UpdateTask(id int, req manager.UpdateTaskRequest) (*manager.Task, error) {
	// Заглушка
	return nil, nil
}

func (m *MemoryStorage) DeleteTask(id int) error {
	// Заглушка
	return nil
}

func (m *MemoryStorage) ToggleComplete(id int) (*manager.Task, error) {
	// Заглушка
	return nil, nil
}

func (m *MemoryStorage) FilterTasks(completed *bool) ([]manager.Task, error) {
	// Заглушка
	return nil, nil
}

func (m *MemoryStorage) AddSubTask(taskID int, description string) (int, error) {
	// Заглушка
	return 0, nil
}

func (m *MemoryStorage) GetSubTasks(taskID int) ([]manager.SubTask, error) {
	// Заглушка
	return nil, nil
}

func (m *MemoryStorage) ToggleSubTask(id int) error {
	// Заглушка
	return nil
}

func (m *MemoryStorage) DeleteSubTask(id int) error {
	// Заглушка
	return nil
}

func (m *MemoryStorage) Close() error {
	// Заглушка
	return nil
}

// В MemoryStorage добавляем заглушки для новых методов
func (m *MemoryStorage) FilterByPriority(priority manager.Priority) ([]manager.Task, error) {
	// Заглушка
	return nil, nil
}

func (m *MemoryStorage) FilterByTag(tag string) ([]manager.Task, error) {
	// Заглушка  
	return nil, nil
}

func (m *MemoryStorage) GetUpcomingTasks(days int) ([]manager.Task, error) {
	// Заглушка
	return nil, nil
}

func (m *MemoryStorage) FilterByDateRange(start, end time.Time) ([]manager.Task, error) {
	// Заглушка
	return nil, nil
}

func (m *MemoryStorage) FilterTasksAdvanced(options manager.FilterOptions) ([]manager.Task, error) {
	// Заглушка
	return nil, nil
}