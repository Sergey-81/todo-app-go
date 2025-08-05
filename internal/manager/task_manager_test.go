package manager

import (
	
	"strings" // Добавляем этот импорт
	"testing"
)

func TestAddTask(t *testing.T) {
	tm := &TaskManager{}

	id, err := tm.AddTask("Купить молоко")
	if err != nil {
		t.Fatalf("Ошибка при добавлении задачи: %v", err)
	}

	if id != 1 {
		t.Errorf("Ожидался ID=1, получено %d", id)
	}

	if len(tm.tasks) != 1 {
		t.Errorf("Ожидалась 1 задача, получено %d", len(tm.tasks))
	}
}

func TestAddEmptyTask(t *testing.T) {
	tm := &TaskManager{}

	_, err := tm.AddTask("")
	if err == nil {
		t.Error("Ожидалась ошибка при пустом описании")
	}
}

func TestAddTaskWithMaxLength(t *testing.T) {
    tm := &TaskManager{}
    
    // Генерируем строку длиной ровно 1000 символов
    validDesc := strings.Repeat("a", 1000)
    _, err := tm.AddTask(validDesc)
    if err != nil {
        t.Errorf("Ожидалась успешная валидация для 1000 символов: %v", err)
    }
    
    // Тест на 1001 символ
    invalidDesc := validDesc + "a"
    _, err = tm.AddTask(invalidDesc)
    if err == nil {
        t.Error("Ожидалась ошибка при 1001 символе")
    }
}