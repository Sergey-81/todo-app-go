package manager

import (
	"errors"
	"sync"
	"time"
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
	if description == "" {
		return 0, errors.New("описание задачи обязательно")
	}

    if len(description) > 1000 {
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
	return task.ID, nil
}