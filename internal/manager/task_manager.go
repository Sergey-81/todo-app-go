package manager

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
	"todo-app/internal/models"
)

type TaskManager struct {
	mu     sync.Mutex
	tasks  map[int]models.Task
	nextID int
}

func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks:  make(map[int]models.Task),
		nextID: 1,
	}
}

func (tm *TaskManager) AddTask(description string, tags []string) (int, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if description == "" {
		return 0, errors.New("описание задачи обязательно")
	}

	id := tm.nextID
	tm.tasks[id] = models.Task{
		ID:          id,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Completed:   false,
		Priority:    models.PriorityMedium,
		Tags:        normalizeTags(tags),
	}
	tm.nextID++

	return id, nil
}

func (tm *TaskManager) UpdateTask(id int, req models.UpdateTaskRequest) (*models.Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		return nil, fmt.Errorf("задача с ID %d не найдена", id)
	}

	if req.Description != nil {
		if *req.Description == "" {
			return nil, errors.New("описание не может быть пустым")
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

	return &task, nil
}

func (tm *TaskManager) DeleteTask(id int) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.tasks[id]; !exists {
		return fmt.Errorf("задача с ID %d не найдена", id)
	}

	delete(tm.tasks, id)
	return nil
}

func (tm *TaskManager) GetTask(id int) (*models.Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		return nil, fmt.Errorf("задача не найдена")
	}
	return &task, nil
}

func (tm *TaskManager) GetAllTasks() []models.Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tasks := make([]models.Task, 0, len(tm.tasks))
	for _, task := range tm.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

func (tm *TaskManager) ToggleComplete(id int) (*models.Task, error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task, exists := tm.tasks[id]
	if !exists {
		return nil, fmt.Errorf("задача с ID %d не найдена", id)
	}

	task.Completed = !task.Completed
	task.UpdatedAt = time.Now()
	tm.tasks[id] = task

	return &task, nil
}

func (tm *TaskManager) FilterTasks(completed *bool) []models.Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tasks := make([]models.Task, 0)
	for _, task := range tm.tasks {
		if completed == nil || task.Completed == *completed {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (tm *TaskManager) FilterByPriority(priority models.Priority) []models.Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tasks := make([]models.Task, 0)
	for _, task := range tm.tasks {
		if task.Priority == priority {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (tm *TaskManager) FilterByTag(tag string) []models.Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tag = strings.TrimSpace(strings.ToLower(tag))
	var result []models.Task

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

func (tm *TaskManager) GetUpcomingTasks(days int) []models.Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endDate := today.AddDate(0, 0, days+1)

	tasks := make([]models.Task, 0)
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