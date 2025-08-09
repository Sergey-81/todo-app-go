package manager

import (
	"strings"
	"sync"
	"testing"
	//"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestNewTaskManager(t *testing.T) {
	tm := NewTaskManager()
	if tm == nil {
		t.Fatal("NewTaskManager() вернул nil")
	}
	if tm.nextID != 1 {
		t.Errorf("Ожидался nextID=1, получен %d", tm.nextID)
	}
	if len(tm.tasks) != 0 {
		t.Errorf("Ожидался пустой список задач, получено %d задач", len(tm.tasks))
	}
}

func TestAddTask(t *testing.T) {
	tm := NewTaskManager()

	t.Run("Успешное добавление задачи", func(t *testing.T) {
		id, err := tm.AddTask("Новая задача")
		if err != nil {
			t.Fatalf("Ошибка при добавлении задачи: %v", err)
		}
		if id != 1 {
			t.Errorf("Ожидался ID=1, получен %d", id)
		}
	})

	t.Run("Пустое описание задачи", func(t *testing.T) {
		_, err := tm.AddTask("")
		if err == nil {
			t.Error("Ожидалась ошибка при пустом описании")
		}
	})

	t.Run("Слишком длинное описание", func(t *testing.T) {
		longDesc := strings.Repeat("a", 1001)
		_, err := tm.AddTask(longDesc)
		if err == nil {
			t.Error("Ожидалась ошибка при слишком длинном описании")
		}
	})
}

func TestUpdateTask(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Исходная задача")

	t.Run("Обновление только описания", func(t *testing.T) {
		newDesc := "Новое описание"
		updated, err := tm.UpdateTask(id, UpdateTaskRequest{Description: &newDesc})
		if err != nil {
			t.Fatalf("Ошибка при обновлении: %v", err)
		}
		if updated.Description != newDesc {
			t.Errorf("Описание не обновилось, ожидалось '%s', получено '%s'", newDesc, updated.Description)
		}
	})

	t.Run("Обновление только статуса", func(t *testing.T) {
		completed := true
		updated, err := tm.UpdateTask(id, UpdateTaskRequest{Completed: &completed})
		if err != nil {
			t.Fatalf("Ошибка при обновлении: %v", err)
		}
		if !updated.Completed {
			t.Error("Статус должен был измениться на завершенный")
		}
	})

	t.Run("Пустое описание", func(t *testing.T) {
		empty := ""
		_, err := tm.UpdateTask(id, UpdateTaskRequest{Description: &empty})
		if err == nil {
			t.Error("Ожидалась ошибка при пустом описании")
		}
	})

	t.Run("Слишком длинное описание", func(t *testing.T) {
		longDesc := strings.Repeat("a", 1001)
		_, err := tm.UpdateTask(id, UpdateTaskRequest{Description: &longDesc})
		if err == nil {
			t.Error("Ожидалась ошибка при слишком длинном описании")
		}
	})

	t.Run("Несуществующая задача", func(t *testing.T) {
		_, err := tm.UpdateTask(999, UpdateTaskRequest{})
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего ID")
		}
	})
}

func TestDeleteTask(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Задача для удаления")

	t.Run("Успешное удаление", func(t *testing.T) {
		err := tm.DeleteTask(id)
		if err != nil {
			t.Fatalf("Ошибка при удалении: %v", err)
		}
	})

	t.Run("Удаление несуществующей задачи", func(t *testing.T) {
		err := tm.DeleteTask(999)
		if err == nil {
			t.Error("Ожидалась ошибка при удалении несуществующей задачи")
		}
	})
}

func TestGetTask(t *testing.T) {
	tm := NewTaskManager()
	id, _ := tm.AddTask("Тестовая задача")

	t.Run("Получение существующей задачи", func(t *testing.T) {
		task, err := tm.GetTask(id)
		if err != nil {
			t.Fatalf("Ошибка при получении задачи: %v", err)
		}
		if task.ID != id {
			t.Errorf("Ожидался ID %d, получен %d", id, task.ID)
		}
		if task.Description != "Тестовая задача" {
			t.Errorf("Ожидалось описание 'Тестовая задача', получено '%s'", task.Description)
		}
	})

	t.Run("Получение несуществующей задачи", func(t *testing.T) {
		_, err := tm.GetTask(999)
		if err == nil {
			t.Error("Ожидалась ошибка при получении несуществующей задачи")
		}
	})
}

func TestGetAllTasks(t *testing.T) {
	tm := NewTaskManager()

	t.Run("Пустой список задач", func(t *testing.T) {
		tasks := tm.GetAllTasks()
		if len(tasks) != 0 {
			t.Errorf("Ожидался пустой список, получено %d задач", len(tasks))
		}
	})

	t.Run("Список с задачами", func(t *testing.T) {
		tm.AddTask("Задача 1")
		tm.AddTask("Задача 2")
		tasks := tm.GetAllTasks()
		if len(tasks) != 2 {
			t.Errorf("Ожидалось 2 задачи, получено %d", len(tasks))
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	tm := NewTaskManager()
	var wg sync.WaitGroup
	count := 100

	wg.Add(count)
	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()
			_, _ = tm.AddTask("Конкурентная задача")
		}()
	}
	wg.Wait()

	if len(tm.tasks) != count {
		t.Errorf("Ожидалось %d задач, получено %d", count, len(tm.tasks))
	}
}

func TestMetrics(t *testing.T) {
	// Сбрасываем счетчики
	AddTaskCount.Reset()
	UpdateTaskCount.Reset()
	DeleteTaskCount.Reset()

	tm := NewTaskManager()

	t.Run("Метрики AddTask", func(t *testing.T) {
		// Успешное добавление
		_, err := tm.AddTask("Тестовая задача")
		if err != nil {
			t.Fatalf("Ошибка при добавлении задачи: %v", err)
		}

		// Проверяем счетчик успешных операций
		if got := testutil.ToFloat64(AddTaskCount.WithLabelValues("success")); got != 1 {
			t.Errorf("AddTaskCount success = %v, want 1", got)
		}

		// Неудачное добавление
		_, err = tm.AddTask("")
		if err == nil {
			t.Error("Ожидалась ошибка при пустом описании")
		}

		// Проверяем счетчик ошибок
		if got := testutil.ToFloat64(AddTaskCount.WithLabelValues("error")); got != 1 {
			t.Errorf("AddTaskCount error = %v, want 1", got)
		}
	})

	t.Run("Метрики UpdateTask", func(t *testing.T) {
		id, _ := tm.AddTask("Тестовая задача")

		// Успешное обновление
		completed := true
		_, err := tm.UpdateTask(id, UpdateTaskRequest{Completed: &completed})
		if err != nil {
			t.Fatalf("Ошибка при обновлении задачи: %v", err)
		}

		// Проверяем счетчик успешных операций
		if got := testutil.ToFloat64(UpdateTaskCount.WithLabelValues("success")); got != 1 {
			t.Errorf("UpdateTaskCount success = %v, want 1", got)
		}

		// Неудачное обновление
		_, err = tm.UpdateTask(999, UpdateTaskRequest{})
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего ID")
		}

		// Проверяем счетчик ошибок
		if got := testutil.ToFloat64(UpdateTaskCount.WithLabelValues("error")); got != 1 {
			t.Errorf("UpdateTaskCount error = %v, want 1", got)
		}
	})

	t.Run("Метрики DeleteTask", func(t *testing.T) {
		id, _ := tm.AddTask("Тестовая задача")

		// Успешное удаление
		err := tm.DeleteTask(id)
		if err != nil {
			t.Fatalf("Ошибка при удалении задачи: %v", err)
		}

		// Проверяем счетчик успешных операций
		if got := testutil.ToFloat64(DeleteTaskCount.WithLabelValues("success")); got != 1 {
			t.Errorf("DeleteTaskCount success = %v, want 1", got)
		}

		// Неудачное удаление
		err = tm.DeleteTask(999)
		if err == nil {
			t.Error("Ожидалась ошибка для несуществующего ID")
		}

		// Проверяем счетчик ошибок
		if got := testutil.ToFloat64(DeleteTaskCount.WithLabelValues("error")); got != 1 {
			t.Errorf("DeleteTaskCount error = %v, want 1", got)
		}
	})
}