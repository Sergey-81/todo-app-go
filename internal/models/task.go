package models

import "time"
// Существующая структура (не меняем)
type Task struct {
    ID          int
    Description string
    IsCompleted bool
    DueDate   time.Time
}

// Новая структура только для HTTP-запроса
type CreateTaskRequest struct {
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"` // omitempty - поле необязательное
}