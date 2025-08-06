package models

// Существующая структура (не меняем)
type Task struct {
    ID          int
    Description string
    IsCompleted bool
}

// Новая структура только для HTTP-запроса
type CreateTaskRequest struct {
    Description string `json:"description"` // Важно: поле должно быть публичным (с большой буквы)
}