package manager

import "todo-app/internal/models"

type TaskManager struct {
    tasks []models.Task
}

func New() *TaskManager {
    return &TaskManager{tasks: make([]models.Task, 0)}
}