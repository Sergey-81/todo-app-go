package main

import (
	"fmt"
	"todo-app/internal/manager"
)

func main() {
	tm := &manager.TaskManager{}

	id, _ := tm.AddTask("Проверка работы")
	fmt.Printf("Добавлена задача с ID: %d\n", id)

    // Тест на валидацию
    _, err := tm.AddTask(strings.Repeat("a", 1001))
    if err != nil {
        fmt.Println("✅ Валидация длины работает:", err)
    }
}
